// Package health provides health checking functionality for the medicaments API.
package health

import (
	"runtime"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
)

// HealthCheckerImpl implements the interfaces.HealthChecker interface
type HealthCheckerImpl struct {
	dataStore interfaces.DataStore
}

// NewHealthChecker creates a new health checker with injected dependencies
func NewHealthChecker(dataStore interfaces.DataStore) interfaces.HealthChecker {
	return &HealthCheckerImpl{
		dataStore: dataStore,
	}
}

// HealthCheck returns current system health status
func (h *HealthCheckerImpl) HealthCheck() (status string, details map[string]any, err error) {
	// Get memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get data statistics
	medicaments := h.dataStore.GetMedicaments()
	generiques := h.dataStore.GetGeneriques()
	lastUpdate := h.dataStore.GetLastUpdated()
	isUpdating := h.dataStore.IsUpdating()
	dataAge := time.Since(lastUpdate)

	// Determine health status based on data availability and age
	switch {
	case len(medicaments) == 0:
		status = "unhealthy"
	case dataAge > 24*time.Hour:
		status = "degraded"
	default:
		status = "healthy"
	}

	// Build details map
	details = map[string]any{
		"last_update":    lastUpdate.Format(time.RFC3339),
		"data_age_hours": dataAge.Hours(),
		"data": map[string]any{
			"api_version": "1.0",
			"medicaments": len(medicaments),
			"generiques":  len(generiques),
			"is_updating": isUpdating,
			"next_update": h.CalculateNextUpdate().Format(time.RFC3339),
		},
		"system": map[string]any{
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]any{
				"alloc_mb":       int(m.Alloc / 1024 / 1024),
				"total_alloc_mb": int(m.TotalAlloc / 1024 / 1024),
				"sys_mb":         int(m.Sys / 1024 / 1024),
				"num_gc":         m.NumGC,
			},
		},
	}

	return status, details, nil
}

// CalculateNextUpdate returns the next scheduled update time
func (h *HealthCheckerImpl) CalculateNextUpdate() time.Time {
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
	return sixAM.AddDate(0, 0, 1)
}
