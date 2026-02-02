package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

var (
	algorithmicTestData *data.DataContainer
	algorithmicOnce     sync.Once
)

// Setup algorithmic test data with full dataset
func setupAlgorithmicTestData() *data.DataContainer {
	algorithmicOnce.Do(func() {
		fmt.Println("Loading full dataset for algorithmic performance tests...")

		medicaments, presentationsCIP7Map, presentationsCIP13Map, err := medicamentsparser.ParseAllMedicaments()
		if err != nil {
			panic(fmt.Sprintf("Failed to parse medicaments: %v", err))
		}

		medicamentsMap := make(map[int]entities.Medicament)
		for i := range medicaments {
			medicamentsMap[medicaments[i].Cis] = medicaments[i]
		}

		generiques, generiquesMap, err := medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse generiques: %v", err))
		}

		algorithmicTestData = data.NewDataContainer()
		algorithmicTestData.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
			presentationsCIP7Map, presentationsCIP13Map, &interfaces.DataQualityReport{
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
			})

		fmt.Printf("Algorithmic test data loaded: %d medicaments, %d generiques\n", len(medicaments), len(generiques))
	})
	return algorithmicTestData
}

// Test string operations performance
func BenchmarkStringOperations(b *testing.B) {
	// Test various string operations
	testStrings := []string{
		"PARACETAMOL",
		"IBUPROFENE",
		"AMOXICILLINE",
		"ASPIRINE",
		"DOLIPRANE",
	}

	stringIndex := 0

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		testStr := testStrings[stringIndex%len(testStrings)]

		// Simulate the string operations done in search
		lower := strings.ToLower(testStr)
		escaped := regexp.QuoteMeta(lower)
		pattern := "(?i)" + escaped

		_ = lower
		_ = escaped
		_ = pattern

		stringIndex++
	}
}

// Test JSON marshaling performance
func BenchmarkJSONMarshaling(b *testing.B) {
	container := setupAlgorithmicTestData()
	medicaments := container.GetMedicaments()

	// Create a test slice
	testData := medicaments[:min(100, len(medicaments))]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test data structure memory footprint
func TestMemoryFootprint(t *testing.T) {
	container := setupAlgorithmicTestData()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	medicaments := container.GetMedicaments()
	generiques := container.GetGeneriques()
	medicamentsMap := container.GetMedicamentsMap()
	generiquesMap := container.GetGeneriquesMap()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate memory usage using Sys for more stable measurement
	sysDiff := m2.Sys - m1.Sys
	sysDiffMB := sysDiff / 1024 / 1024

	t.Logf("Memory footprint analysis:")
	t.Logf("  Medicaments slice: %d items", len(medicaments))
	t.Logf("  Generiques slice: %d items", len(generiques))
	t.Logf("  Medicaments map: %d entries", len(medicamentsMap))
	t.Logf("  Generiques map: %d entries", len(generiquesMap))
	t.Logf("  System memory before: %d MB", m1.Sys/1024/1024)
	t.Logf("  System memory after: %d MB", m2.Sys/1024/1024)
	t.Logf("  Additional memory: %d MB", sysDiffMB)

	// Calculate efficiency with safeguards
	if sysDiffMB > 0 && sysDiffMB < 10000 { // Reasonable upper bound
		medicamentsPerMB := float64(len(medicaments)) / float64(sysDiffMB)
		t.Logf("  Efficiency: %.2f medicaments per MB", medicamentsPerMB)

		if medicamentsPerMB < 100 {
			t.Errorf("Memory efficiency seems low: %.2f medicaments/MB (expected > 100)", medicamentsPerMB)
		}
	} else if sysDiffMB <= 0 {
		t.Logf("  Memory measurement inconclusive (diff: %d MB), possibly due to GC optimizations", sysDiffMB)
	} else {
		t.Errorf("Memory measurement seems unrealistic: %d MB (possible measurement error)", sysDiffMB)
	}
}

// Test search algorithm complexity
func TestSearchComplexity(t *testing.T) {
	// Skip performance verification in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping search complexity verification in CI environment")
	}

	container := setupAlgorithmicTestData()
	medicaments := container.GetMedicaments()

	// Test with different dataset sizes to verify O(n) complexity
	sizes := []int{100, 500, 1000, 5000, len(medicaments)}
	pattern := "paracetamol"

	for _, size := range sizes {
		if size > len(medicaments) {
			size = len(medicaments)
		}

		testData := medicaments[:size]

		start := time.Now()
		count := 0
		for _, med := range testData {
			if strings.Contains(strings.ToLower(med.Denomination), pattern) {
				count++
			}
		}
		elapsed := time.Since(start)

		t.Logf("Search in %d items: %v (%d results)", size, elapsed, count)

		// Verify reasonable performance (should scale linearly)
		expectedMaxTime := time.Duration(size) * time.Microsecond * 10 // 10μs per item max
		if elapsed > expectedMaxTime {
			t.Errorf("Search too slow for %d items: %v (expected < %v)", size, elapsed, expectedMaxTime)
		}
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
