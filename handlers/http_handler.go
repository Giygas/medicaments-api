// Package handlers provides HTTP request handlers for the medicaments API endpoints.
// This file implements the HTTPHandler interface with dependency injection.
package handlers

import (
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
	Status        string                 `json:"status"`
	LastUpdate    string                 `json:"last_update"`
	DataAgeHours  float64                `json:"data_age_hours"`
	UptimeSeconds float64                `json:"uptime_seconds"`
	Data          map[string]interface{} `json:"data"`
	System        map[string]interface{} `json:"system"`
}

// RespondWithJSON writes a JSON response with compression optimization
func (h *HTTPHandlerImpl) RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		logging.Error("Failed to marshal JSON response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.WriteHeader(code)
	w.Write(data)
}

// RespondWithError writes a JSON error response
func (h *HTTPHandlerImpl) RespondWithError(w http.ResponseWriter, code int, message string) {
	errorResponse := map[string]interface{}{
		"error":   http.StatusText(code),
		"message": message,
		"code":    code,
	}
	h.RespondWithJSON(w, code, errorResponse)
}

// formatUptimeHuman formats duration into a human-readable string
func (h *HTTPHandlerImpl) formatUptimeHuman(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	parts = append(parts, fmt.Sprintf("%ds", seconds))

	return strings.Join(parts, " ")
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

// ServeAllMedicaments returns all medicaments
func (h *HTTPHandlerImpl) ServeAllMedicaments(w http.ResponseWriter, r *http.Request) {
	medicaments := h.dataStore.GetMedicaments()
	h.RespondWithJSON(w, http.StatusOK, medicaments)
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

	response := map[string]interface{}{
		"data":       pagedMedicaments,
		"page":       page,
		"pageSize":   pageSize,
		"totalItems": totalItems,
		"maxPage":    maxPage,
	}

	h.RespondWithJSON(w, http.StatusOK, response)
}

// FindMedicament searches for medicaments by name
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

	// Sanitize input
	sanitizedElement := regexp.QuoteMeta(strings.ToLower(element))

	medicaments := h.dataStore.GetMedicaments()
	var results []entities.Medicament

	for _, med := range medicaments {
		if strings.Contains(strings.ToLower(med.Denomination), sanitizedElement) {
			results = append(results, med)
		}
	}

	// Always return 200 with results array (empty if no matches)
	h.RespondWithJSON(w, http.StatusOK, results)
}

// FindMedicamentByID finds a medicament by CIS
func (h *HTTPHandlerImpl) FindMedicamentByID(w http.ResponseWriter, r *http.Request) {
	cisStr := chi.URLParam(r, "cis")
	cis, err := strconv.Atoi(cisStr)
	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, "Invalid CIS")
		return
	}

	medicamentsMap := h.dataStore.GetMedicamentsMap()
	med, exists := medicamentsMap[cis]
	if !exists {
		h.RespondWithError(w, http.StatusNotFound, "Medicament not found")
		return
	}

	h.RespondWithJSON(w, http.StatusOK, med)
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

	generiques := h.dataStore.GetGeneriques()
	var results []entities.GeneriqueList

	for _, gen := range generiques {
		if strings.Contains(strings.ToLower(gen.Libelle), sanitizedLibelle) {
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

	// Calculate uptime (using a fixed start time for now)
	uptime := time.Since(time.Now().Add(-24 * time.Hour)) // Placeholder

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
		Data: map[string]interface{}{
			"api_version": "1.0",
			"medicaments": len(medicaments),
			"generiques":  len(generiques),
			"is_updating": isUpdating,
			"next_update": h.calculateNextUpdate().Format(time.RFC3339),
		},
		System: map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]interface{}{
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
