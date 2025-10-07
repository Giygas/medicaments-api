package data

import (
	"sync"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

func TestNewDataContainer(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	if dc == nil {
		t.Fatal("NewDataContainer returned nil")
	}

	// Test initial state
	if dc.IsUpdating() {
		t.Error("NewDataContainer should not be updating")
	}

	if !dc.GetLastUpdated().IsZero() {
		t.Error("NewDataContainer should have zero lastUpdated time")
	}

	if len(dc.GetMedicaments()) != 0 {
		t.Error("NewDataContainer should have empty medicaments")
	}

	if len(dc.GetGeneriques()) != 0 {
		t.Error("NewDataContainer should have empty generiques")
	}

	if len(dc.GetMedicamentsMap()) != 0 {
		t.Error("NewDataContainer should have empty medicaments map")
	}

	if len(dc.GetGeneriquesMap()) != 0 {
		t.Error("NewDataContainer should have empty generiques map")
	}
}

func TestUpdateData(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Create test data
	medicaments := []entities.Medicament{
		{Cis: 1, Denomination: "Test1"},
		{Cis: 2, Denomination: "Test2"},
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Gen1"},
		{GroupID: 2, Libelle: "Gen2"},
	}

	medicamentsMap := map[int]entities.Medicament{
		1: {Cis: 1, Denomination: "Test1"},
		2: {Cis: 2, Denomination: "Test2"},
	}

	generiquesMap := map[int]entities.Generique{
		1: {Cis: 1, Libelle: "Gen1"},
		2: {Cis: 2, Libelle: "Gen2"},
	}

	// Update data
	dc.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap)

	// Verify data was updated
	retrievedMedicaments := dc.GetMedicaments()
	if len(retrievedMedicaments) != 2 {
		t.Errorf("Expected 2 medicaments, got %d", len(retrievedMedicaments))
	}

	retrievedGeneriques := dc.GetGeneriques()
	if len(retrievedGeneriques) != 2 {
		t.Errorf("Expected 2 generiques, got %d", len(retrievedGeneriques))
	}

	retrievedMedicamentsMap := dc.GetMedicamentsMap()
	if len(retrievedMedicamentsMap) != 2 {
		t.Errorf("Expected 2 medicaments in map, got %d", len(retrievedMedicamentsMap))
	}

	retrievedGeneriquesMap := dc.GetGeneriquesMap()
	if len(retrievedGeneriquesMap) != 2 {
		t.Errorf("Expected 2 generiques in map, got %d", len(retrievedGeneriquesMap))
	}

	// Check last updated was set
	if dc.GetLastUpdated().IsZero() {
		t.Error("LastUpdated should be set after UpdateData")
	}
}

func TestBeginUpdateEndUpdate(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Test initial state
	if dc.IsUpdating() {
		t.Error("Should not be updating initially")
	}

	// Test BeginUpdate
	if !dc.BeginUpdate() {
		t.Error("BeginUpdate should return true first time")
	}

	if !dc.IsUpdating() {
		t.Error("Should be updating after BeginUpdate")
	}

	// Test that second BeginUpdate fails
	if dc.BeginUpdate() {
		t.Error("BeginUpdate should return false when already updating")
	}

	// Test EndUpdate
	dc.EndUpdate()

	if dc.IsUpdating() {
		t.Error("Should not be updating after EndUpdate")
	}

	// Test that BeginUpdate works again after EndUpdate
	if !dc.BeginUpdate() {
		t.Error("BeginUpdate should return true after EndUpdate")
	}

	dc.EndUpdate()
}

func TestConcurrentAccess(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Create test data
	medicaments := []entities.Medicament{
		{Cis: 1, Denomination: "Test1"},
		{Cis: 2, Denomination: "Test2"},
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Gen1"},
		{GroupID: 2, Libelle: "Gen2"},
	}

	medicamentsMap := map[int]entities.Medicament{
		1: {Cis: 1, Denomination: "Test1"},
		2: {Cis: 2, Denomination: "Test2"},
	}

	generiquesMap := map[int]entities.Generique{
		1: {Cis: 1, Libelle: "Gen1"},
	}

	// Set initial data
	dc.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap)

	var wg sync.WaitGroup
	numReaders := 10
	numWriters := 3

	// Start concurrent readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				// Test all getter methods
				meds := dc.GetMedicaments()
				gens := dc.GetGeneriques()
				medsMap := dc.GetMedicamentsMap()
				gensMap := dc.GetGeneriquesMap()
				lastUpdated := dc.GetLastUpdated()
				isUpdating := dc.IsUpdating()

				// Basic sanity checks
				if len(meds) == 0 && !isUpdating {
					t.Errorf("Reader %d: Expected non-empty medicaments", id)
				}
				if len(gens) == 0 && !isUpdating {
					t.Errorf("Reader %d: Expected non-empty generiques", id)
				}
				if len(medsMap) == 0 && !isUpdating {
					t.Errorf("Reader %d: Expected non-empty medicaments map", id)
				}
				if len(gensMap) == 0 && !isUpdating {
					t.Errorf("Reader %d: Expected non-empty generiques map", id)
				}
				if lastUpdated.IsZero() && !isUpdating {
					t.Errorf("Reader %d: Expected non-zero lastUpdated", id)
				}

				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// Start concurrent writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if dc.BeginUpdate() {
					// Simulate some work
					time.Sleep(time.Microsecond * 100)

					// Update with new data
					newMedicaments := []entities.Medicament{
						{Cis: id*10 + 1, Denomination: "Test1"},
						{Cis: id*10 + 2, Denomination: "Test2"},
					}

					newGeneriques := []entities.GeneriqueList{
						{GroupID: id*10 + 1, Libelle: "Gen1"},
					}

					newMedicamentsMap := map[int]entities.Medicament{
						id*10 + 1: {Cis: id*10 + 1, Denomination: "Test1"},
						id*10 + 2: {Cis: id*10 + 2, Denomination: "Test2"},
					}

					newGeneriquesMap := map[int]entities.Generique{
						id*10 + 1: {Cis: id*10 + 1, Libelle: "Gen1"},
					}

					dc.UpdateData(newMedicaments, newGeneriques, newMedicamentsMap, newGeneriquesMap)
					dc.EndUpdate()
				}

				time.Sleep(time.Microsecond * 200)
			}
		}(i)
	}

	wg.Wait()

	// Final verification
	finalMedicaments := dc.GetMedicaments()
	if len(finalMedicaments) == 0 {
		t.Error("Final medicaments should not be empty")
	}
}

