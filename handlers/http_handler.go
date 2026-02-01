// Package handlers provides HTTP request handlers for the medicaments API endpoints.
// This file implements the HTTPHandler interface with dependency injection.
package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

// HTTPHandlerImpl implements the interfaces.HTTPHandler interface
type HTTPHandlerImpl struct {
	dataStore interfaces.DataStore
	validator interfaces.DataValidator
}

// NewHTTPHandler creates a new HTTP handler with injected dependencies
func NewHTTPHandler(dataStore interfaces.DataStore, validator interfaces.DataValidator) interfaces.HTTPHandler {
	return &HTTPHandlerImpl{
		dataStore: dataStore,
		validator: validator,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *HTTPHandlerImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// This is a placeholder - the actual routing is handled by chi
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// HealthResponseImpl defines the structure for consistent JSON ordering
type HealthResponseImpl struct {
	Status        string         `json:"status"`
	LastUpdate    string         `json:"last_update"`
	DataAgeHours  float64        `json:"data_age_hours"`
	UptimeSeconds float64        `json:"uptime_seconds"`
	Data          map[string]any `json:"data"`
	System        map[string]any `json:"system"`
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
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
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

	// Cloudflare-specific optimizations
	w.Header().Set("CF-Cache-Status", "DYNAMIC") // Tell Cloudflare this is dynamic content
	w.Header().Set("CF-RAY", "")                 // Will be set by Cloudflare
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		logging.Error("Failed to write response", "error", err)
	}
}

// calculateNextUpdate calculates the next scheduled update time
func (h *HTTPHandlerImpl) calculateNextUpdate() time.Time {
	now := time.Now()

	// Get today's 6:00 AM and 6:00 PM times
	sixAM := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())

	// If current time is before 6:00 AM, next update is 6:00 AM today
	if now.Before(sixAM) {
		return sixAM
	}

	// If current time is between 6:00 AM and 6:00 PM, next update is 6:00 PM today
	if now.Before(sixPM) {
		return sixPM
	}

	// If current time is after 6:00 PM, next update is 6:00 AM tomorrow
	tomorrow := now.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 6, 0, 0, 0, tomorrow.Location())
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
	fmt.Printf("path: %v\n", path)

	medicaments := h.dataStore.GetMedicaments()
	h.RespondWithJSONAndETag(w, r, http.StatusOK, medicaments)
}

// ServePagedMedicaments returns paginated medicaments
func (h *HTTPHandlerImpl) ServePagedMedicaments(w http.ResponseWriter, r *http.Request) {

	pageNumber := chi.URLParam(r, "pageNumber")
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
	element := chi.URLParam(r, "element")
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
	sanitizedElement := regexp.QuoteMeta(strings.ToLower(element))
	sanitizedElement = strings.ReplaceAll(sanitizedElement, "+", " ")

	// Add deprecation headers
	newPath := fmt.Sprintf("/v1/medicament?search=%v", element)
	h.AddDeprecationHeaders(w, r, newPath)

	medicaments := h.dataStore.GetMedicaments()
	var results []entities.Medicament

	for _, med := range medicaments {
		// Normalize medication name for comparison
		medName := strings.ToLower(med.Denomination)
		medName = strings.ReplaceAll(medName, "+", " ")

		if strings.Contains(medName, sanitizedElement) {
			results = append(results, med)
		}
	}

	// Always return 200 with results array (empty if no matches)
	h.RespondWithJSON(w, http.StatusOK, results)
}

