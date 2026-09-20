// Package handlers provides HTTP request handlers for the medicaments API endpoints.
// This file implements the HTTPHandler interface with dependency injection.
package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

const (
	maxMedicamentSearchResults = 250
	maxGeneriqueSearchResults  = 100

	maxPageSize     = 200
	defaultPageSize = 10

	errTooManyMedicamentsResults = "Search too broad. Maximum 250 results returned. Use more specific search terms or /export for full dataset"
	errTooManyGeneriquesResults  = "Search too broad. Maximum 100 results returned. Use more specific search terms or /export for full dataset"
)

// Handler implements the interfaces.HTTPHandler interface
type Handler struct {
	dataStore     interfaces.DataStore
	validator     interfaces.DataValidator
	healthChecker interfaces.HealthChecker
}

// NewHTTPHandler creates a new HTTP handler with injected dependencies
func NewHTTPHandler(dataStore interfaces.DataStore, validator interfaces.DataValidator, healthChecker interfaces.HealthChecker) interfaces.HTTPHandler {
	return &Handler{
		dataStore:     dataStore,
		validator:     validator,
		healthChecker: healthChecker,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// This is a placeholder - the actual routing is handled by chi
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// HealthResponseImpl defines the structure for consistent JSON ordering
type HealthResponseImpl struct {
	Status string         `json:"status"`
	Data   map[string]any `json:"data"`
}

// RespondWithJSON writes a JSON response with compression optimization
func (h *Handler) RespondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		logging.Error("Failed to marshal JSON response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Last-Modified", h.dataStore.GetLastUpdated().UTC().Format(http.TimeFormat))
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		logging.Error("Failed to write response", "error", err)
	}
}

// RespondWithError writes a JSON error response
func (h *Handler) RespondWithError(w http.ResponseWriter, code int, message string) {
	errorResponse := map[string]any{
		"error":   http.StatusText(code),
		"message": message,
		"code":    code,
	}
	h.RespondWithJSON(w, code, errorResponse)
}

// GenerateETag creates an ETag from data using SHA256 hash
func GenerateETag(data []byte) string {
	hash := sha256.Sum256(data)
	// Use first 8 bytes of hash for shorter ETag
	return fmt.Sprintf(`W/"%x"`, hash[:8])
}

// CheckETag validates If-None-Match header against provided ETag
func CheckETag(r *http.Request, etag string) bool {
	clientETag := strings.TrimPrefix(r.Header.Get("If-None-Match"), "W/")
	serverETag := strings.TrimPrefix(etag, "W/")
	return clientETag != "" && clientETag == serverETag
}

// findMedicamentByCIP searches for a medicament by CIP7 or CIP13
// Returns (medicament, true) if found, (nil, false) if not found
func (h *Handler) findMedicamentByCIP(cip int) (*entities.Medicament, bool) {
	medicamentsMap := h.dataStore.GetMedicamentsMap()

	// Search in CIP7 map first (O(1) lookup)
	presentationsCIP7 := h.dataStore.GetPresentationsCIP7Map()
	if pres, ok := presentationsCIP7[cip]; ok {
		if med, exists := medicamentsMap[pres.Cis]; exists {
			return &med, true
		}
	}

	// If not found, try CIP13 map (O(1) lookup)
	presentationsCIP13 := h.dataStore.GetPresentationsCIP13Map()
	if pres, ok := presentationsCIP13[cip]; ok {
		if med, exists := medicamentsMap[pres.Cis]; exists {
			return &med, true
		}
	}

	return nil, false
}

