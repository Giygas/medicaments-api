// Package interfaces defines core abstractions for the medicaments API
// to improve testability, maintainability, and separation of concerns.
package interfaces

import (
	"net/http"
	"time"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// DataStore defines the contract for data storage operations.
// It provides thread-safe access to medicaments and generiques data
// with atomic operations for zero-downtime updates.
type DataStore interface {
	// Data retrieval methods
	GetMedicaments() []entities.Medicament
	GetGeneriques() []entities.GeneriqueList
	GetMedicamentsMap() map[int]entities.Medicament
	GetGeneriquesMap() map[int]entities.Generique
	GetLastUpdated() time.Time
	IsUpdating() bool

	// Data update methods
	UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList,
		medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.Generique)
	BeginUpdate() bool
	EndUpdate()
}

// Parser defines the contract for parsing medicament data from external sources.
// It handles downloading, processing, and transforming raw data into structured entities.
type Parser interface {
	// ParseAllMedicaments downloads and parses all medicament data
	ParseAllMedicaments() ([]entities.Medicament, error)

	// GeneriquesParser processes medicaments data to create generique groups
	GeneriquesParser(medicaments *[]entities.Medicament, medicamentsMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.Generique, error)
}

// Scheduler defines the contract for job scheduling and health monitoring.
// It manages automated data updates and system health checks.
type Scheduler interface {
	// Lifecycle management
	Start() error
	Stop()
}

// HTTPHandler defines the contract for HTTP request handlers.
// It provides a consistent interface for all API endpoints.
type HTTPHandler interface {
	// ServeHTTP implements the http.Handler interface
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Specific endpoint handlers
	ServeAllMedicaments(w http.ResponseWriter, r *http.Request)
	ServePagedMedicaments(w http.ResponseWriter, r *http.Request)
	FindMedicament(w http.ResponseWriter, r *http.Request)
	FindMedicamentByID(w http.ResponseWriter, r *http.Request)
	FindGeneriques(w http.ResponseWriter, r *http.Request)
	FindGeneriquesByGroupID(w http.ResponseWriter, r *http.Request)
	HealthCheck(w http.ResponseWriter, r *http.Request)
}

// HealthChecker defines the contract for health check functionality.
// It provides system health monitoring and reporting.
type HealthChecker interface {
	// HealthCheck returns current system health status
	HealthCheck() (status string, details map[string]interface{}, err error)

	// CalculateNextUpdate returns the next scheduled update time
	CalculateNextUpdate() time.Time
}

// DataValidator defines the contract for data validation operations.
// It ensures data integrity and consistency.
type DataValidator interface {
	// ValidateMedicament checks if a medicament entity is valid
	ValidateMedicament(m *entities.Medicament) error

	// ValidateDataIntegrity performs comprehensive data validation
	ValidateDataIntegrity(medicaments []entities.Medicament, generiques []entities.GeneriqueList) error

	// ValidateInput validates user input strings
	ValidateInput(input string) error
}
