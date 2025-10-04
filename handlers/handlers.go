// Package handlers provides HTTP request handlers for the medicaments API endpoints.
// It includes handlers for medicament search, generique lookup, pagination, health checks,
// and response formatting with proper input validation and error handling.
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

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

// Global variables (these will be moved to a better place later)
var serverStartTime = time.Now()

// RespondWithJSON writes a JSON response with compression optimization
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
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

// formatUptimeHuman formats duration into a human-readable string
func formatUptimeHuman(d time.Duration) string {
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
func calculateNextUpdate() time.Time {
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
func ServeAllMedicaments(dataContainer *data.DataContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		medicaments := dataContainer.GetMedicaments()
		RespondWithJSON(w, http.StatusOK, medicaments)
	}
}

// ServePagedMedicaments returns paginated medicaments
func ServePagedMedicaments(dataContainer *data.DataContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageNumber := chi.URLParam(r, "pageNumber")
		page, err := strconv.Atoi(pageNumber)
		if err != nil || page < 1 {
			logging.Warn("Unusual user input", "pageNumber", pageNumber)
			http.Error(w, "Invalid page number", http.StatusBadRequest)
			return
		}

		medicaments := dataContainer.GetMedicaments()
		pageSize := 10
		start := (page - 1) * pageSize
		end := start + pageSize

		if start >= len(medicaments) {
			http.Error(w, "Page not found", http.StatusNotFound)
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

		RespondWithJSON(w, http.StatusOK, response)
	}
}

// FindMedicament searches for medicaments by name
func FindMedicament(dataContainer *data.DataContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		element := chi.URLParam(r, "element")
		if element == "" {
			http.Error(w, "Missing search term", http.StatusBadRequest)
			return
		}

		// Sanitize input
		sanitizedElement := regexp.QuoteMeta(strings.ToLower(element))

		medicaments := dataContainer.GetMedicaments()
		var results []entities.Medicament

		for _, med := range medicaments {
			if strings.Contains(strings.ToLower(med.Denomination), sanitizedElement) {
				results = append(results, med)
			}
		}

		if len(results) == 0 {
			http.Error(w, "No medicaments found", http.StatusNotFound)
			return
		}

		RespondWithJSON(w, http.StatusOK, results)
	}
}

// FindMedicamentByID finds a medicament by CIS
func FindMedicamentByID(dataContainer *data.DataContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cisStr := chi.URLParam(r, "cis")
		cis, err := strconv.Atoi(cisStr)
		if err != nil {
			http.Error(w, "Invalid CIS", http.StatusBadRequest)
			return
		}

		medicamentsMap := dataContainer.GetMedicamentsMap()
		med, exists := medicamentsMap[cis]
		if !exists {
			http.Error(w, "Medicament not found", http.StatusNotFound)
			return
		}

		RespondWithJSON(w, http.StatusOK, med)
	}
}

// FindGeneriques searches for generiques by libelle (case-insensitive partial match)
func FindGeneriques(dataContainer *data.DataContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libelle := chi.URLParam(r, "libelle")
		if libelle == "" {
			http.Error(w, "Missing libelle", http.StatusBadRequest)
			return
		}

		// Sanitize input and convert to lowercase for case-insensitive search
		sanitizedLibelle := strings.ToLower(libelle)

		generiques := dataContainer.GetGeneriques()
		var results []entities.GeneriqueList

		for _, gen := range generiques {
			if strings.Contains(strings.ToLower(gen.Libelle), sanitizedLibelle) {
				results = append(results, gen)
			}
		}

		if len(results) == 0 {
			http.Error(w, "No generiques found", http.StatusNotFound)
			return
		}

		RespondWithJSON(w, http.StatusOK, results)
	}
}

// FindGeneriquesByGroupID finds generiques by group ID
func FindGeneriquesByGroupID(dataContainer *data.DataContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := chi.URLParam(r, "groupId")
		groupID, err := strconv.Atoi(groupIDStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		generiquesMap := dataContainer.GetGeneriquesMap()
		gen, exists := generiquesMap[groupID]
		if !exists {
			http.Error(w, "Generique group not found", http.StatusBadRequest)
			return
		}

		RespondWithJSON(w, http.StatusOK, gen)
	}
}

// HealthResponse defines the structure for consistent JSON ordering
type HealthResponse struct {
	Status         string                 `json:"status"`
	LastUpdate     string                 `json:"last_update"`
	DataAgeHours   float64                `json:"data_age_hours"`
	UptimeSeconds  float64                `json:"uptime_seconds"`
	Data           map[string]interface{} `json:"data"`
	System         map[string]interface{} `json:"system"`
}

// HealthCheck returns server health information
func HealthCheck(dataContainer *data.DataContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get memory statistics
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Calculate uptime
		uptime := time.Since(serverStartTime)

		// Get data statistics
		medicaments := dataContainer.GetMedicaments()
		generiques := dataContainer.GetGeneriques()
		lastUpdate := dataContainer.GetLastUpdated()
		isUpdating := dataContainer.IsUpdating()
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

		response := HealthResponse{
			Status:       healthStatus,
			LastUpdate:   lastUpdate.Format(time.RFC3339),
			DataAgeHours: dataAge.Hours(),
			UptimeSeconds: uptime.Seconds(),
			Data: map[string]interface{}{
				"api_version": "1.0",
				"medicaments":  len(medicaments),
				"generiques":   len(generiques),
				"is_updating":  isUpdating,
				"next_update":  calculateNextUpdate().Format(time.RFC3339),
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

		RespondWithJSON(w, httpStatus, response)
	}
}
