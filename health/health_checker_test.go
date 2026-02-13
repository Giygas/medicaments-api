package health

import (
	"net/http"
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

func TestHealthCheck_Healthy(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
			{Cis: 2, Denomination: "Test2"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status)
	}

	if httpStatus != http.StatusOK {
		t.Errorf("Expected HTTP status 200, got %d", httpStatus)
	}

	if data == nil {
		t.Error("Data should not be nil")
	}

	requiredFields := []string{"last_update", "data_age_hours", "medicaments", "generiques", "is_updating"}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Errorf("Data should contain '%s' field", field)
		}
	}
}

func TestHealthCheck_Unhealthy_NoMedicaments(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}

	if data["medicaments"] != 0 {
		t.Errorf("Expected 0 medicaments, got %v", data["medicaments"])
	}
}

func TestHealthCheck_Unhealthy_NoGeneriques(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques:  []entities.GeneriqueList{},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}

	if data["generiques"] != 0 {
		t.Errorf("Expected 0 generiques, got %v", data["generiques"])
	}
}

func TestHealthCheck_Unhealthy_NoData(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{},
		generiques:  []entities.GeneriqueList{},
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, _, httpStatus := healthChecker.HealthCheck()

	if status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}
}

func TestHealthCheck_Unhealthy_Stale48h(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-49 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}

	dataAge := data["data_age_hours"].(float64)
	if dataAge < 48 {
		t.Errorf("Expected data age >= 48 hours, got %f", dataAge)
	}
}

func TestHealthCheck_Degraded_24h(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-25 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "degraded" {
		t.Errorf("Expected status 'degraded', got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}

	dataAge := data["data_age_hours"].(float64)
	if dataAge < 24 {
		t.Errorf("Expected data age >= 24 hours, got %f", dataAge)
	}
}

func TestHealthCheck_Degraded_StuckUpdating(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-7 * time.Hour),
		isUpdating:  true,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "degraded" {
		t.Errorf("Expected status 'degraded', got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}

	dataAge := data["data_age_hours"].(float64)
	if dataAge < 6 {
		t.Errorf("Expected data age >= 6 hours for stuck update, got %f", dataAge)
	}
}

func TestHealthCheck_Boundary_Exactly24h(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-24 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "degraded" {
		t.Errorf("Expected status 'degraded' at exactly 24h, got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}

	dataAge := data["data_age_hours"].(float64)
	if dataAge < 23.9 || dataAge > 24.1 {
		t.Errorf("Expected data age ~24 hours, got %f", dataAge)
	}
}

func TestHealthCheck_Boundary_Exactly48h(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-48 * time.Hour),
		isUpdating:  false,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy' at exactly 48h, got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}

	dataAge := data["data_age_hours"].(float64)
	if dataAge < 47.9 || dataAge > 48.1 {
		t.Errorf("Expected data age ~48 hours, got %f", dataAge)
	}
}

func TestHealthCheck_Boundary_Exactly6hUpdating(t *testing.T) {
	mockDataStore := &MockHealthDataStore{
		medicaments: []entities.Medicament{
			{Cis: 1, Denomination: "Test1"},
		},
		generiques: []entities.GeneriqueList{
			{GroupID: 1, Libelle: "Gen1"},
		},
		lastUpdated: time.Now().Add(-6 * time.Hour),
		isUpdating:  true,
	}

	healthChecker := NewHealthChecker(mockDataStore)
	status, data, httpStatus := healthChecker.HealthCheck()

	if status != "degraded" {
		t.Errorf("Expected status 'degraded' at exactly 6h updating, got '%s'", status)
	}

	if httpStatus != http.StatusServiceUnavailable {
		t.Errorf("Expected HTTP status 503, got %d", httpStatus)
	}

	if data["is_updating"] != true {
		t.Errorf("Expected is_updating true, got %v", data["is_updating"])
	}
}

func TestHealthCheck_DataFields(t *testing.T) {
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
	_, data, _ := healthChecker.HealthCheck()

	requiredFields := []string{"last_update", "data_age_hours", "medicaments", "generiques", "is_updating"}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Errorf("Data should contain '%s' field", field)
		}
	}
}

func TestHealthCheck_HTTPStatus(t *testing.T) {
	testCases := []struct {
		name           string
		medicaments    []entities.Medicament
		generiques     []entities.GeneriqueList
		lastUpdated    time.Time
		isUpdating     bool
		expectedStatus string
		expectedHTTP   int
	}{
		{
			name:           "healthy returns 200",
			medicaments:    []entities.Medicament{{Cis: 1}},
			generiques:     []entities.GeneriqueList{{GroupID: 1}},
			lastUpdated:    time.Now().Add(-1 * time.Hour),
			isUpdating:     false,
			expectedStatus: "healthy",
			expectedHTTP:   http.StatusOK,
		},
		{
			name:           "degraded returns 503",
			medicaments:    []entities.Medicament{{Cis: 1}},
			generiques:     []entities.GeneriqueList{{GroupID: 1}},
			lastUpdated:    time.Now().Add(-25 * time.Hour),
			isUpdating:     false,
			expectedStatus: "degraded",
			expectedHTTP:   http.StatusServiceUnavailable,
		},
		{
			name:           "unhealthy returns 503",
			medicaments:    []entities.Medicament{{Cis: 1}},
			generiques:     []entities.GeneriqueList{{GroupID: 1}},
			lastUpdated:    time.Now().Add(-49 * time.Hour),
			isUpdating:     false,
			expectedStatus: "unhealthy",
			expectedHTTP:   http.StatusServiceUnavailable,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockDataStore := &MockHealthDataStore{
				medicaments: tt.medicaments,
				generiques:  tt.generiques,
				lastUpdated: tt.lastUpdated,
				isUpdating:  tt.isUpdating,
			}

			healthChecker := NewHealthChecker(mockDataStore)
			status, _, httpStatus := healthChecker.HealthCheck()

			if status != tt.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, status)
			}

			if httpStatus != tt.expectedHTTP {
				t.Errorf("Expected HTTP status %d, got %d", tt.expectedHTTP, httpStatus)
			}
		})
	}
}

