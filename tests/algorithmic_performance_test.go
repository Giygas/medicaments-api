package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/data"
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

		medicaments, err := medicamentsparser.ParseAllMedicaments()
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
		algorithmicTestData.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap)

		fmt.Printf("Algorithmic test data loaded: %d medicaments, %d generiques\n", len(medicaments), len(generiques))
	})
	return algorithmicTestData
}

// Test raw map lookup performance (O(1) operations)
func BenchmarkMapLookupByID(b *testing.B) {
	container := setupAlgorithmicTestData()
	medicamentsMap := container.GetMedicamentsMap()

	// Use a variety of CIS values for realistic testing
	testCIS := []int{100, 500, 1000, 5000, 10000}
	cisIndex := 0

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cis := testCIS[cisIndex%len(testCIS)]
		_ = medicamentsMap[cis] // Direct map lookup
		cisIndex++
	}
}

// Test regex compilation and matching performance
func BenchmarkRegexMatching(b *testing.B) {
	container := setupAlgorithmicTestData()
	medicaments := container.GetMedicaments()

	// Test patterns of varying complexity
	patterns := []string{
		"paracetamol",
		"ibuprofène",
		"amoxicilline",
		"aspirine",
		"doliprane",
	}

	patternIndex := 0

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pattern := patterns[patternIndex%len(patterns)]
		regex := regexp.MustCompile("(?i)" + regexp.QuoteMeta(pattern))

		// Search through medicaments
		found := false
		for _, med := range medicaments {
			if regex.MatchString(med.Denomination) {
				found = true
				break // Found one match, move to next pattern
			}
		}
		_ = found // Prevent optimization
		patternIndex++
	}
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

// Test slice iteration performance
func BenchmarkSliceIteration(b *testing.B) {
	container := setupAlgorithmicTestData()
	medicaments := container.GetMedicaments()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		count := 0
		for _, med := range medicaments {
			if med.Cis > 0 {
				count++
			}
		}
		_ = count // Prevent optimization
	}
}

// Test memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	container := setupAlgorithmicTestData()
	medicaments := container.GetMedicaments()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate creating response slices
		results := make([]entities.Medicament, 0, 100)

		for j := 0; j < 100 && j < len(medicaments); j++ {
			results = append(results, medicaments[j])
		}

		_ = results // Prevent optimization
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

// Test pagination performance
func BenchmarkPagination(b *testing.B) {
	container := setupAlgorithmicTestData()
	medicaments := container.GetMedicaments()
	pageSize := 10

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		page := (i % 100) + 1 // Test pages 1-100
		start := (page - 1) * pageSize
		end := start + pageSize

		if start >= len(medicaments) {
			start = 0
			end = pageSize
		}
		if end > len(medicaments) {
			end = len(medicaments)
		}

		_ = medicaments[start:end] // Simulate pagination slice
	}
}

// Test concurrent map access
func BenchmarkConcurrentMapAccess(b *testing.B) {
	container := setupAlgorithmicTestData()
	medicamentsMap := container.GetMedicamentsMap()
	testCIS := []int{100, 500, 1000, 5000, 10000}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		cisIndex := 0
		for pb.Next() {
			cis := testCIS[cisIndex%len(testCIS)]
			_ = medicamentsMap[cis] // Direct map lookup
			cisIndex++
		}
	})
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

	// Calculate memory usage
	allocDiff := m2.Alloc - m1.Alloc
	allocDiffMB := allocDiff / 1024 / 1024

	t.Logf("Memory footprint analysis:")
	t.Logf("  Medicaments slice: %d items", len(medicaments))
	t.Logf("  Generiques slice: %d items", len(generiques))
	t.Logf("  Medicaments map: %d entries", len(medicamentsMap))
	t.Logf("  Generiques map: %d entries", len(generiquesMap))
	t.Logf("  Additional memory: %d MB", allocDiffMB)

	// Calculate efficiency
	if allocDiffMB > 0 {
		medicamentsPerMB := float64(len(medicaments)) / float64(allocDiffMB)
		t.Logf("  Efficiency: %.2f medicaments per MB", medicamentsPerMB)

		if medicamentsPerMB < 100 {
			t.Errorf("Memory efficiency seems low: %.2f medicaments/MB (expected > 100)", medicamentsPerMB)
		}
	}
}

// Test search algorithm complexity
func TestSearchComplexity(t *testing.T) {
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

// Test atomic operations performance
func BenchmarkAtomicOperations(b *testing.B) {
	container := setupAlgorithmicTestData()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Test atomic data access
		_ = container.GetMedicaments()
		_ = container.GetGeneriques()
		_ = container.GetMedicamentsMap()
		_ = container.GetGeneriquesMap()
	}
}

// Test regex compilation overhead
func BenchmarkRegexCompilation(b *testing.B) {
	patterns := []string{
		"paracetamol",
		"ibuprofène.*acide",
		"amoxicilline.*acide.*clavulanique",
		"[a-z]+[0-9]+",
		"^(doliprane|efferalgan).*",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pattern := patterns[i%len(patterns)]
		regex := regexp.MustCompile("(?i)" + regexp.QuoteMeta(pattern))
		_ = regex
	}
}

// Test data validation performance
func BenchmarkDataValidation(b *testing.B) {
	container := setupAlgorithmicTestData()
	medicaments := container.GetMedicaments()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		med := medicaments[i%len(medicaments)]

		// Simulate validation checks
		valid := med.Cis > 0 &&
			med.Denomination != "" &&
			len(med.Denomination) >= 3 &&
			len(med.Denomination) <= 50

		_ = valid // Prevent optimization
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
