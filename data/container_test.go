package data

import (
	"fmt"
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

	generiquesMap := map[int]entities.GeneriqueList{
		1: {GroupID: 1, Libelle: "Gen1"},
		2: {GroupID: 2, Libelle: "Gen2"},
	}

	// Update data
	presentationsCIP7Map := map[int]entities.Presentation{
		1234567: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
	}
	presentationsCIP13Map := map[int]entities.Presentation{
		3400912345678: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
	}
	dc.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap, presentationsCIP7Map, presentationsCIP13Map)

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

	generiquesMap := map[int]entities.GeneriqueList{
		1: {GroupID: 1, Libelle: "Gen1"},
	}

	// Set initial data
	dc.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
		map[int]entities.Presentation{}, map[int]entities.Presentation{})

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

					newGeneriquesMap := map[int]entities.GeneriqueList{
						id*10 + 1: {GroupID: id*10 + 1, Libelle: "Gen1"},
					}

					dc.UpdateData(newMedicaments, newGeneriques, newMedicamentsMap, newGeneriquesMap,
						map[int]entities.Presentation{}, map[int]entities.Presentation{})
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
		map[int]entities.GeneriqueList{},
		map[int]entities.Presentation{}, map[int]entities.Presentation{})

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
			map[int]entities.GeneriqueList{},
			map[int]entities.Presentation{}, map[int]entities.Presentation{})
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

	// Last updated is zero for empty container (expected)
}

func TestSetServerStartTime(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Test initial state
	initialTime := dc.GetServerStartTime()
	if !initialTime.IsZero() {
		t.Error("Initial server start time should be zero")
	}

	// Set a specific time
	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	dc.SetServerStartTime(testTime)

	// Verify the time was set correctly
	retrievedTime := dc.GetServerStartTime()
	if retrievedTime != testTime {
		t.Errorf("Expected server start time %v, got %v", testTime, retrievedTime)
	}

	// Test updating the time
	newTestTime := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)
	dc.SetServerStartTime(newTestTime)

	retrievedTime = dc.GetServerStartTime()
	if retrievedTime != newTestTime {
		t.Errorf("Expected updated server start time %v, got %v", newTestTime, retrievedTime)
	}

	// Test that the time is not zero after setting
	if retrievedTime.IsZero() {
		t.Error("Server start time should not be zero after SetServerStartTime")
	}
}

func TestGetServerStartTime(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Test GetServerStartTime when not set
	unsetTime := dc.GetServerStartTime()
	if !unsetTime.IsZero() {
		t.Error("GetServerStartTime should return zero time when not set")
	}

	// Set a time and verify retrieval
	now := time.Now()
	dc.SetServerStartTime(now)

	retrievedTime := dc.GetServerStartTime()
	if retrievedTime.IsZero() {
		t.Error("GetServerStartTime should not return zero after SetServerStartTime")
	}

	// The retrieved time should be very close to the set time
	// (accounting for any potential clock adjustments)
	if retrievedTime.Sub(now) > time.Second {
		t.Errorf("Retrieved time differs significantly from set time: expected %v, got %v", now, retrievedTime)
	}

	// Test multiple containers have independent times
	dc2 := NewDataContainer()
	dc2Time := dc2.GetServerStartTime()
	if !dc2Time.IsZero() {
		t.Error("New container should have zero server start time")
	}

	// Verify dc1 still has its time
	dc1Time := dc.GetServerStartTime()
	if dc1Time.IsZero() {
		t.Error("Original container should still have server start time")
	}
}

func TestServerStartTimeConcurrentAccess(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	var wg sync.WaitGroup
	numWriters := 10
	numReaders := 20

	// Start concurrent writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				// Set a unique time for this goroutine
				testTime := time.Date(2024, 1, 1, id, j, 0, 0, time.UTC)
				dc.SetServerStartTime(testTime)
			}
		}(i)
	}

	// Start concurrent readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				startTime := dc.GetServerStartTime()
				// Basic sanity check - should be able to get time without panicking
				if !startTime.IsZero() {
					// Time should be reasonable (not in the far future or distant past)
					if startTime.Year() < 2020 || startTime.Year() > 2030 {
						t.Errorf("Reader %d: Got unexpected time: %v", id, startTime)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Final verification - container should still have a time
	finalTime := dc.GetServerStartTime()
	if finalTime.IsZero() {
		t.Error("Server start time should be set after concurrent access")
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
		map[int]entities.Medicament{}, map[int]entities.GeneriqueList{},
		map[int]entities.Presentation{}, map[int]entities.Presentation{})

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
		medicamentsMap, map[int]entities.GeneriqueList{},
		map[int]entities.Presentation{}, map[int]entities.Presentation{})

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

	generiquesMap := make(map[int]entities.GeneriqueList)
	for i := 0; i < 100; i++ {
		generiquesMap[i] = entities.GeneriqueList{GroupID: i, Libelle: "Test"}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dc.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
			map[int]entities.Presentation{}, map[int]entities.Presentation{})
	}
}

