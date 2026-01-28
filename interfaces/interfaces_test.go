package interfaces

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// MockDataStore implements DataStore interface for testing
type MockDataStore struct {
	medicaments           []entities.Medicament
	generiques            []entities.GeneriqueList
	medicamentsMap        map[int]entities.Medicament
	generiquesMap         map[int]entities.Generique
	presentationsCIP7Map  map[int]entities.Presentation
	presentationsCIP13Map map[int]entities.Presentation
	lastUpdated           time.Time
	updating              bool
}

func (m *MockDataStore) GetMedicaments() []entities.Medicament {
	return m.medicaments
}

func (m *MockDataStore) GetGeneriques() []entities.GeneriqueList {
	return m.generiques
}

func (m *MockDataStore) GetMedicamentsMap() map[int]entities.Medicament {
	return m.medicamentsMap
}

func (m *MockDataStore) GetGeneriquesMap() map[int]entities.Generique {
	return m.generiquesMap
}

func (m *MockDataStore) GetPresentationsCIP7Map() map[int]entities.Presentation {
	return m.presentationsCIP7Map
}

func (m *MockDataStore) GetPresentationsCIP13Map() map[int]entities.Presentation {
	return m.presentationsCIP13Map
}

func (m *MockDataStore) GetLastUpdated() time.Time {
	return m.lastUpdated
}

func (m *MockDataStore) IsUpdating() bool {
	return m.updating
}

func (m *MockDataStore) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList, medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.Generique, presentationsCIP7Map map[int]entities.Presentation, presentationsCIP13Map map[int]entities.Presentation) {
	m.medicaments = medicaments
	m.generiques = generiques
	m.medicamentsMap = medicamentsMap
	m.generiquesMap = generiquesMap
	m.presentationsCIP7Map = presentationsCIP7Map
	m.presentationsCIP13Map = presentationsCIP13Map
	m.lastUpdated = time.Now()
}

func (m *MockDataStore) BeginUpdate() bool {
	if m.updating {
		return false
	}
	m.updating = true
	return true
}

func (m *MockDataStore) EndUpdate() {
	m.updating = false
}

func (m *MockDataStore) GetServerStartTime() time.Time {
	return time.Time{} // Return zero time for mock
}

// MockParser implements Parser interface for testing
type MockParser struct {
	shouldFail bool
}

func (m *MockParser) ParseAllMedicaments() ([]entities.Medicament, map[int]entities.Presentation, map[int]entities.Presentation, error) {
	if m.shouldFail {
		return nil, nil, nil, &mockError{"parse failed"}
	}

	return []entities.Medicament{
			{Cis: 1, Denomination: "Test Medicament"},
			{Cis: 2, Denomination: "Another Test"},
		}, map[int]entities.Presentation{
			1234567: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
		}, map[int]entities.Presentation{
			3400912345678: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
		}, nil
}

func (m *MockParser) GeneriquesParser(medicaments *[]entities.Medicament, medicamentsMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.Generique, error) {
	if m.shouldFail {
		return nil, nil, &mockError{"generiques parse failed"}
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Test Generique"},
	}
	generiquesMap := map[int]entities.Generique{
		1: {Group: 1, Libelle: "Test Generique"},
	}

	return generiques, generiquesMap, nil
}

// MockScheduler implements Scheduler interface for testing
type MockScheduler struct {
	started bool
	stopped bool
}

func (m *MockScheduler) Start() error {
	if m.started {
		return &mockError{"already started"}
	}
	m.started = true
	return nil
}

func (m *MockScheduler) Stop() {
	m.stopped = true
}

// MockHTTPHandler implements HTTPHandler interface for testing
type MockHTTPHandler struct {
	responseCode int
	responseBody string
}