// FindMedicamentByID finds a medicament by CIS
func (h *HTTPHandlerImpl) FindMedicamentByID(w http.ResponseWriter, r *http.Request) {
	cisStr := chi.URLParam(r, "cis")

	cis, err := h.validator.ValidateCIS(cisStr)

	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Add deprecation headers
	newPath := fmt.Sprintf("/v1/medicament?cis=%v", cis)
	h.AddDeprecationHeaders(w, r, newPath)

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
	cipStr := chi.URLParam(r, "cip")

	cip, err := h.validator.ValidateCIP(cipStr)

	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Add deprecation headers
	newPath := fmt.Sprintf("/v1/medicament?cip=%v", cip)
	h.AddDeprecationHeaders(w, r, newPath)

	medicamentsMap := h.dataStore.GetMedicamentsMap()

	// Search in CIP7 map first (O(1) lookup)
	presentationsCIP7 := h.dataStore.GetPresentationsCIP7Map()
	if pres, ok := presentationsCIP7[cip]; ok {
		if med, exists := medicamentsMap[pres.Cis]; exists {
			h.RespondWithJSONAndETag(w, r, http.StatusOK, med)
			return
		}
	}

	// If not found, try CIP13 map (O(1) lookup)
	presentationsCIP13 := h.dataStore.GetPresentationsCIP13Map()
	if pres, ok := presentationsCIP13[cip]; ok {
		if med, exists := medicamentsMap[pres.Cis]; exists {
			h.RespondWithJSONAndETag(w, r, http.StatusOK, med)
			return
		}
	}

	h.RespondWithError(w, http.StatusNotFound, "Medicament not found")
}