func TestGetPresentationsCIP7Map(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Initial state should have empty map
	cip7Map := dc.GetPresentationsCIP7Map()
	if len(cip7Map) != 0 {
		t.Errorf("Expected empty CIP7 map initially, got %d entries", len(cip7Map))
	}

	// Add test data
	testPresentations := map[int]entities.Presentation{
		1234567: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
		2345678: {Cis: 2, Cip7: 2345678, Cip13: 3400923456789},
	}

	dc.UpdateData([]entities.Medicament{}, []entities.GeneriqueList{},
		map[int]entities.Medicament{}, map[int]entities.GeneriqueList{},
		testPresentations, map[int]entities.Presentation{})

	// Verify data was stored
	retrievedMap := dc.GetPresentationsCIP7Map()
	if len(retrievedMap) != 2 {
		t.Errorf("Expected 2 CIP7 map entries, got %d", len(retrievedMap))
	}

	// Verify specific entries
	if _, exists := retrievedMap[1234567]; !exists {
		t.Error("CIP7 1234567 not found in map")
	}

	if _, exists := retrievedMap[2345678]; !exists {
		t.Error("CIP7 2345678 not found in map")
	}
}

func TestGetPresentationsCIP13Map(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Initial state should have empty map
	cip13Map := dc.GetPresentationsCIP13Map()
	if len(cip13Map) != 0 {
		t.Errorf("Expected empty CIP13 map initially, got %d entries", len(cip13Map))
	}

	// Add test data
	testPresentations := map[int]entities.Presentation{
		3400912345678: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
		3400923456789: {Cis: 2, Cip7: 2345678, Cip13: 3400923456789},
	}

	dc.UpdateData([]entities.Medicament{}, []entities.GeneriqueList{},
		map[int]entities.Medicament{}, map[int]entities.GeneriqueList{},
		map[int]entities.Presentation{}, testPresentations)

	// Verify data was stored
	retrievedMap := dc.GetPresentationsCIP13Map()
	if len(retrievedMap) != 2 {
		t.Errorf("Expected 2 CIP13 map entries, got %d", len(retrievedMap))
	}

	// Verify specific entries
	if _, exists := retrievedMap[3400912345678]; !exists {
		t.Error("CIP13 3400912345678 not found in map")
	}

	if _, exists := retrievedMap[3400923456789]; !exists {
		t.Error("CIP13 3400923456789 not found in map")
	}
}

func TestPresentationMapsConcurrentAccess(t *testing.T) {
	logging.InitLogger("")

	dc := NewDataContainer()

	// Set up test data
	cip7Map := map[int]entities.Presentation{
		1234567: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
	}
	cip13Map := map[int]entities.Presentation{
		3400912345678: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
	}

	dc.UpdateData([]entities.Medicament{}, []entities.GeneriqueList{},
		map[int]entities.Medicament{}, map[int]entities.GeneriqueList{},
		cip7Map, cip13Map)

	var wg sync.WaitGroup
	numReaders := 20
	numWriters := 5
	errors := make(chan error, numReaders+numWriters)

	// Start concurrent readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				cip7 := dc.GetPresentationsCIP7Map()
				cip13 := dc.GetPresentationsCIP13Map()

				// Basic sanity checks
				if len(cip7) == 0 {
					errors <- fmt.Errorf("Reader %d: CIP7 map empty", id)
					return
				}
				if len(cip13) == 0 {
					errors <- fmt.Errorf("Reader %d: CIP13 map empty", id)
					return
				}
			}
		}(i)
	}

	// Start concurrent writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				newCIP7 := map[int]entities.Presentation{
					1000 + id*10 + j: {Cis: id, Cip7: 1000 + id*10 + j, Cip13: int(3400900000000 + int64(id)*100000000)},
				}
				newCIP13 := map[int]entities.Presentation{
					int(3400900000000 + int64(id)*100000000): {Cis: id, Cip7: 1000 + id*10 + j, Cip13: int(3400900000000 + int64(id)*100000000)},
				}
				dc.UpdateData([]entities.Medicament{}, []entities.GeneriqueList{},
					map[int]entities.Medicament{}, map[int]entities.GeneriqueList{},
					newCIP7, newCIP13)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Concurrent access test failed with %d errors", errorCount)
	}
}