func (m *MockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServeAllMedicaments(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServePagedMedicaments(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindMedicament(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindMedicamentByID(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindMedicamentByCIP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindGeneriques(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindGeneriquesByGroupID(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServeMedicamentsV1(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServePresentationsV1(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServeGeneriquesV1(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

// MockHealthChecker implements HealthChecker interface for testing
type MockHealthChecker struct {
	status  string
	details map[string]any
	err     error
}

func (m *MockHealthChecker) HealthCheck() (string, map[string]any, error) {
	return m.status, m.details, m.err
}

func (m *MockHealthChecker) CalculateNextUpdate() time.Time {
	return time.Now().Add(1 * time.Hour)
}

// MockDataValidator implements DataValidator interface for testing
type MockDataValidator struct {
	shouldFail bool
}

func (m *MockDataValidator) ValidateMedicament(med *entities.Medicament) error {
	if m.shouldFail {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func (m *MockDataValidator) ValidateDataIntegrity(medicaments []entities.Medicament, generiques []entities.GeneriqueList) error {
	if m.shouldFail {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func (m *MockDataValidator) ValidateInput(input string) error {
	if m.shouldFail {
		return fmt.Errorf("input validation failed")
	}
	return nil
}

func (m *MockDataValidator) ValidateCIP(input string) (int, error) {
	if m.shouldFail {
		return -1, fmt.Errorf("CIP validation failed")
	}
	// Try to parse as int for simple validation
	val, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("input is not a number")
	}
	return val, nil
}

func (m *MockDataValidator) ValidateCIS(input string) (int, error) {
	if m.shouldFail {
		return -1, fmt.Errorf("CIS validation failed")
	}
	// Try to parse as int for simple validation
	val, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("input is not a number")
	}
	return val, nil
}

// mockError is a simple error type for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// Test functions demonstrating the benefits of interfaces

func TestDataStoreInterface(t *testing.T) {
	// We can easily test with a mock implementation
	store := &MockDataStore{
		medicaments: []entities.Medicament{{Cis: 1, Denomination: "Test"}},
	}

	medicaments := store.GetMedicaments()
	if len(medicaments) != 1 {
		t.Errorf("Expected 1 medicament, got %d", len(medicaments))
	}
}

func TestParserInterface(t *testing.T) {
	// Test successful parsing
	parser := &MockParser{shouldFail: false}
	medicaments, cip7Map, cip13Map, err := parser.ParseAllMedicaments()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(medicaments) != 2 {
		t.Errorf("Expected 2 medicaments, got %d", len(medicaments))
	}
	if len(cip7Map) != 1 {
		t.Errorf("Expected 1 CIP7 map entry, got %d", len(cip7Map))
	}
	if len(cip13Map) != 1 {
		t.Errorf("Expected 1 CIP13 map entry, got %d", len(cip13Map))
	}

	// Test failed parsing
	parser = &MockParser{shouldFail: true}
	_, _, _, err = parser.ParseAllMedicaments()
	if err == nil {
		t.Error("Expected error but got none")
	}
}

func TestSchedulerInterface(t *testing.T) {
	scheduler := &MockScheduler{}

	err := scheduler.Start()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !scheduler.started {
		t.Error("Scheduler should be started")
	}

	scheduler.Stop()
	if !scheduler.stopped {
		t.Error("Scheduler should be stopped")
	}
}

func TestHTTPHandlerInterface(t *testing.T) {
	handler := &MockHTTPHandler{
		responseCode: http.StatusOK,
		responseBody: "test response",
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "test response" {
		t.Errorf("Expected body 'test response', got '%s'", w.Body.String())
	}
}

func TestHealthCheckerInterface(t *testing.T) {
	checker := &MockHealthChecker{
		status: "healthy",
		details: map[string]any{
			"uptime": "1h",
			"memory": "50MB",
		},
	}

	status, details, err := checker.HealthCheck()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status)
	}

	if details["uptime"] != "1h" {
		t.Errorf("Expected uptime '1h', got '%v'", details["uptime"])
	}
}

func TestDataValidatorInterface(t *testing.T) {
	validator := &MockDataValidator{shouldFail: false}

	med := &entities.Medicament{Cis: 1, Denomination: "Test"}
	err := validator.ValidateMedicament(med)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test validation failure
	validator = &MockDataValidator{shouldFail: true}
	err = validator.ValidateMedicament(med)
	if err == nil {
		t.Error("Expected validation error but got none")
	}
}

// Example of how interfaces enable dependency injection
type Service struct {
	dataStore DataStore
	parser    Parser
	scheduler Scheduler
}

func NewService(dataStore DataStore, parser Parser, scheduler Scheduler) *Service {
	return &Service{
		dataStore: dataStore,
		parser:    parser,
		scheduler: scheduler,
	}
}

func (s *Service) GetMedicamentCount() int {
	return len(s.dataStore.GetMedicaments())
}

func TestServiceWithDependencyInjection(t *testing.T) {
	// We can easily test the service with mock dependencies
	mockStore := &MockDataStore{
		medicaments: []entities.Medicament{{Cis: 1}, {Cis: 2}},
	}
	mockParser := &MockParser{}
	mockScheduler := &MockScheduler{}

	service := NewService(mockStore, mockParser, mockScheduler)

	count := service.GetMedicamentCount()
	if count != 2 {
		t.Errorf("Expected 2 medicaments, got %d", count)
	}
}

// Compile-time checks to ensure our implementations implement the interfaces
func TestCompileTimeChecks(t *testing.T) {
	// These will fail to compile if the implementations don't match the interfaces
	var _ DataStore = (*MockDataStore)(nil)
	var _ Parser = (*MockParser)(nil)
	var _ Scheduler = (*MockScheduler)(nil)
	var _ HTTPHandler = (*MockHTTPHandler)(nil)
	var _ HealthChecker = (*MockHealthChecker)(nil)
	var _ DataValidator = (*MockDataValidator)(nil)
}
