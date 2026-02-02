package health

import (
	"slices"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// MockDataStore for testing
type MockHealthDataStore struct {
	medicaments           []entities.Medicament
	generiques            []entities.GeneriqueList
	presentationsCIP7Map  map[int]entities.Presentation
	presentationsCIP13Map map[int]entities.Presentation
	lastUpdated           time.Time
	isUpdating            bool
	shouldFail            bool
}

func (m *MockHealthDataStore) GetMedicaments() []entities.Medicament {
	if m.shouldFail {
		return nil
	}
	return m.medicaments
}

func (m *MockHealthDataStore) GetGeneriques() []entities.GeneriqueList {
	if m.shouldFail {
		return nil
	}
	return m.generiques
}

func (m *MockHealthDataStore) GetMedicamentsMap() map[int]entities.Medicament {
	return make(map[int]entities.Medicament)
}

func (m *MockHealthDataStore) GetGeneriquesMap() map[int]entities.GeneriqueList {
	return make(map[int]entities.GeneriqueList)
}

func (m *MockHealthDataStore) GetPresentationsCIP7Map() map[int]entities.Presentation {
	return m.presentationsCIP7Map
}

func (m *MockHealthDataStore) GetPresentationsCIP13Map() map[int]entities.Presentation {
	return m.presentationsCIP13Map
}

func (m *MockHealthDataStore) GetLastUpdated() time.Time {
	return m.lastUpdated
}

func (m *MockHealthDataStore) IsUpdating() bool {
	return m.isUpdating
}

func (m *MockHealthDataStore) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList, medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.GeneriqueList, presentionsCIP7Map map[int]entities.Presentation, presentionsCIP13Map map[int]entities.Presentation, report *interfaces.DataQualityReport) {
	// Not used in health tests
}

func (m *MockHealthDataStore) BeginUpdate() bool {
	return true
}

func (m *MockHealthDataStore) EndUpdate() {
	// Not used in health tests
}

func (m *MockHealthDataStore) GetServerStartTime() time.Time {
	return time.Time{} // Return zero time for mock
}

func (m *MockHealthDataStore) GetDataQualityReport() *interfaces.DataQualityReport {
	return &interfaces.DataQualityReport{
		DuplicateCIS:                       []int{},
		DuplicateGroupIDs:                  []int{},
		MedicamentsWithoutConditions:       0,
		MedicamentsWithoutGeneriques:       0,
		MedicamentsWithoutPresentations:    0,
		MedicamentsWithoutCompositions:     0,
		GeneriqueOnlyCIS:                   0,
		MedicamentsWithoutConditionsCIS:    []int{},
		MedicamentsWithoutGeneriquesCIS:    []int{},
		MedicamentsWithoutPresentationsCIS: []int{},
		MedicamentsWithoutCompositionsCIS:  []int{},
		GeneriqueOnlyCISList:               []int{},
	}
}

func TestNewHealthChecker(t *testing.T) {
	mockDataStore := &MockHealthDataStore{}

	healthChecker := NewHealthChecker(mockDataStore)

	if healthChecker == nil {
		t.Fatal("NewHealthChecker returned nil")
	}

	// Type assertion to verify it's the correct type
	if _, ok := healthChecker.(*HealthCheckerImpl); !ok {
		t.Error("NewHealthChecker should return *HealthCheckerImpl")
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	// Setup mock with recent data
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
			{Cis: 2, Denomination: "Test2"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-1 * time.Hour), // Recent data
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, details, err := healthChecker.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	if status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status)
	}

	if details == nil {
		t.Error("Details should not be nil")
	}

	// Check required fields
	if _, ok := details["last_update"]; !ok {
		t.Error("Details should contain 'last_update'")
	}

	if _, ok := details["data_age_hours"]; !ok {
		t.Error("Details should contain 'data_age_hours'")
	}

	if _, ok := details["data"]; !ok {
		t.Error("Details should contain 'data'")
	}

	if _, ok := details["system"]; !ok {
		t.Error("Details should contain 'system'")
	}

	// Check data section
	data := details["data"].(map[string]any)
	if data["medicaments"] != 2 {
		t.Errorf("Expected 2 medicaments, got %v", data["medicaments"])
	}

	if data["generiques"] != 1 {
		t.Errorf("Expected 1 generique, got %v", data["generiques"])
	}

	if data["is_updating"] != false {
		t.Errorf("Expected is_updating false, got %v", data["is_updating"])
	}

	// Check system section
	system := details["system"].(map[string]any)
	if system["goroutines"] == nil {
		t.Error("System should contain goroutines count")
	}

	if _, ok := system["memory"]; !ok {
		t.Error("System should contain memory info")
	}
}

func TestHealthCheck_Unhealthy_NoData(t *testing.T) {
	// Setup mock with no medicaments
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{}, // Empty
		generiques:  []entities.GeneriqueList{},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, details, err := healthChecker.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	if status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", status)
	}

	if details == nil {
		t.Error("Details should not be nil")
	}
}

func TestHealthCheck_Degraded_OldData(t *testing.T) {
	// Setup mock with old data (>24 hours)
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-25 * time.Hour), // Old data
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, details, err := healthChecker.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	if status != "degraded" {
		t.Errorf("Expected status 'degraded', got '%s'", status)
	}

	if details == nil {
		t.Error("Details should not be nil")
	}

	// Check data age
	dataAge := details["data_age_hours"].(float64)
	if dataAge < 24 {
		t.Errorf("Expected data age > 24 hours, got %f", dataAge)
	}
}