func TestCalculateNextUpdate_Before6AM(t *testing.T) {
	mockDataStore := &MockHealthDataStore{}
	healthChecker := NewHealthChecker(mockDataStore)

	now := time.Now()

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

	nextUpdate := healthChecker.CalculateNextUpdate()

	now := time.Now()
	sixAM := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
	tomorrowSixAM := sixAM.AddDate(0, 0, 1)

	validTimes := []time.Time{sixAM, sixPM, tomorrowSixAM}

	valid := slices.ContainsFunc(validTimes, nextUpdate.Equal)

	if !valid {
		t.Errorf("Next update time %v is not valid (expected 6AM today, 6PM today, or 6AM tomorrow)", nextUpdate)
	}
}

func TestCalculateNextUpdate_After6PM(t *testing.T) {
	mockDataStore := &MockHealthDataStore{}
	healthChecker := NewHealthChecker(mockDataStore)

	nextUpdate := healthChecker.CalculateNextUpdate()

	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	expected := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 6, 0, 0, 0, tomorrow.Location())

	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
	if now.After(sixPM) {
		if !nextUpdate.Equal(expected) {
			t.Errorf("Expected next update at %v, got %v", expected, nextUpdate)
		}
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	mockDataStore := &MockHealthDataStore{
		medicaments: make([]entities.Medicament, 1000),
		generiques:  make([]entities.GeneriqueList, 100),
		lastUpdated: time.Now().Add(-1 * time.Hour),
		isUpdating:  false,
	}

	for i := 0; i < 1000; i++ {
		mockDataStore.medicaments[i] = entities.Medicament{Cis: i, Denomination: "Test"}
	}

	for i := 0; i < 100; i++ {
		mockDataStore.generiques[i] = entities.GeneriqueList{GroupID: i, Libelle: "Test"}
	}

	healthChecker := NewHealthChecker(mockDataStore)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = healthChecker.HealthCheck()
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