// RespondWithJSONAndETag writes a JSON response with ETag and cache validation
func (h *Handler) RespondWithJSONAndETag(w http.ResponseWriter, r *http.Request, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		logging.Error("Failed to marshal JSON response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	etag := GenerateETag(data)

	// Check if client has cached version
	if CheckETag(r, etag) && code == http.StatusOK {
		// Add cache headers to 304 response as well
		w.Header().Set("ETag", etag)
		w.Header().Set("Last-Modified", h.dataStore.GetLastUpdated().UTC().Format(http.TimeFormat))
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour cache
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", h.dataStore.GetLastUpdated().UTC().Format(http.TimeFormat))
	w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour cache
	w.WriteHeader(code)

	if _, err := w.Write(data); err != nil {
		logging.Error("Failed to write response", "error", err)
	}
}

// legacySunsetDate is the sunset date (RFC 8594) after which the legacy
// endpoints were permanently removed and started responding 410 Gone.
const legacySunsetDate = "2026-07-31T23:59:59Z"

// RespondWithGone writes a 410 Gone response for a removed legacy endpoint,
// pointing clients to its v1 successor via standard deprecation headers.
func (h *Handler) RespondWithGone(w http.ResponseWriter, r *http.Request, successorPath string) {
	oldPath := r.URL.Path

	// Primary deprecation header (RFC 9745)
	w.Header().Set("Deprecation", "true")

	// Sunset header indicates when the endpoint was removed (RFC 8594)
	w.Header().Set("Sunset", legacySunsetDate)

	// Link header points to the replacement endpoint (RFC 5988 Web Linking)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, successorPath)
	w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"successor-version\"", fullURL))

	// Non standard but good practice
	w.Header().Set("X-Deprecated", fmt.Sprintf("Use %s instead", successorPath))

	// Warning header (HTTP/1.1 standard - RFC 7234)
	// Format: 299 - "warning-text"
	warningMsg := fmt.Sprintf("299 - \"Removed endpoint %s. Use %s instead\"", oldPath, successorPath)
	w.Header().Set("Warning", warningMsg)

	h.RespondWithError(w, http.StatusGone, fmt.Sprintf("Endpoint %s was removed on %s. Use %s instead", oldPath, legacySunsetDate, successorPath))
}

// prixSunsetDate is the enforcement date for the field-level sunset of the
// legacy "0 means absent" semantics of the prix field (announced in v2.2.0).
// At v3.0.0, prix becomes null when the source declares no price, like
// prixPublique and honorairesDispensation already do.
const prixSunsetDate = "2026-12-31"

// addPrixSunsetWarning attaches the RFC 7234 Warning header announcing the
// prix field-level sunset to responses that embed presentation data.
// Remove this helper and its call sites at v3.0.0 enforcement.
func addPrixSunsetWarning(w http.ResponseWriter) {
	w.Header().Set("Warning", fmt.Sprintf(
		"299 - \"Field 'prix' returns 0 when the source declares no price; it will return null after %s (see CHANGELOG)\"",
		prixSunsetDate))
}

// ExportMedicaments returns all medicaments.
// The legacy /database route was removed (sunset 2026-07-31) and now returns 410 Gone.
func (h *Handler) ExportMedicaments(w http.ResponseWriter, r *http.Request) {
	// Removed legacy route: respond 410 Gone before any processing
	if r.URL.Path == "/database" {
		h.RespondWithGone(w, r, "/v1/medicaments/export")
		return
	}

	medicaments := h.dataStore.GetMedicaments()
	addPrixSunsetWarning(w)
	h.RespondWithJSONAndETag(w, r, http.StatusOK, medicaments)
}

// ServePagedMedicaments responds 410 Gone for the removed legacy endpoint
// /database/{pageNumber}. Use /v1/medicaments?page={n}&pageSize={m} instead.
func (h *Handler) ServePagedMedicaments(w http.ResponseWriter, r *http.Request) {
	page := r.PathValue("pageNumber")
	if page == "" {
		page = "1"
	}
	h.RespondWithGone(w, r, fmt.Sprintf("/v1/medicaments?page=%s", page))
}

// FindMedicament responds 410 Gone for the removed legacy endpoint
// /medicament/{element}. Use /v1/medicaments?search={element} instead.
func (h *Handler) FindMedicament(w http.ResponseWriter, r *http.Request) {
	h.RespondWithGone(w, r, fmt.Sprintf("/v1/medicaments?search=%s", r.PathValue("element")))
}

// FindMedicamentByCIS finds a medicament by CIS.
// The legacy /medicament/id/{cis} route was removed (sunset 2026-07-31)
// and now returns 410 Gone.
func (h *Handler) FindMedicamentByCIS(w http.ResponseWriter, r *http.Request) {
	// Removed legacy route: respond 410 Gone before any processing
	if strings.HasPrefix(r.URL.Path, "/medicament/id/") {
		h.RespondWithGone(w, r, fmt.Sprintf("/v1/medicaments/%s", r.PathValue("cis")))
		return
	}

	cisStr := r.PathValue("cis")

	cis, err := h.validator.ValidateCIS(cisStr)

	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	medicamentsMap := h.dataStore.GetMedicamentsMap()
	med, exists := medicamentsMap[cis]
	if !exists {
		h.RespondWithError(w, http.StatusNotFound, "Medicament not found")
		return
	}

	addPrixSunsetWarning(w)
	h.RespondWithJSON(w, http.StatusOK, med)
}

