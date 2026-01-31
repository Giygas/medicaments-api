package data

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestDataContainer_EdgeCases(t *testing.T) {
	container := NewDataContainer()

	if container == nil {
		t.Fatal("NewDataContainer returned nil")
	}

	// Verify all atomic values are initialized
	if container.GetMedicaments() == nil {
		t.Error("Medicaments should not be nil")
	}
	if container.GetGeneriques() == nil {
		t.Error("Generiques should not be nil")
	}
	if container.GetMedicamentsMap() == nil {
		t.Error("MedicamentsMap should not be nil")
	}
	if container.GetGeneriquesMap() == nil {
		t.Error("GeneriquesMap should not be nil")
	}
	if container.GetPresentationsCIP7Map() == nil {
		t.Error("PresentationsCIP7Map should not be nil")
	}
	if container.GetPresentationsCIP13Map() == nil {
		t.Error("PresentationsCIP13Map should not be nil")
	}
}

func TestDataContainer_GetServerStartTime(t *testing.T) {
	container := NewDataContainer()

	// Initially should be zero time
	startTime := container.GetServerStartTime()
	if !startTime.IsZero() {
		t.Error("Server start time should initially be zero")
	}

	// Set a start time
	now := time.Now()
	container.SetServerStartTime(now)

	// Verify it was set
	retrievedTime := container.GetServerStartTime()
	if retrievedTime.IsZero() {
		t.Error("Server start time should not be zero after being set")
	}
	if !retrievedTime.Equal(now) {
		t.Errorf("Expected start time %v, got %v", now, retrievedTime)
	}
}

func TestDataContainer_BeginEndUpdate(t *testing.T) {
	container := NewDataContainer()

	// Initially should not be updating
	if container.IsUpdating() {
		t.Error("Container should not be updating initially")
	}

	// Begin update
	begin := container.BeginUpdate()
	if !begin {
		t.Error("BeginUpdate should return true when not updating")
	}
	if !container.IsUpdating() {
		t.Error("IsUpdating should return true after BeginUpdate")
	}

	// Second BeginUpdate should fail
	begin2 := container.BeginUpdate()
	if begin2 {
		t.Error("Second BeginUpdate should return false when already updating")
	}

	// End update
	container.EndUpdate()

	if container.IsUpdating() {
		t.Error("IsUpdating should return false after EndUpdate")
	}

	// Can begin update again
	begin3 := container.BeginUpdate()
	if !begin3 {
		t.Error("BeginUpdate should return true after EndUpdate")
	}

	container.EndUpdate()
}

func TestDataContainer_ConcurrentReads(t *testing.T) {
	container := NewDataContainer()

	// Add some data
	medicaments := []entities.Medicament{
		{Cis: 1, Denomination: "Test 1"},
		{Cis: 2, Denomination: "Test 2"},
	}
	generiques := []entities.GeneriqueList{}
	medicamentsMap := map[int]entities.Medicament{
		1: {Cis: 1, Denomination: "Test 1"},
		2: {Cis: 2, Denomination: "Test 2"},
	}
	generiquesMap := map[int]entities.GeneriqueList{}
	presentationsCIP7Map := map[int]entities.Presentation{}
	presentationsCIP13Map := map[int]entities.Presentation{}

	container.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
		presentationsCIP7Map, presentationsCIP13Map)

	// Concurrent reads
	var wg sync.WaitGroup
	numReaders := 100

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Access all data
			_ = container.GetMedicaments()
			_ = container.GetGeneriques()
			_ = container.GetMedicamentsMap()
			_ = container.GetGeneriquesMap()
			_ = container.GetPresentationsCIP7Map()
			_ = container.GetPresentationsCIP13Map()
			_ = container.GetLastUpdated()
			_ = container.IsUpdating()
		}()
	}

	wg.Wait()

	// If we got here without panic/deadlock, the test passed
	t.Logf("Successfully performed %d concurrent reads", numReaders)
}

func TestDataContainer_ConcurrentReadsDuringUpdate(t *testing.T) {
	container := NewDataContainer()

	// Add initial data
	medicaments := []entities.Medicament{{Cis: 1, Denomination: "Test 1"}}
	generiques := []entities.GeneriqueList{}
	medicamentsMap := map[int]entities.Medicament{1: {Cis: 1, Denomination: "Test 1"}}
	generiquesMap := map[int]entities.GeneriqueList{}
	presentationsCIP7Map := map[int]entities.Presentation{}
	presentationsCIP13Map := map[int]entities.Presentation{}

	container.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
		presentationsCIP7Map, presentationsCIP13Map)

	// Begin update
	container.BeginUpdate()

	// Concurrent reads during update (should see old data)
	var wg sync.WaitGroup
	numReaders := 50
	var sawOldData atomic.Bool
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			meds := container.GetMedicaments()
			if len(meds) == 0 {
				sawOldData.Store(false)
			} else {
				sawOldData.Store(true)
			}
		}()
	}

	wg.Wait()

	// End update
	container.EndUpdate()

	// Verify no data race or panic
	t.Logf("Successfully performed %d concurrent reads during update", numReaders)
}