// FindGeneriques searches for generiques by libelle (case-insensitive partial match)
func (h *HTTPHandlerImpl) FindGeneriques(w http.ResponseWriter, r *http.Request) {
	libelle := chi.URLParam(r, "libelle")
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
		// Normalize generique libelle for comparison
		genLibelle := strings.ToLower(gen.Libelle)
		genLibelle = strings.ReplaceAll(genLibelle, "+", " ")

		if strings.Contains(genLibelle, sanitizedLibelle) {
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
	groupIDStr := chi.URLParam(r, "groupId")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	// Add deprecation headers
	newPath := fmt.Sprintf("/v1/generiques?group=%v", groupID)
	h.AddDeprecationHeaders(w, r, newPath)

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
	medicaments := h.dataStore.GetMedicaments()
	generiques := h.dataStore.GetGeneriques()
	lastUpdate := h.dataStore.GetLastUpdated()
	isUpdating := h.dataStore.IsUpdating()
	dataAge := time.Since(lastUpdate)

	// Determine health status based on data availability and age
	var healthStatus string
	var httpStatus int
	switch {
	case len(medicaments) == 0:
		healthStatus = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	case dataAge > 24*time.Hour:
		healthStatus = "degraded"
		httpStatus = http.StatusOK
	default:
		healthStatus = "healthy"
		httpStatus = http.StatusOK
	}

	response := HealthResponseImpl{
		Status:        healthStatus,
		LastUpdate:    lastUpdate.Format(time.RFC3339),
		DataAgeHours:  dataAge.Hours(),
		UptimeSeconds: uptime.Seconds(),
		Data: map[string]any{
			"api_version": "1.0",
			"medicaments": len(medicaments),
			"generiques":  len(generiques),
			"is_updating": isUpdating,
			"next_update": h.calculateNextUpdate().Format(time.RFC3339),
		},
		System: map[string]any{
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]any{
				"alloc_mb":       int(m.Alloc / 1024 / 1024),
				"total_alloc_mb": int(m.TotalAlloc / 1024 / 1024),
				"sys_mb":         int(m.Sys / 1024 / 1024),
				"num_gc":         m.NumGC,
			},
		},
	}

	// Set appropriate status code
	if healthStatus == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	h.RespondWithJSON(w, httpStatus, response)
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

func (h *HTTPHandlerImpl) ServeGeneriquesV1(w http.ResponseWriter, r *http.Request) {

	groupStr := r.URL.Query().Get("group")
	libelle := r.URL.Query().Get("libelle")

	if groupStr == "" && libelle == "" {
		h.RespondWithError(w, http.StatusBadRequest, "Needs libelle or group param")
		return
	}

	// GroupID block
	if groupStr != "" {
		groupID, err := strconv.Atoi(groupStr)
		if err != nil {
			h.RespondWithError(w, http.StatusBadRequest, "Invalid group ID")
			return
		}

		if groupID <= 0 || groupID > 9999 {
			h.RespondWithError(w, http.StatusBadRequest, "Group ID should be between 1 and 9999")
			return
		}

		// Get the generiques group map for the group ID param
		generiquesGroupMap, exists := h.dataStore.GetGeneriquesMap()[groupID]
		if exists {
			h.RespondWithJSONAndETag(w, r, http.StatusOK, generiquesGroupMap)
			return
		}

		h.RespondWithError(w, http.StatusNotFound, "Generique group not found")
	}

	// Libelle block
	if libelle != "" {
		// Validate user input using the validator
		if err := h.validator.ValidateInput(libelle); err != nil {
			h.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Sanitize input and convert to lowercase for case-insensitive search
		sanitizedLibelle := strings.ToLower(libelle)
		// Normalize: replace + with space for flexible matching
		sanitizedLibelle = strings.ReplaceAll(sanitizedLibelle, "+", " ")

		generiques := h.dataStore.GetGeneriques()
		var results []entities.GeneriqueList

		for _, gen := range generiques {
			// Normalize generique libelle for comparison
			genLibelle := strings.ToLower(gen.Libelle)
			genLibelle = strings.ReplaceAll(genLibelle, "+", " ")

			if strings.Contains(genLibelle, sanitizedLibelle) {
				results = append(results, gen)
			}
		}

		if len(results) != 0 {
			h.RespondWithJSONAndETag(w, r, http.StatusOK, results)
			return
		}
		h.RespondWithError(w, http.StatusNotFound, "No generiques found")

	}
}

func (h *HTTPHandlerImpl) ServeMedicamentsV1(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	totalParams := 0
	for _, v := range []string{q.Get("cip"), q.Get("cis"), q.Get("search"), q.Get("page")} {
		if v != "" {
			totalParams++
		}
	}

	if totalParams == 0 {
		h.RespondWithError(w, http.StatusBadRequest, "Needs at least one param. See documentation")
		return
	}

	if totalParams > 1 {
		h.RespondWithError(w, http.StatusBadRequest, "Only one parameter allowed at a time. Choose: page, cis, cip, search")
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
		sanitizedElement := regexp.QuoteMeta(strings.ToLower(searchQuery))
		sanitizedElement = strings.ReplaceAll(sanitizedElement, "+", " ")

		medicaments := h.dataStore.GetMedicaments()
		var results []entities.Medicament

		for _, med := range medicaments {
			// Normalize medication name for comparison
			medName := strings.ToLower(med.Denomination)
			medName = strings.ReplaceAll(medName, "+", " ")

			if strings.Contains(medName, sanitizedElement) {
				results = append(results, med)
			}
		}

		// Always return 200 with results array (empty if no matches)
		h.RespondWithJSONAndETag(w, r, http.StatusOK, results)
		return
	}

	// Search medicament by CIS
	if cisStr := q.Get("cis"); cisStr != "" {

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

		h.RespondWithJSON(w, http.StatusOK, med)
		return
	}

	// Search medicament by CIP7 or CIP13
	if cipStr := q.Get("cip"); cipStr != "" {
		cip, err := h.validator.ValidateCIP(cipStr)
		if err != nil {
			h.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		medicamentsMap := h.dataStore.GetMedicamentsMap()

		// Search in CIP7 map first (O(1) lookup)
		presentationsCIP7 := h.dataStore.GetPresentationsCIP7Map()
		if pres, ok := presentationsCIP7[cip]; ok {
			if med, exists := medicamentsMap[pres.Cis]; exists {
				h.RespondWithJSON(w, http.StatusOK, med)
				return
			}
		}

		// If not found, try CIP13 map (O(1) lookup)
		presentationsCIP13 := h.dataStore.GetPresentationsCIP13Map()
		if pres, ok := presentationsCIP13[cip]; ok {
			if med, exists := medicamentsMap[pres.Cis]; exists {
				h.RespondWithJSON(w, http.StatusOK, med)
				return
			}
		}

		h.RespondWithError(w, http.StatusNotFound, "Medicament not found")
		return
	}

	h.RespondWithError(w, http.StatusBadRequest, "Unexpected error")

}
