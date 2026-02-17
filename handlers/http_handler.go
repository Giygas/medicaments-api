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

// HTTPHandlerImpl implements the interfaces.HTTPHandler interface
type HTTPHandlerImpl struct {
	dataStore     interfaces.DataStore
	validator     interfaces.DataValidator
	healthChecker interfaces.HealthChecker
}

// NewHTTPHandler creates a new HTTP handler with injected dependencies
func NewHTTPHandler(dataStore interfaces.DataStore, validator interfaces.DataValidator, healthChecker interfaces.HealthChecker) interfaces.HTTPHandler {
	return &HTTPHandlerImpl{
		dataStore:     dataStore,
		validator:     validator,
		healthChecker: healthChecker,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *HTTPHandlerImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// This is a placeholder - the actual routing is handled by chi
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// HealthResponseImpl defines the structure for consistent JSON ordering
type HealthResponseImpl struct {
	Status string         `json:"status"`
	Data   map[string]any `json:"data"`
}

// RespondWithJSON writes a JSON response with compression optimization
func (h *HTTPHandlerImpl) RespondWithJSON(w http.ResponseWriter, code int, payload any) {
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
func (h *HTTPHandlerImpl) RespondWithError(w http.ResponseWriter, code int, message string) {
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
	return fmt.Sprintf(`"%x"`, hash[:8])
}

// CheckETag validates If-None-Match header against provided ETag
func CheckETag(r *http.Request, etag string) bool {
	clientETag := r.Header.Get("If-None-Match")
	return clientETag == etag
}

// findMedicamentByCIP searches for a medicament by CIP7 or CIP13
// Returns (medicament, true) if found, (nil, false) if not found
func (h *HTTPHandlerImpl) findMedicamentByCIP(cip int) (*entities.Medicament, bool) {
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
func (h *HTTPHandlerImpl) RespondWithJSONAndETag(w http.ResponseWriter, r *http.Request, code int, payload any) {
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

func (h *HTTPHandlerImpl) AddDeprecationHeaders(w http.ResponseWriter, r *http.Request, newPath string) {
	oldPath := r.URL.Path
	// Primary deprecation header (HTTP/1.1 standard)
	w.Header().Set("Deprecation", "true")

	// Link header points to the replacement endpoint
	// Follows RFC 5988 Web Linking standard
	// Build full URL first
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, newPath)
	w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"successor-version\"", fullURL))

	// Sunset header indicates when the endpoint will be removed
	// Format: RFC 1123 date format
	w.Header().Set("Sunset", "2026-07-31T23:59:59Z")

	// Non standard but good practice
	w.Header().Set("X-Deprecated", fmt.Sprintf("Use %s instead", newPath))

	// Warning header (HTTP/1.1 standard - RFC 7234)
	// Format: 299 - "warning-text"
	// This is the HTTP standard way to warn clients about deprecated behavior
	warningMsg := fmt.Sprintf("299 - \"Deprecated endpoint %s. Use %s instead\"", oldPath, newPath)
	w.Header().Set("Warning", warningMsg)

}

// ExportMedicaments returns all medicaments
func (h *HTTPHandlerImpl) ExportMedicaments(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/database" {
		h.AddDeprecationHeaders(w, r, "/v1/medicaments/export")
	}

	medicaments := h.dataStore.GetMedicaments()
	h.RespondWithJSONAndETag(w, r, http.StatusOK, medicaments)
}

// ServePagedMedicaments returns paginated medicaments
func (h *HTTPHandlerImpl) ServePagedMedicaments(w http.ResponseWriter, r *http.Request) {
	pageNumber := r.PathValue("pageNumber")
	page, err := strconv.Atoi(pageNumber)
	if err != nil || page < 1 {
		logging.Warn("Unusual user input", "pageNumber", pageNumber)
		h.RespondWithError(w, http.StatusBadRequest, "Invalid page number")
		return
	}

	// Add deprecation headers
	newPath := fmt.Sprintf("/v1/medicaments?page=%v", page)
	h.AddDeprecationHeaders(w, r, newPath)

	medicaments := h.dataStore.GetMedicaments()
	pageSize := 10
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

	h.RespondWithJSON(w, http.StatusOK, response)
}

// FindMedicament searches for medicaments by name or CIP using query parameters
func (h *HTTPHandlerImpl) FindMedicament(w http.ResponseWriter, r *http.Request) {
	element := r.PathValue("element")
	if element == "" {
		h.RespondWithError(w, http.StatusBadRequest, "Missing search term")
		return
	}

	// Validate input using the validator
	if err := h.validator.ValidateInput(element); err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize input (replace + with space for flexible matching)
	sanitizedElement := strings.ToLower(element)
	sanitizedElement = strings.ReplaceAll(sanitizedElement, "+", " ")

	// Add deprecation headers
	newPath := fmt.Sprintf("/v1/medicament?search=%v", element)
	h.AddDeprecationHeaders(w, r, newPath)

	medicaments := h.dataStore.GetMedicaments()
	var results []entities.Medicament

	for _, med := range medicaments {
		if strings.Contains(med.DenominationNormalized, sanitizedElement) {
			results = append(results, med)
		}
	}

	// Return 404 if no results found
	if len(results) == 0 {
		h.RespondWithError(w, http.StatusNotFound, "No medicaments found")
		return
	}

	h.RespondWithJSON(w, http.StatusOK, results)
}

// FindMedicamentByCIS finds a medicament by CIS
func (h *HTTPHandlerImpl) FindMedicamentByCIS(w http.ResponseWriter, r *http.Request) {
	cisStr := r.PathValue("cis")
	path := r.URL.Path

	cis, err := h.validator.ValidateCIS(cisStr)

	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Add deprecation headers for legacy endpoint
	if strings.HasPrefix(path, "/medicament/id/") {
		newPath := fmt.Sprintf("/v1/medicaments/%v", cis)
		h.AddDeprecationHeaders(w, r, newPath)
	}

	medicamentsMap := h.dataStore.GetMedicamentsMap()
	med, exists := medicamentsMap[cis]
	if !exists {
		h.RespondWithError(w, http.StatusNotFound, "Medicament not found")
		return
	}

	h.RespondWithJSON(w, http.StatusOK, med)
}

// FindMedicamentByCIP finds a medicament by its presentation cip7 or cip13
func (h *HTTPHandlerImpl) FindMedicamentByCIP(w http.ResponseWriter, r *http.Request) {
	cipStr := r.PathValue("cip")
	cip, err := h.validator.ValidateCIP(cipStr)
	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Add deprecation headers
	newPath := fmt.Sprintf("/v1/medicament?cip=%v", cip)
	h.AddDeprecationHeaders(w, r, newPath)

	med, found := h.findMedicamentByCIP(cip)
	if !found {
		h.RespondWithError(w, http.StatusNotFound, "Medicament not found")
		return
	}

	h.RespondWithJSONAndETag(w, r, http.StatusOK, med)
}

// FindGeneriques searches for generiques by libelle (case-insensitive partial match)
func (h *HTTPHandlerImpl) FindGeneriques(w http.ResponseWriter, r *http.Request) {
	libelle := r.PathValue("libelle")
	if libelle == "" {
		h.RespondWithError(w, http.StatusBadRequest, "Missing libelle")
		return
	}

	// Validate input using the validator
	if err := h.validator.ValidateInput(libelle); err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize input and convert to lowercase for case-insensitive search
	sanitizedLibelle := strings.ToLower(libelle)
	// Normalize: replace + with space for flexible matching
	sanitizedLibelle = strings.ReplaceAll(sanitizedLibelle, "+", " ")

	// Add deprecation headers
	newPath := fmt.Sprintf("/v1/generiques?libelle=%v", libelle)
	h.AddDeprecationHeaders(w, r, newPath)

	generiques := h.dataStore.GetGeneriques()
	var results []entities.GeneriqueList

	for _, gen := range generiques {
		if strings.Contains(gen.LibelleNormalized, sanitizedLibelle) {
			results = append(results, gen)
		}
	}

	if len(results) == 0 {
		h.RespondWithError(w, http.StatusNotFound, "No generiques found")
		return
	}

	h.RespondWithJSON(w, http.StatusOK, results)
}

// FindGeneriquesByGroupID finds generiques by group ID
func (h *HTTPHandlerImpl) FindGeneriquesByGroupID(w http.ResponseWriter, r *http.Request) {
	// Support both v1 path parameter (groupID) and legacy path parameter (groupId)
	groupIDStr := r.PathValue("groupID")
	if groupIDStr == "" {
		groupIDStr = r.PathValue("groupId")
	}

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	path := r.URL.Path

	// Add deprecation headers for legacy endpoint only
	if strings.HasPrefix(path, "/generiques/group/") {
		newPath := fmt.Sprintf("/v1/generiques/%v", groupID)
		h.AddDeprecationHeaders(w, r, newPath)
	}

	generiquesMap := h.dataStore.GetGeneriquesMap()
	gen, exists := generiquesMap[groupID]
	if !exists {
		h.RespondWithError(w, http.StatusNotFound, "Generique group not found")
		return
	}

	h.RespondWithJSON(w, http.StatusOK, gen)
}

// HealthCheck returns server health information
func (h *HTTPHandlerImpl) HealthCheck(w http.ResponseWriter, r *http.Request) {
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
func (h *HTTPHandlerImpl) ServeDiagnosticsV1(w http.ResponseWriter, r *http.Request) {
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

func (h *HTTPHandlerImpl) ServePresentationsV1(w http.ResponseWriter, r *http.Request) {
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
		h.RespondWithJSONAndETag(w, r, http.StatusOK, pres)
		return
	}

	// If not, in the CIP13
	presentationsCIP13 := h.dataStore.GetPresentationsCIP13Map()

	if pres, ok := presentationsCIP13[cip]; ok {
		h.RespondWithJSONAndETag(w, r, http.StatusOK, pres)
		return
	}

	// If not found, return error
	h.RespondWithError(w, http.StatusNotFound, "Presentation not found")
}

// ServePresentationsMissingCIP handles requests to /v1/presentations/ without a CIP path parameter.
// Returns a 400 Bad Request error indicating that the CIP path parameter is required.
func (h *HTTPHandlerImpl) ServePresentationsMissingCIP(w http.ResponseWriter, r *http.Request) {
	h.RespondWithError(w, http.StatusBadRequest, "CIP path parameter is required")
}

func (h *HTTPHandlerImpl) ServeGeneriquesV1(w http.ResponseWriter, r *http.Request) {
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
		}
	}

	if len(results) != 0 {
		h.RespondWithJSONAndETag(w, r, http.StatusOK, results)
		return
	}
	h.RespondWithError(w, http.StatusNotFound, "No generiques found")
}

func (h *HTTPHandlerImpl) ServeMedicamentsV1(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	totalParams := 0
	for _, v := range []string{q.Get("cip"), q.Get("search"), q.Get("page")} {
		if v != "" {
			totalParams++
		}
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

		medicaments := h.dataStore.GetMedicaments()
		pageSize := 10
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
			}
		}

		// Return 404 if no results found
		if len(results) == 0 {
			h.RespondWithError(w, http.StatusNotFound, "No medicaments found")
			return
		}

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

		h.RespondWithJSON(w, http.StatusOK, med)
		return
	}

	h.RespondWithError(w, http.StatusBadRequest, "Unexpected error")

}