func TestDataContainer_UpdateDataWithNil(t *testing.T) {
	container := NewDataContainer()

	// Update with nil medicaments
	container.UpdateData(nil, nil, nil, nil, nil, nil)

	// Get data - should return empty slices (not nil) for safety
	medicaments := container.GetMedicaments()
	if len(medicaments) != 0 {
		t.Errorf("Expected 0 medicaments after nil update, got %d", len(medicaments))
	}

	generiques := container.GetGeneriques()
	if len(generiques) != 0 {
		t.Errorf("Expected 0 generiques after nil update, got %d", len(generiques))
	}

	// Maps should return empty maps (not nil) for safety
	medicamentsMap := container.GetMedicamentsMap()
	if len(medicamentsMap) != 0 {
		t.Errorf("Expected 0 map entries after nil update, got %d", len(medicamentsMap))
	}

	generiquesMap := container.GetGeneriquesMap()
	if len(generiquesMap) != 0 {
		t.Errorf("Expected 0 generique map entries after nil update, got %d", len(generiquesMap))
	}
}

func TestDataContainer_UpdateDataWithEmptySlices(t *testing.T) {
	container := NewDataContainer()

	// Update with empty slices
	container.UpdateData([]entities.Medicament{}, []entities.GeneriqueList{}, map[int]entities.Medicament{}, map[int]entities.GeneriqueList{}, map[int]entities.Presentation{}, map[int]entities.Presentation{})

	// Verify data was stored
	if len(container.GetMedicaments()) != 0 {
		t.Error("Expected empty medicaments slice")
	}
	if len(container.GetGeneriques()) != 0 {
		t.Error("Expected empty generiques slice")
	}
	if len(container.GetMedicamentsMap()) != 0 {
		t.Error("Expected empty medicaments map")
	}
	if len(container.GetGeneriquesMap()) != 0 {
		t.Error("Expected empty generiques map")
	}
}

func TestDataContainer_ThreadSafety(t *testing.T) {
	container := NewDataContainer()

	medicaments := []entities.Medicament{
		{Cis: 1, Denomination: "Test 1"},
		{Cis: 2, Denomination: "Test 2"},
	}
	generiques := []entities.GeneriqueList{}
	medicamentsMap := map[int]entities.Medicament{
		1: {Cis: 1, Denomination: "Test 1"},
		2: {Cis: 2, Denomination: "Test 2"},
	}
	generiquesMap := map[int]entities.GeneriqueList{}
	presentationsCIP7Map := map[int]entities.Presentation{}
	presentationsCIP13Map := map[int]entities.Presentation{}

	// Concurrent updates and reads
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Begin update
			if !container.BeginUpdate() {
				return // Skip if another update is in progress
			}
			defer container.EndUpdate()

			// Update data
			newMedicaments := make([]entities.Medicament, len(medicaments))
			copy(newMedicaments, medicaments)
			newMedicaments[0].Cis = id + 100

			container.UpdateData(newMedicaments, generiques, medicamentsMap, generiquesMap,
				presentationsCIP7Map, presentationsCIP13Map)

			// Read data
			_ = container.GetMedicaments()
			_ = container.GetMedicamentsMap()
		}(i)
	}

	wg.Wait()

	// If we got here without panic/deadlock, the test passed
	t.Log("Successfully performed 20 concurrent update/read cycles")
}

func TestDataContainer_GetLastUpdated(t *testing.T) {
	container := NewDataContainer()

	// Initially should be zero time
	lastUpdated := container.GetLastUpdated()
	if !lastUpdated.IsZero() {
		t.Error("Last updated should initially be zero time")
	}

	// Update data (which sets last updated)
	medicaments := []entities.Medicament{{Cis: 1, Denomination: "Test"}}
	generiques := []entities.GeneriqueList{}
	medicamentsMap := map[int]entities.Medicament{1: {Cis: 1, Denomination: "Test"}}
	generiquesMap := map[int]entities.GeneriqueList{}
	presentationsCIP7Map := map[int]entities.Presentation{}
	presentationsCIP13Map := map[int]entities.Presentation{}

	container.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
		presentationsCIP7Map, presentationsCIP13Map)

	// Should now have a time
	lastUpdated = container.GetLastUpdated()
	if lastUpdated.IsZero() {
		t.Error("Last updated should be set after data update")
	}

	// Verify it's recent (within last second)
	if time.Since(lastUpdated) > time.Second {
		t.Errorf("Last updated time too old: %v", lastUpdated)
	}
}