// FindMedicamentByCIP responds 410 Gone for the removed legacy endpoint
// /medicament/cip/{cip}. Use /v1/medicaments?cip={cip} instead.
func (h *Handler) FindMedicamentByCIP(w http.ResponseWriter, r *http.Request) {
	h.RespondWithGone(w, r, fmt.Sprintf("/v1/medicaments?cip=%s", r.PathValue("cip")))
}

// FindGeneriques responds 410 Gone for the removed legacy endpoint
// /generiques/{libelle}. Use /v1/generiques?libelle={libelle} instead.
func (h *Handler) FindGeneriques(w http.ResponseWriter, r *http.Request) {
	h.RespondWithGone(w, r, fmt.Sprintf("/v1/generiques?libelle=%s", r.PathValue("libelle")))
}

// FindGeneriquesByGroupID finds generiques by group ID.
// The legacy /generiques/group/{groupId} route was removed (sunset 2026-07-31)
// and now returns 410 Gone.
func (h *Handler) FindGeneriquesByGroupID(w http.ResponseWriter, r *http.Request) {
	// Removed legacy route: respond 410 Gone before any processing
	if strings.HasPrefix(r.URL.Path, "/generiques/group/") {
		h.RespondWithGone(w, r, fmt.Sprintf("/v1/generiques/%s", r.PathValue("groupId")))
		return
	}

	groupIDStr := r.PathValue("groupID")

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	generiquesMap := h.dataStore.GetGeneriquesMap()
	gen, exists := generiquesMap[groupID]
	if !exists {
		h.RespondWithError(w, http.StatusNotFound, "Generique group not found")
		return
	}

	h.RespondWithJSONAndETag(w, r, http.StatusOK, gen)
}

// HealthCheck returns server health information
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status, data, httpStatus := h.healthChecker.HealthCheck()

	response := HealthResponseImpl{
		Status: status,
		Data:   data,
	}

	h.RespondWithJSON(w, httpStatus, response)
}

// DiagnosticsResponseImpl defines the structure for diagnostics endpoint
type DiagnosticsResponseImpl struct {
	Timestamp     string         `json:"timestamp"`
	UptimeSeconds float64        `json:"uptime_seconds"`
	NextUpdate    string         `json:"next_update"`
	DataAgeHours  float64        `json:"data_age_hours"`
	System        map[string]any `json:"system"`
	DataIntegrity map[string]any `json:"data_integrity"`
}

// ServeDiagnosticsV1 returns detailed system diagnostics including data integrity
func (h *Handler) ServeDiagnosticsV1(w http.ResponseWriter, r *http.Request) {
	// Get memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate uptime using actual server start time
	serverStartTime := h.dataStore.GetServerStartTime()
	var uptime time.Duration
	if serverStartTime.IsZero() {
		// Fallback if start time is not available
		uptime = 0
	} else {
		uptime = time.Since(serverStartTime)
	}

	// Get data statistics
	lastUpdate := h.dataStore.GetLastUpdated()
	dataAge := time.Since(lastUpdate)

	// Get cached data quality report (no recomputation)
	report := h.dataStore.GetDataQualityReport()

	// Build data integrity section with sample CIS (include all categories, even with zero count)
	dataIntegrity := map[string]any{
		"medicaments_without_conditions": map[string]any{
			"count":      report.MedicamentsWithoutConditions,
			"sample_cis": report.MedicamentsWithoutConditionsCIS,
		},
		"medicaments_without_generiques": map[string]any{
			"count":      report.MedicamentsWithoutGeneriques,
			"sample_cis": report.MedicamentsWithoutGeneriquesCIS,
		},
		"medicaments_without_presentations": map[string]any{
			"count":      report.MedicamentsWithoutPresentations,
			"sample_cis": report.MedicamentsWithoutPresentationsCIS,
		},
		"medicaments_without_compositions": map[string]any{
			"count":      report.MedicamentsWithoutCompositions,
			"sample_cis": report.MedicamentsWithoutCompositionsCIS, // All CIS stored here
		},
		"generique_only_cis": map[string]any{
			"count":      report.GeneriqueOnlyCIS,
			"sample_cis": report.GeneriqueOnlyCISList,
		},
		"presentations_with_orphaned_cis": map[string]any{
			"count":      report.PresentationsWithOrphanedCIS,
			"sample_cip": report.PresentationsWithOrphanedCISCIPList,
		},
	}

	response := DiagnosticsResponseImpl{
		Timestamp:     time.Now().Format(time.RFC3339),
		UptimeSeconds: uptime.Seconds(),
		NextUpdate:    h.healthChecker.CalculateNextUpdate().Format(time.RFC3339),
		DataAgeHours:  dataAge.Hours(),
		System: map[string]any{
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]any{
				"alloc_mb": m.Alloc / 1024 / 1024,
				"sys_mb":   m.Sys / 1024 / 1024,
				"num_gc":   m.NumGC,
			},
		},
		DataIntegrity: dataIntegrity,
	}

	// Add 10-second cache to prevent hammering while keeping data reasonably fresh
	w.Header().Set("Cache-Control", "public, max-age=10")
	h.RespondWithJSON(w, http.StatusOK, response)
}

