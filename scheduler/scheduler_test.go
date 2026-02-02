package scheduler

import (
	"testing"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// MockDataStore for testing scheduler
type mockSchedulerDataStore struct {
	medicaments           []entities.Medicament
	generiques            []entities.GeneriqueList
	medicamentsMap        map[int]entities.Medicament
	generiquesMap         map[int]entities.GeneriqueList
	presentationsCIP7Map  map[int]entities.Presentation
	presentationsCIP13Map map[int]entities.Presentation
	lastUpdated           time.Time
	updating              bool
	updateCount           int
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

func (m *mockSchedulerDataStore) GetGeneriquesMap() map[int]entities.GeneriqueList {
	return m.generiquesMap
}

func (m *mockSchedulerDataStore) GetPresentationsCIP7Map() map[int]entities.Presentation {
	return m.presentationsCIP7Map
}

func (m *mockSchedulerDataStore) GetPresentationsCIP13Map() map[int]entities.Presentation {
	return m.presentationsCIP13Map
}

func (m *mockSchedulerDataStore) GetLastUpdated() time.Time {
	return m.lastUpdated
}

func (m *mockSchedulerDataStore) IsUpdating() bool {
	return m.updating
}

func (m *mockSchedulerDataStore) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList, medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.GeneriqueList, presentationsCIP7Map map[int]entities.Presentation, presentationsCIP13Map map[int]entities.Presentation, report *interfaces.DataQualityReport) {
	m.medicaments = medicaments
	m.generiques = generiques
	m.medicamentsMap = medicamentsMap
	m.generiquesMap = generiquesMap
	m.presentationsCIP7Map = presentationsCIP7Map
	m.presentationsCIP13Map = presentationsCIP13Map
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

func (m *mockSchedulerDataStore) GetServerStartTime() time.Time {
	return time.Time{} // Return zero time for mock
}

func (m *mockSchedulerDataStore) GetDataQualityReport() *interfaces.DataQualityReport {
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

// MockParser for testing scheduler
type mockSchedulerParser struct {
	parseCount int
	shouldFail bool
	// Configurable presentation maps for testing
	cip7Map  map[int]entities.Presentation
	cip13Map map[int]entities.Presentation
}

func (m *mockSchedulerParser) ParseAllMedicaments() ([]entities.Medicament, map[int]entities.Presentation, map[int]entities.Presentation, error) {
	m.parseCount++
	if m.shouldFail {
		return nil, nil, nil, &mockSchedulerError{"parse failed"}
	}

	// Use configured presentation maps if available, otherwise use default
	cip7Map := m.cip7Map
	cip13Map := m.cip13Map

	if cip7Map == nil {
		cip7Map = map[int]entities.Presentation{
			1234567: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
		}
	}

	if cip13Map == nil {
		cip13Map = map[int]entities.Presentation{
			3400912345678: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
		}
	}

	return []entities.Medicament{
		{Cis: 1, Denomination: "Test Medicament"},
		{Cis: 2, Denomination: "Another Test"},
	}, cip7Map, cip13Map, nil
}

func (m *mockSchedulerParser) GeneriquesParser(medicaments *[]entities.Medicament, medicamentsMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.GeneriqueList, error) {
	if m.shouldFail {
		return nil, nil, &mockSchedulerError{"generiques parse failed"}
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Test Generique"},
	}
	generiquesMap := map[int]entities.GeneriqueList{
		1: {GroupID: 1, Libelle: "Test Generique"},
	}

	return generiques, generiquesMap, nil
}

type mockSchedulerError struct {
	msg string
}

func (e *mockSchedulerError) Error() string {
	return e.msg
}

func TestScheduler_SuccessfulUpdate(t *testing.T) {
	// Create mock dependencies
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{shouldFail: false}

	// Create scheduler with dependency injection
	scheduler := NewScheduler(mockDataStore, mockParser)

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

func TestScheduler_ParseFailure(t *testing.T) {
	// Create mock dependencies that will fail
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{shouldFail: true}

	// Create scheduler with dependency injection
	scheduler := NewScheduler(mockDataStore, mockParser)

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

func TestScheduler_ConcurrentUpdatePrevention(t *testing.T) {
	// Create mock dependencies
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{shouldFail: false}

	// Create scheduler with dependency injection
	scheduler := NewScheduler(mockDataStore, mockParser)

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
func TestScheduler_DependencyInjectionBenefits(t *testing.T) {
	// We can easily test with different implementations
	var dataStore interfaces.DataStore = &mockSchedulerDataStore{}
	var parser interfaces.Parser = &mockSchedulerParser{shouldFail: false}

	// The scheduler works with any implementation of the interfaces
	scheduler := NewScheduler(dataStore, parser)

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

func TestScheduler_MultiplePresentations(t *testing.T) {
	// Test that multiple presentations are all stored correctly
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{
		shouldFail: false,
		// Override to return multiple presentations
		cip7Map: map[int]entities.Presentation{
			1234567: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
			2345678: {Cis: 2, Cip7: 2345678, Cip13: 34009234567890},
			3456789: {Cis: 3, Cip7: 3456789, Cip13: 3400934567890},
		},
		cip13Map: map[int]entities.Presentation{
			3400912345678:  {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
			34009234567890: {Cis: 2, Cip7: 2345678, Cip13: 34009234567890},
			3400934567890:  {Cis: 3, Cip7: 3456789, Cip13: 3400934567890},
		},
	}

	// Create scheduler with dependency injection
	scheduler := NewScheduler(mockDataStore, mockParser)

	// Test initial data load
	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Unexpected error during start: %v", err)
	}

	// Verify all presentations are stored
	cip7Map := mockDataStore.GetPresentationsCIP7Map()
	cip13Map := mockDataStore.GetPresentationsCIP13Map()

	if len(cip7Map) != 3 {
		t.Errorf("Expected 3 CIP7 entries, got %d", len(cip7Map))
	}
	if len(cip13Map) != 3 {
		t.Errorf("Expected 3 CIP13 entries, got %d", len(cip13Map))
	}

	// Verify specific entries exist
	expectedCIP7s := []int{1234567, 2345678, 3456789}
	for _, cip7 := range expectedCIP7s {
		if _, exists := cip7Map[cip7]; !exists {
			t.Errorf("CIP7 %d not found in map", cip7)
		}
	}

	// Verify all presentations have correct CIS values
	cip7Pres123 := cip7Map[1234567]
	if cip7Pres123.Cis != 1 {
		t.Errorf("Presentation for CIP7 1234567 should have CIS=1, got %d", cip7Pres123.Cis)
	}

	cip7Pres234 := cip7Map[2345678]
	if cip7Pres234.Cis != 2 {
		t.Errorf("Presentation for CIP7 2345678 should have CIS=2, got %d", cip7Pres234.Cis)
	}

	cip7Pres345 := cip7Map[3456789]
	if cip7Pres345.Cis != 3 {
		t.Errorf("Presentation for CIP7 3456789 should have CIS=3, got %d", cip7Pres345.Cis)
	}

	// Clean up
	scheduler.Stop()
}

func TestScheduler_UpdateOverridesMaps(t *testing.T) {
	// Test that subsequent updates properly replace old maps
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{
		shouldFail: false,
		// First update will have these presentations
		cip7Map: map[int]entities.Presentation{
			1111111: {Cis: 1, Cip7: 1111111, Cip13: 3400900000001},
		},
		cip13Map: map[int]entities.Presentation{
			3400900000001: {Cis: 1, Cip7: 1111111, Cip13: 3400900000001},
		},
	}

	// Create scheduler with dependency injection
	scheduler := NewScheduler(mockDataStore, mockParser)

	// First update
	err := scheduler.Start()
	if err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	// Verify first maps are stored
	cip7Map1 := mockDataStore.GetPresentationsCIP7Map()
	if _, exists := cip7Map1[1111111]; !exists {
		t.Error("First CIP7 map should contain 1111111")
	}

	// Second update with different data
	mockParser.cip7Map = map[int]entities.Presentation{
		2222222: {Cis: 2, Cip7: 2222222, Cip13: 3400900000002},
	}
	mockParser.cip13Map = map[int]entities.Presentation{
		3400900000002: {Cis: 2, Cip7: 2222222, Cip13: 3400900000002},
	}

	// Trigger second update
	_ = scheduler.updateData()

	// Verify maps were replaced (not merged)
	cip7Map2 := mockDataStore.GetPresentationsCIP7Map()
	if _, exists := cip7Map2[1111111]; exists {
		t.Error("Old CIP7 entry should be replaced")
	}
	if _, exists := cip7Map2[2222222]; !exists {
		t.Error("New CIP7 entry should exist")
	}

	// Clean up
	scheduler.Stop()
}

func TestScheduler_PresentationMapsStored(t *testing.T) {
	// Test that CIP7 and CIP13 maps are properly stored
	mockDataStore := &mockSchedulerDataStore{}
	mockParser := &mockSchedulerParser{
		shouldFail: false,
	}

	// Create scheduler with dependency injection
	scheduler := NewScheduler(mockDataStore, mockParser)

	// Test initial data load
	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Unexpected error during start: %v", err)
	}

	// Verify that maps were stored
	cip7Map := mockDataStore.GetPresentationsCIP7Map()
	cip13Map := mockDataStore.GetPresentationsCIP13Map()

	if len(cip7Map) != 1 {
		t.Errorf("Expected 1 CIP7 map entry, got %d", len(cip7Map))
	}

	if len(cip13Map) != 1 {
		t.Errorf("Expected 1 CIP13 map entry, got %d", len(cip13Map))
	}

	// Verify the actual presentation data exists
	expectedCIP7 := 1234567
	expectedCIP13 := 3400912345678

	if _, exists := cip7Map[expectedCIP7]; !exists {
		t.Errorf("CIP7 %d not found in map", expectedCIP7)
	}

	if _, exists := cip13Map[expectedCIP13]; !exists {
		t.Errorf("CIP13 %d not found in map", expectedCIP13)
	}

	// Verify presentation data has correct values
	cip7Presentation := cip7Map[expectedCIP7]
	if cip7Presentation.Cip7 != expectedCIP7 {
		t.Errorf("CIP7 presentation has wrong Cip7: got %d, want %d",
			cip7Presentation.Cip7, expectedCIP7)
	}

	cip13Presentation := cip13Map[expectedCIP13]
	if cip13Presentation.Cip13 != expectedCIP13 {
		t.Errorf("CIP13 presentation has wrong Cip13: got %d, want %d",
			cip13Presentation.Cip13, expectedCIP13)
	}

	// Clean up
	scheduler.Stop()
}
