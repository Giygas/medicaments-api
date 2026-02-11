// Package health provides health checking functionality for the medicaments API.
package health

import (
	"math"
	"net/http"
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

// HealthCheck returns HTTP-specific health data with stricter thresholds
// Used by /health HTTP endpoint
func (h *HealthCheckerImpl) HealthCheck() (status string, data map[string]any, httpStatus int) {
	// Get data statistics
	medicaments := h.dataStore.GetMedicaments()
	generiques := h.dataStore.GetGeneriques()
	lastUpdate := h.dataStore.GetLastUpdated()
	isUpdating := h.dataStore.IsUpdating()

	dataAge := time.Since(lastUpdate)

	// Determine health status and HTTP code using stricter thresholds
	switch {
	case len(medicaments) == 0 || len(generiques) == 0:
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable

	case dataAge > 48*time.Hour:
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable

	case dataAge > 24*time.Hour:
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable

	case isUpdating && dataAge > 6*time.Hour:
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable

	default:
		status = "healthy"
		httpStatus = http.StatusOK
	}

	// Build response data (no system metrics, only data-related fields)
	data = map[string]any{
		"last_update":    lastUpdate.Format(time.RFC3339),
		"data_age_hours": math.Round(dataAge.Hours()*10) / 10,
		"medicaments":    len(medicaments),
		"generiques":     len(generiques),
		"is_updating":    isUpdating,
	}

	return status, data, httpStatus
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