// NEW v1 handlers

func (h *Handler) ServePresentationsV1(w http.ResponseWriter, r *http.Request) {
	cipStr := r.PathValue("cip")

	// Validate the CIP
	cip, err := h.validator.ValidateCIP(cipStr)
	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Search first in the CIP7
	presentationsCIP7 := h.dataStore.GetPresentationsCIP7Map()
	if pres, ok := presentationsCIP7[cip]; ok {
		addPrixSunsetWarning(w)
		h.RespondWithJSONAndETag(w, r, http.StatusOK, pres)
		return
	}

	// If not, in the CIP13
	presentationsCIP13 := h.dataStore.GetPresentationsCIP13Map()

	if pres, ok := presentationsCIP13[cip]; ok {
		addPrixSunsetWarning(w)
		h.RespondWithJSONAndETag(w, r, http.StatusOK, pres)
		return
	}

	// If not found, return error
	h.RespondWithError(w, http.StatusNotFound, "Presentation not found")
}

// ServePresentationsMissingCIP handles requests to /v1/presentations/ without a CIP path parameter.
// Returns a 400 Bad Request error indicating that the CIP path parameter is required.
func (h *Handler) ServePresentationsMissingCIP(w http.ResponseWriter, r *http.Request) {
	h.RespondWithError(w, http.StatusBadRequest, "CIP path parameter is required")
}

func (h *Handler) ServeGeneriquesV1(w http.ResponseWriter, r *http.Request) {
	libelle := r.URL.Query().Get("libelle")

	if libelle == "" {
		h.RespondWithError(w, http.StatusBadRequest, "Needs libelle param")
		return
	}

	// Validate user input using the validator
	if err := h.validator.ValidateInput(libelle); err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize input and convert to lowercase for case-insensitive search
	sanitizedLibelle := strings.ToLower(libelle)
	// Normalize: replace + with space for flexible matching
	sanitizedLibelle = strings.ReplaceAll(sanitizedLibelle, "+", " ")

	// Split search query into individual words for multi-word search
	searchWords := strings.Fields(sanitizedLibelle)

	generiques := h.dataStore.GetGeneriques()
	var results []entities.GeneriqueList

	for _, gen := range generiques {
		// Check if ALL search words exist in libelle (AND logic)
		allMatch := true
		for _, word := range searchWords {
			if !strings.Contains(gen.LibelleNormalized, word) {
				allMatch = false
				break
			}
		}

		if allMatch {
			results = append(results, gen)
			// Check if there are more results than the maximum, return error
			if len(results) > maxGeneriqueSearchResults {
				h.RespondWithError(w, http.StatusBadRequest, errTooManyGeneriquesResults)
				return

			}
		}
	}

	if len(results) != 0 {
		h.RespondWithJSONAndETag(w, r, http.StatusOK, results)
		return
	}
	h.RespondWithError(w, http.StatusNotFound, "No generiques found")
}