func TestHealthCheck_Updating(t *testing.T) {
	// Setup mock with updating flag
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  true,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, details, err := healthChecker.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	if status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status)
	}

	// Check is_updating flag
	data := details["data"].(map[string]any)
	if data["is_updating"] != true {
		t.Errorf("Expected is_updating true, got %v", data["is_updating"])
	}
}

func TestCalculateNextUpdate_Before6AM(t *testing.T) {
	mockDataStore := &MockHealthDataStore{}
	healthChecker := NewHealthChecker(mockDataStore)

	now := time.Now()

	// Calculate what the next update should be based on current time
	nextUpdate := healthChecker.CalculateNextUpdate()

	sixAM := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())

	var expected time.Time
	if now.Before(sixAM) {
		expected = sixAM
	} else if now.Before(sixPM) {
		expected = sixPM
	} else {
		tomorrow := now.AddDate(0, 0, 1)
		expected = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 6, 0, 0, 0, tomorrow.Location())
	}

	if !nextUpdate.Equal(expected) {
		t.Errorf("Expected next update at %v, got %v", expected, nextUpdate)
	}
}

func TestCalculateNextUpdate_Between6AMAnd6PM(t *testing.T) {
	mockDataStore := &MockHealthDataStore{}
	healthChecker := NewHealthChecker(mockDataStore)

	// This test is tricky without time mocking, but we can verify the logic
	// by checking that the result is either 6 AM today, 6 PM today, or 6 AM tomorrow
	nextUpdate := healthChecker.CalculateNextUpdate()

	now := time.Now()
	sixAM := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
	tomorrowSixAM := sixAM.AddDate(0, 0, 1)

	// Next update should be one of these times depending on current time
	validTimes := []time.Time{sixAM, sixPM, tomorrowSixAM}

	valid := slices.ContainsFunc(validTimes, nextUpdate.Equal)

	if !valid {
		t.Errorf("Next update time %v is not valid (expected 6AM today, 6PM today, or 6AM tomorrow)", nextUpdate)
	}
}

func TestCalculateNextUpdate_After6PM(t *testing.T) {
	mockDataStore := &MockHealthDataStore{}
	healthChecker := NewHealthChecker(mockDataStore)

	// This test verifies that after 6PM, next update is tomorrow 6AM
	nextUpdate := healthChecker.CalculateNextUpdate()

	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	expected := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 6, 0, 0, 0, tomorrow.Location())

	// Only check if current time is actually after 6PM
	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
	if now.After(sixPM) {
		if !nextUpdate.Equal(expected) {
			t.Errorf("Expected next update at %v, got %v", expected, nextUpdate)
		}
	}
}

func TestHealthCheck_MemoryStats(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	_, details, err := healthChecker.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	// Check memory stats
	system := details["system"].(map[string]any)
	memory := system["memory"].(map[string]any)

	// Check required memory fields
	requiredFields := []string{"alloc_mb", "total_alloc_mb", "sys_mb", "num_gc"}
	for _, field := range requiredFields {
		if _, ok := memory[field]; !ok {
			t.Errorf("Memory stats should contain '%s'", field)
		}
	}

	// Check that values are reasonable
	allocMB := memory["alloc_mb"].(int)
	if allocMB < 0 {
		t.Error("Alloc memory should be non-negative")
	}

	numGC := memory["num_gc"].(uint32)
	if numGC > 100000 {
		t.Logf("High GC count detected: %d (may indicate memory pressure)", numGC)
	}
}

func TestHealthCheck_GoroutineCount(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	_, details, err := healthChecker.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	// Check goroutine count
	system := details["system"].(map[string]any)
	goroutines := system["goroutines"].(int)

	if goroutines <= 0 {
		t.Error("Goroutine count should be positive")
	}
}

func TestHealthCheck_NextUpdateField(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	_, details, err := healthChecker.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	// Check next_update field
	data := details["data"].(map[string]any)
	nextUpdate := data["next_update"].(string)

	if nextUpdate == "" {
		t.Error("Next update should not be empty")
	}

	// Try to parse the time to ensure it's valid RFC3339 format
	_, parseErr := time.Parse(time.RFC3339, nextUpdate)
	if parseErr != nil {
		t.Errorf("Next update should be valid RFC3339 format: %v", parseErr)
	}
}

func TestHealthCheck_ZeroTimeLastUpdate(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Time{}, // Zero time
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, details, err := healthChecker.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	// With zero time, data age will be very large, should be degraded
	if status != "degraded" {
		t.Errorf("Expected status 'degraded' with zero last update, got '%s'", status)
	}

	// Check data age
	dataAge := details["data_age_hours"].(float64)
	if dataAge < 24 {
		t.Errorf("Expected data age > 24 hours with zero time, got %f", dataAge)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	mockDataStore := &MockHealthDataStore{
		medicaments: make([]entities.Medicament, 1000),
		generiques:  make([]entities.GeneriqueList, 100),
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	// Initialize medicaments
	for i := 0; i < 1000; i++ {
		mockDataStore.medicaments[i] = entities.Medicament{Cis: i, Denomination: "Test"}
	}

	// Initialize generiques
	for i := 0; i < 100; i++ {
		mockDataStore.generiques[i] = entities.GeneriqueList{GroupID: i, Libelle: "Test"}
	}

	healthChecker := NewHealthChecker(mockDataStore)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := healthChecker.HealthCheck()
		if err != nil {
			b.Logf("HealthCheck failed: %v", err)
		}
	}
}

func BenchmarkCalculateNextUpdate(b *testing.B) {
	mockDataStore := &MockHealthDataStore{}
	healthChecker := NewHealthChecker(mockDataStore)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		healthChecker.CalculateNextUpdate()
	}
}
