package scheduler

import (
	"testing"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// MockDataStore for testing scheduler
type mockSchedulerDataStore struct {
	medicaments    []entities.Medicament
	generiques     []entities.GeneriqueList
	medicamentsMap map[int]entities.Medicament
	generiquesMap  map[int]entities.Generique
	lastUpdated    time.Time
	updating       bool
	updateCount    int
}

func (m *mockSchedulerDataStore) GetMedicaments() []entities.Medicament {
	return m.medicaments
}

func (m *mockSchedulerDataStore) GetGeneriques() []entities.GeneriqueList {
	return m.generiques
}

func (m *mockSchedulerDataStore) GetMedicamentsMap() map[int]entities.Medicament {
	return m.medicamentsMap
}

func (m *mockSchedulerDataStore) GetGeneriquesMap() map[int]entities.Generique {
	return m.generiquesMap
}

func (m *mockSchedulerDataStore) GetLastUpdated() time.Time {
	return m.lastUpdated
}

func (m *mockSchedulerDataStore) IsUpdating() bool {
	return m.updating
}

func (m *mockSchedulerDataStore) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList, medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.Generique) {
	m.medicaments = medicaments
	m.generiques = generiques
	m.medicamentsMap = medicamentsMap
	m.generiquesMap = generiquesMap
	m.lastUpdated = time.Now()
	m.updateCount++
}

func (m *mockSchedulerDataStore) BeginUpdate() bool {
	if m.updating {
		return false
	}
	m.updating = true
	return true
}

func (m *mockSchedulerDataStore) EndUpdate() {
	m.updating = false
}

// MockParser for testing scheduler
type mockSchedulerParser struct {
	parseCount int
	shouldFail bool
}

func (m *mockSchedulerParser) ParseAllMedicaments() ([]entities.Medicament, error) {
	m.parseCount++
	if m.shouldFail {
		return nil, &mockSchedulerError{"parse failed"}
	}

	return []entities.Medicament{
		{Cis: 1, Denomination: "Test Medicament"},
		{Cis: 2, Denomination: "Another Test"},
	}, nil
}

func (m *mockSchedulerParser) GeneriquesParser(medicaments *[]entities.Medicament, medicamentsMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.Generique, error) {
	if m.shouldFail {
		return nil, nil, &mockSchedulerError{"generiques parse failed"}
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Test Generique"},
	}
	generiquesMap := map[int]entities.Generique{
		1: {Group: 1, Libelle: "Test Generique"},
	}

	return generiques, generiquesMap, nil
}

type mockSchedulerError struct {
	msg string
}

func (e *mockSchedulerError) Error() string {
	return e.msg
}

func TestSchedulerWithDI_SuccessfulUpdate(t *testing.T) {
	// Create mock dependencies
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{shouldFail: false}

	// Create scheduler with dependency injection
	scheduler := NewSchedulerWithDI(mockDataStore, mockParser)

	// Test initial data load
	err := scheduler.Start()
	if err != nil {
		t.Errorf("Unexpected error during start: %v", err)
	}

	// Verify that data was updated
	if mockDataStore.updateCount != 1 {
		t.Errorf("Expected 1 update, got %d", mockDataStore.updateCount)
	}

	if mockParser.parseCount != 1 {
		t.Errorf("Expected 1 parse call, got %d", mockParser.parseCount)
	}

	// Verify data was stored correctly
	medicaments := mockDataStore.GetMedicaments()
	if len(medicaments) != 2 {
		t.Errorf("Expected 2 medicaments, got %d", len(medicaments))
	}

	generiques := mockDataStore.GetGeneriques()
	if len(generiques) != 1 {
		t.Errorf("Expected 1 generique, got %d", len(generiques))
	}

	// Clean up
	scheduler.Stop()
}

func TestSchedulerWithDI_ParseFailure(t *testing.T) {
	// Create mock dependencies that will fail
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{shouldFail: true}

	// Create scheduler with dependency injection
	scheduler := NewSchedulerWithDI(mockDataStore, mockParser)

	// Test initial data load failure
	err := scheduler.Start()
	if err == nil {
		t.Error("Expected error during start but got none")
	}

	// Verify that no data was updated due to failure
	if mockDataStore.updateCount != 0 {
		t.Errorf("Expected 0 updates due to failure, got %d", mockDataStore.updateCount)
	}
}

func TestSchedulerWithDI_ConcurrentUpdatePrevention(t *testing.T) {
	// Create mock dependencies
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{shouldFail: false}

	// Create scheduler with dependency injection
	scheduler := NewSchedulerWithDI(mockDataStore, mockParser)

	// Simulate an update in progress
	mockDataStore.BeginUpdate()

	// Try to start scheduler (should skip initial update)
	err := scheduler.Start()
	if err != nil {
		t.Errorf("Unexpected error during start with concurrent update: %v", err)
	}

	// Verify that no update occurred due to concurrent update
	if mockDataStore.updateCount != 0 {
		t.Errorf("Expected 0 updates due to concurrent update, got %d", mockDataStore.updateCount)
	}

	// Clean up
	scheduler.Stop()
}

// This test demonstrates how interfaces make testing much easier
// compared to testing the original scheduler which had tight coupling
func TestSchedulerWithDI_DependencyInjectionBenefits(t *testing.T) {
	// We can easily test with different implementations
	var dataStore interfaces.DataStore = &mockSchedulerDataStore{}
	var parser interfaces.Parser = &mockSchedulerParser{shouldFail: false}

	// The scheduler works with any implementation of the interfaces
	scheduler := NewSchedulerWithDI(dataStore, parser)

	// We can verify behavior without needing real data files, HTTP calls, etc.
	err := scheduler.Start()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Clean up
	scheduler.Stop()

	// This test runs in milliseconds instead of seconds/minutes
	// because we're using mocks instead of real implementations
}