func (h *Handler) ServeMedicamentsV1(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	totalParams := 0
	for _, v := range []string{q.Get("cip"), q.Get("search"), q.Get("page")} {
		if v != "" {
			totalParams++
		}
	}

	if q.Get("pageSize") != "" && q.Get("page") == "" {
		h.RespondWithError(w, http.StatusBadRequest, "pageSize can only be used with page")
	}

	if totalParams == 0 {
		h.RespondWithError(w, http.StatusBadRequest, "Needs at least one param. See documentation")
		return
	}

	if totalParams > 1 {
		h.RespondWithError(w, http.StatusBadRequest, "Only one parameter allowed at a time. Choose: page, cip, search")
		return
	}

	// Paginated results
	if pageNumber := q.Get("page"); pageNumber != "" {
		page, err := strconv.Atoi(pageNumber)
		if err != nil || page < 1 {
			logging.Warn("Unusual user input", "pageNumber", pageNumber)
			h.RespondWithError(w, http.StatusBadRequest, "Invalid page number")
			return
		}

		pageSize := defaultPageSize

		if pageSizeStr := q.Get("pageSize"); pageSizeStr != "" {
			pageSize, err = strconv.Atoi(pageSizeStr)
			if err != nil || pageSize < 1 || pageSize > maxPageSize {
				msg := fmt.Sprintf("Invalid pageSize. Must be between 1 and %d", maxPageSize)
				h.RespondWithError(w, http.StatusBadRequest, msg)
				return
			}

		}

		medicaments := h.dataStore.GetMedicaments()

		start := (page - 1) * pageSize
		end := start + pageSize

		if start >= len(medicaments) {
			h.RespondWithError(w, http.StatusNotFound, "Page not found")
			return
		}

		if end > len(medicaments) {
			end = len(medicaments)
		}

		pagedMedicaments := medicaments[start:end]
		totalItems := len(medicaments)
		maxPage := (totalItems + pageSize - 1) / pageSize

		response := map[string]any{
			"data":       pagedMedicaments,
			"page":       page,
			"pageSize":   pageSize,
			"totalItems": totalItems,
			"maxPage":    maxPage,
		}

		addPrixSunsetWarning(w)
		h.RespondWithJSONAndETag(w, r, http.StatusOK, response)
		return
	}

	// Search query
	if searchQuery := q.Get("search"); searchQuery != "" {
		// Validate input using the validator
		if err := h.validator.ValidateInput(searchQuery); err != nil {
			h.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Sanitize and normalize input (replace + with space for flexible matching)
		sanitizedElement := strings.ToLower(searchQuery)
		sanitizedElement = strings.ReplaceAll(sanitizedElement, "+", " ")

		// Split search query into individual words for multi-word search
		searchWords := strings.Fields(sanitizedElement)

		medicaments := h.dataStore.GetMedicaments()
		var results []entities.Medicament

		for _, med := range medicaments {
			// Check if ALL search words exist in denomination (AND logic)
			allMatch := true
			for _, word := range searchWords {
				if !strings.Contains(med.DenominationNormalized, word) {
					allMatch = false
					break // Early termination - skip this medicament immediately
				}
			}
			if allMatch {
				results = append(results, med)
				if len(results) > maxMedicamentSearchResults {
					h.RespondWithError(w, http.StatusBadRequest, errTooManyMedicamentsResults)
					return
				}

			}
		}

		// Return 404 if no results found
		if len(results) == 0 {
			h.RespondWithError(w, http.StatusNotFound, "No medicaments found")
			return
		}

		addPrixSunsetWarning(w)
		h.RespondWithJSONAndETag(w, r, http.StatusOK, results)
		return
	}

	// Search medicament by CIP7 or CIP13
	if cipStr := q.Get("cip"); cipStr != "" {
		cip, err := h.validator.ValidateCIP(cipStr)
		if err != nil {
			h.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		med, found := h.findMedicamentByCIP(cip)
		if !found {
			h.RespondWithError(w, http.StatusNotFound, "Medicament not found")
			return
		}

		addPrixSunsetWarning(w)
		h.RespondWithJSONAndETag(w, r, http.StatusOK, med)
		return
	}

	h.RespondWithError(w, http.StatusBadRequest, "Unexpected error")

}