func TestAtomicSwapZeroDowntime(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Set initial data
	initialMedicaments := []entities.Medicament{
		{Cis: 1, Denomination: "Initial"},
	}
	dc.UpdateData(initialMedicaments, []entities.GeneriqueList{},
		map[int]entities.Medicament{1: {Cis: 1, Denomination: "Initial"}},
		map[int]entities.Generique{})

	// Start a reader that continuously reads data
	stop := make(chan bool)
	readCount := 0
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				meds := dc.GetMedicaments()
				if len(meds) > 0 {
					readCount++
				}
				time.Sleep(time.Microsecond)
			}
		}
	}()

	// Let the reader run for a bit
	time.Sleep(time.Microsecond * 100)

	// Update data multiple times rapidly
	for i := 0; i < 100; i++ {
		newMedicaments := []entities.Medicament{
			{Cis: i + 2, Denomination: "Update"},
		}
		dc.UpdateData(newMedicaments, []entities.GeneriqueList{},
			map[int]entities.Medicament{i + 2: {Cis: i + 2, Denomination: "Update"}},
			map[int]entities.Generique{})
	}

	// Stop the reader
	stop <- true
	wg.Wait()

	if readCount == 0 {
		t.Error("Reader should have read some data during updates")
	}

	// Verify final state
	finalMedicaments := dc.GetMedicaments()
	if len(finalMedicaments) != 1 {
		t.Errorf("Expected 1 medicament, got %d", len(finalMedicaments))
	}
}

func TestTypeSafety(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Test that getters handle invalid types gracefully
	// This is a bit tricky since we can't directly store invalid types
	// through the public API, but we can test the fallback behavior

	// Test empty container behavior
	medicaments := dc.GetMedicaments()
	if medicaments == nil {
		t.Error("GetMedicaments should never return nil")
	}

	generiques := dc.GetGeneriques()
	if generiques == nil {
		t.Error("GetGeneriques should never return nil")
	}

	medicamentsMap := dc.GetMedicamentsMap()
	if medicamentsMap == nil {
		t.Error("GetMedicamentsMap should never return nil")
	}

	generiquesMap := dc.GetGeneriquesMap()
	if generiquesMap == nil {
		t.Error("GetGeneriquesMap should never return nil")
	}

	lastUpdated := dc.GetLastUpdated()
	if lastUpdated.IsZero() {
		// This is expected for empty container
	}
}

func BenchmarkGetMedicaments(b *testing.B) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Set up test data
	medicaments := make([]entities.Medicament, 1000)
	for i := 0; i < 1000; i++ {
		medicaments[i] = entities.Medicament{Cis: i, Denomination: "Test"}
	}
	dc.UpdateData(medicaments, []entities.GeneriqueList{},
		map[int]entities.Medicament{}, map[int]entities.Generique{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dc.GetMedicaments()
	}
}

func BenchmarkGetMedicamentsMap(b *testing.B) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Set up test data
	medicamentsMap := make(map[int]entities.Medicament)
	for i := 0; i < 1000; i++ {
		medicamentsMap[i] = entities.Medicament{Cis: i, Denomination: "Test"}
	}
	dc.UpdateData([]entities.Medicament{}, []entities.GeneriqueList{},
		medicamentsMap, map[int]entities.Generique{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dc.GetMedicamentsMap()
	}
}

func BenchmarkUpdateData(b *testing.B) {
	logging.InitLogger("")

	dc := NewDataContainer()

	medicaments := make([]entities.Medicament, 1000)
	for i := 0; i < 1000; i++ {
		medicaments[i] = entities.Medicament{Cis: i, Denomination: "Test"}
	}

	generiques := make([]entities.GeneriqueList, 100)
	for i := 0; i < 100; i++ {
		generiques[i] = entities.GeneriqueList{GroupID: i, Libelle: "Test"}
	}

	medicamentsMap := make(map[int]entities.Medicament)
	for i := 0; i < 1000; i++ {
		medicamentsMap[i] = entities.Medicament{Cis: i, Denomination: "Test"}
	}

	generiquesMap := make(map[int]entities.Generique)
	for i := 0; i < 100; i++ {
		generiquesMap[i] = entities.Generique{Cis: i, Libelle: "Test"}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dc.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap)
	}
}
