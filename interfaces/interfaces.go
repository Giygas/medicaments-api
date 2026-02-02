// Package interfaces defines core abstractions for the medicaments API
// to improve testability, maintainability, and separation of concerns.
package interfaces

import (
	"net/http"
	"time"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// DataQualityReport provides a summary of data quality issues
type DataQualityReport struct {
	DuplicateCIS                    []int
	DuplicateGroupIDs               []int
	MedicamentsWithoutConditions    int
	MedicamentsWithoutGeneriques    int
	MedicamentsWithoutPresentations int // Count of medicaments without presentations
	MedicamentsWithoutCompositions  int // Count of medicaments without compositions
	GeneriqueOnlyCIS                int // CIS values in generiques that don't have corresponding medicaments
}

// DataStore defines the contract for data storage operations.
// It provides thread-safe access to medicaments and generiques data
// with atomic operations for zero-downtime updates.
type DataStore interface {
	// Data retrieval methods
	GetMedicaments() []entities.Medicament
	GetGeneriques() []entities.GeneriqueList
	GetMedicamentsMap() map[int]entities.Medicament
	GetGeneriquesMap() map[int]entities.GeneriqueList
	GetPresentationsCIP7Map() map[int]entities.Presentation
	GetPresentationsCIP13Map() map[int]entities.Presentation
	GetLastUpdated() time.Time
	IsUpdating() bool
	GetServerStartTime() time.Time

	// Data update methods
	UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList,
		medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.GeneriqueList,
		presentationsCIP7Map map[int]entities.Presentation, presentationsCIP13Map map[int]entities.Presentation)
	BeginUpdate() bool
	EndUpdate()
}

// Parser defines the contract for parsing medicament data from external sources.
// It handles downloading, processing, and transforming raw data into structured entities.
type Parser interface {
	// ParseAllMedicaments downloads and parses all medicament data
	ParseAllMedicaments() ([]entities.Medicament, map[int]entities.Presentation,
		map[int]entities.Presentation, error)

	// GeneriquesParser processes medicaments data to create generique groups
	GeneriquesParser(medicaments *[]entities.Medicament, medicamentsMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.GeneriqueList, error)
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
	ServePagedMedicaments(w http.ResponseWriter, r *http.Request)
	FindMedicament(w http.ResponseWriter, r *http.Request)
	FindMedicamentByCIS(w http.ResponseWriter, r *http.Request)
	FindMedicamentByCIP(w http.ResponseWriter, r *http.Request)
	FindGeneriques(w http.ResponseWriter, r *http.Request)
	FindGeneriquesByGroupID(w http.ResponseWriter, r *http.Request)
	// This will stay in all versions
	HealthCheck(w http.ResponseWriter, r *http.Request)
	ExportMedicaments(w http.ResponseWriter, r *http.Request)

	// V1 handlers
	ServeMedicamentsV1(w http.ResponseWriter, r *http.Request)
	ServePresentationsV1(w http.ResponseWriter, r *http.Request)
	ServeGeneriquesV1(w http.ResponseWriter, r *http.Request)
}

// HealthChecker defines the contract for health check functionality.
// It provides system health monitoring and reporting.
type HealthChecker interface {
	// HealthCheck returns current system health status
	HealthCheck() (status string, details map[string]any, err error)

	// CalculateNextUpdate returns the next scheduled update time
	CalculateNextUpdate() time.Time
}

// DataValidator defines the contract for data validation operations.
// It ensures data integrity and consistency.
type DataValidator interface {
	// ValidateMedicament checks if a medicament entity is valid
	ValidateMedicament(m *entities.Medicament) error

	// CheckDuplicateCIP validates that CIP7 and CIP13 values are unique
	CheckDuplicateCIP(presentations []entities.Presentation) error

	// ValidateDataIntegrity performs comprehensive data validation
	ValidateDataIntegrity(medicaments []entities.Medicament, generiques []entities.GeneriqueList) error

	// ReportDataQuality generates a data quality report with all issues found
	ReportDataQuality(medicaments []entities.Medicament, generiques []entities.GeneriqueList) *DataQualityReport

	// ValidateInput validates user input strings
	ValidateInput(input string) error

	// ValidateCIP validates CIS codes
	ValidateCIS(input string) (int, error)

	// ValidateCIP validates CIP codes
	ValidateCIP(input string) (int, error)
}
