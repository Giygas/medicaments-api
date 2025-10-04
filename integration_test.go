package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// TestIntegrationFullDataParsingPipeline tests the complete data parsing pipeline
// from download to in-memory data structures used by the API
func TestIntegrationFullDataParsingPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fmt.Println("Starting full data parsing pipeline integration test...")

	// Setup test environment
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	// Record start time
	startTime := time.Now()

	// Execute the full parsing pipeline
	medicaments, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("Failed to parse medicaments: %v", err)
	}

	// Verify parsing completed within reasonable time (should be under 5 minutes)
	elapsed := time.Since(startTime)
	if elapsed > 5*time.Minute {
		t.Errorf("Parsing took too long: %v (expected < 5 minutes)", elapsed)
	}

	// Test 1: Verify we have a significant amount of data
	if len(medicaments) < 1000 {
		t.Errorf("Expected at least 1000 medicaments, got %d", len(medicaments))
	}

	// Test 2: Create medicaments map as done in main.go
	medicamentsMap := make(map[int]entities.Medicament)
	for i := range medicaments {
		medicamentsMap[medicaments[i].Cis] = medicaments[i]
	}

	// Test 3: Execute generiques parsing
	generiques, generiquesMap, err := medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
	if err != nil {
		t.Fatalf("Failed to parse generiques: %v", err)
	}

	// Test 4: Verify generiques data
	if len(generiques) < 100 {
		t.Errorf("Expected at least 100 generique groups, got %d", len(generiques))
	}

	if len(generiquesMap) < 100 {
		t.Errorf("Expected at least 100 generiques in map, got %d", len(generiquesMap))
	}

	// Test 5: Verify data integrity
	verifyDataIntegrity(t, medicaments, generiques, medicamentsMap, generiquesMap)

	// Test 6: Test API endpoints with real data
	testAPIEndpointsWithRealData(t, medicaments, generiques)

	fmt.Printf("Integration test completed successfully in %v\n", elapsed)
	fmt.Printf("Parsed %d medicaments and %d generique groups\n", len(medicaments), len(generiques))
}

// TestIntegrationConcurrentUpdates tests concurrent data updates
func TestIntegrationConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fmt.Println("Starting concurrent updates integration test...")

	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	// First parse
	medicaments1, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("First parse failed: %v", err)
	}

	// Wait a bit to ensure different timestamps
	time.Sleep(2 * time.Second)

	// Second parse (simulating concurrent update)
	medicaments2, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("Second parse failed: %v", err)
	}

	// Verify both parses completed successfully
	if len(medicaments1) == 0 || len(medicaments2) == 0 {
		t.Error("One of the parses returned empty data")
	}

	// Verify data consistency (should be roughly the same size)
	sizeDiff := abs(len(medicaments1) - len(medicaments2))
	if sizeDiff > len(medicaments1)/10 { // Allow 10% difference
		t.Errorf("Data size difference too large: %d vs %d", len(medicaments1), len(medicaments2))
	}

	fmt.Println("Concurrent updates test completed successfully")
}

// TestIntegrationErrorHandling tests error handling in the pipeline
func TestIntegrationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fmt.Println("Starting error handling integration test...")

	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	// Test with corrupted files (if they exist)
	testFiles := []string{
		"files/Specialites.txt",
		"files/Presentations.txt",
		"files/Compositions.txt",
		"files/Generiques.txt",
		"files/Conditions.txt",
	}

	// Backup original files if they exist
	backups := make(map[string][]byte)
	for _, file := range testFiles {
		if data, err := os.ReadFile(file); err == nil {
			backups[file] = data
		}
	}

	// Restore files after test
	defer func() {
		for file, data := range backups {
			os.WriteFile(file, data, 0644)
		}
	}()

	// Test parsing with missing files - the parser auto-downloads missing files
	for _, file := range testFiles {
		// Remove file
		os.Remove(file)

		// Try to parse - should succeed by auto-downloading
		_, err := medicamentsparser.ParseAllMedicaments()
		if err != nil {
			t.Errorf("Expected success when %s is missing (should auto-download), but got error: %v", file, err)
		}

		// Restore file for next iteration
		if data, exists := backups[file]; exists {
			os.WriteFile(file, data, 0644)
		}
	}

	fmt.Println("Error handling test completed successfully")
}

// TestIntegrationMemoryUsage tests memory usage during parsing
func TestIntegrationMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fmt.Println("Starting memory usage integration test...")

	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	// Get initial memory stats
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)

	// Parse data
	medicaments, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("Failed to parse medicaments: %v", err)
	}

	// Get final memory stats
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)

	// Calculate memory usage (handle potential overflow)
	var memoryUsedMB uint64
	if finalMem.Alloc > initialMem.Alloc {
		memoryUsedMB = (finalMem.Alloc - initialMem.Alloc) / 1024 / 1024
	}

	var medicamentsPerMB float64
	if memoryUsedMB > 0 {
		medicamentsPerMB = float64(len(medicaments)) / float64(memoryUsedMB)
	}

	fmt.Printf("Memory used: %d MB\n", memoryUsedMB)
	fmt.Printf("Medicaments per MB: %.2f\n", medicamentsPerMB)

	// Verify memory usage is reasonable (should be less than 1GB for the full dataset)
	if memoryUsedMB > 1024 {
		t.Errorf("Memory usage too high: %d MB (expected < 1024 MB)", memoryUsedMB)
	}

	// Verify efficiency (should handle at least 10 medicaments per MB)
	if memoryUsedMB > 0 && medicamentsPerMB < 10 {
		t.Errorf("Memory efficiency too low: %.2f medicaments/MB (expected > 10)", medicamentsPerMB)
	}

	fmt.Println("Memory usage test completed successfully")
}

// Helper functions

func setupTestEnvironment(t *testing.T) {
	// Create necessary directories
	dirs := []string{"files", "src", "logs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Initialize logging
	logging.InitLogger("logs")
}

func cleanupTestEnvironment(t *testing.T) {
	// Clean up test files (optional - keep for debugging)
	// Uncomment to clean up files after tests
	// os.RemoveAll("files")
	// os.RemoveAll("src")
}

func verifyDataIntegrity(t *testing.T, medicaments []entities.Medicament, generiques []entities.GeneriqueList, medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.Generique) {
	// Test 1: Verify all medicaments have valid CIS
	for _, med := range medicaments {
		if med.Cis <= 0 {
			t.Errorf("Found medicament with invalid CIS: %d", med.Cis)
		}
		if med.Denomination == "" {
			t.Errorf("Found medicament with empty denomination: CIS %d", med.Cis)
		}
	}

	// Test 2: Verify medicaments map consistency
	if len(medicamentsMap) != len(medicaments) {
		t.Errorf("Medicaments map size mismatch: %d vs %d", len(medicamentsMap), len(medicaments))
	}

	// Test 3: Verify all medicaments in map exist in slice
	for cis, med := range medicamentsMap {
		if med.Cis != cis {
			t.Errorf("Map key mismatch: key %d, medicament CIS %d", cis, med.Cis)
		}
	}

	// Test 4: Verify generique groups have valid data
	for _, gen := range generiques {
		if gen.GroupID <= 0 {
			t.Errorf("Found generique group with invalid ID: %d", gen.GroupID)
		}
		if gen.Libelle == "" {
			t.Errorf("Found generique group with empty libelle: ID %d", gen.GroupID)
		}
		if len(gen.Medicaments) == 0 {
			// Some generique groups may have no medicaments - this is expected behavior
			// Log as info rather than warning
			t.Logf("Found generique group with no medicaments: ID %d", gen.GroupID)
		}
	}

	// Test 5: Verify cross-references are valid
	for _, gen := range generiques {
		for _, med := range gen.Medicaments {
			if _, exists := medicamentsMap[med.Cis]; !exists {
				t.Errorf("Found medicament in generique group that doesn't exist in medicaments map: CIS %d", med.Cis)
			}
		}
	}
}

func testAPIEndpointsWithRealData(t *testing.T, medicaments []entities.Medicament, generiques []entities.GeneriqueList) {
	// Create a test router with real data
	router := chi.NewRouter()

	// Create a new data container for testing
	dataContainer := data.NewDataContainer()

	// Create medicaments map
	medicamentsMap := make(map[int]entities.Medicament)
	for i := range medicaments {
		medicamentsMap[medicaments[i].Cis] = medicaments[i]
	}

	// Create generiques map
	_, generiquesMap, err := medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
	if err != nil {
		t.Fatalf("Failed to create generiques map for API testing: %v", err)
	}

	// Load data into the container (simulating real API behavior)
	dataContainer.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap)

	// Add routes using new handlers
	router.Get("/database", handlers.ServeAllMedicaments(dataContainer))
	router.Get("/database/{pageNumber}", handlers.ServePagedMedicaments(dataContainer))
	router.Get("/medicament/id/{cis}", handlers.FindMedicamentByID(dataContainer))
	router.Get("/health", handlers.HealthCheck(dataContainer))

	// Test database endpoint
	req := httptest.NewRequest("GET", "/database", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Database endpoint returned status %d, expected %d", w.Code, http.StatusOK)
	}

	// Verify response contains data
	var response []entities.Medicament
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal database response: %v", err)
	}

	if len(response) != len(medicaments) {
		t.Errorf("Database endpoint returned %d medicaments, expected %d", len(response), len(medicaments))
	}

	// Test paged database endpoint
	req = httptest.NewRequest("GET", "/database/1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Paged database endpoint returned status %d, expected %d", w.Code, http.StatusOK)
	}

	// Test medicament by ID endpoint (use first medicament)
	if len(medicaments) > 0 {
		firstCIS := medicaments[0].Cis
		req = httptest.NewRequest("GET", fmt.Sprintf("/medicament/id/%d", firstCIS), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Medicament by ID endpoint returned status %d, expected %d", w.Code, http.StatusOK)
		}

		var responseMed entities.Medicament
		if err := json.Unmarshal(w.Body.Bytes(), &responseMed); err != nil {
			t.Errorf("Failed to unmarshal medicament response: %v", err)
		}

		if responseMed.Cis != firstCIS {
			t.Errorf("Medicament endpoint returned CIS %d, expected %d", responseMed.Cis, firstCIS)
		}
	}

	// Test health endpoint
	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health endpoint returned status %d, expected %d", w.Code, http.StatusOK)
	}

	// Verify health response contains expected fields
	var healthResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &healthResponse); err != nil {
		t.Errorf("Failed to unmarshal health response: %v", err)
	}

	// Check for top-level fields
	topLevelFields := []string{"status", "last_update", "data_age_hours", "uptime_seconds", "data", "system"}
	for _, field := range topLevelFields {
		if _, exists := healthResponse[field]; !exists {
			t.Errorf("Health response missing %s field", field)
		}
	}

	// Check data section fields
	if dataSection, ok := healthResponse["data"].(map[string]interface{}); ok {
		dataFields := []string{"api_version", "medicaments", "generiques", "is_updating", "next_update"}
		for _, field := range dataFields {
			if _, exists := dataSection[field]; !exists {
				t.Errorf("Health response data section missing %s field", field)
			}
		}
	} else {
		t.Error("Health response data section is not a map")
	}

	// Check system section fields
	if systemSection, ok := healthResponse["system"].(map[string]interface{}); ok {
		systemFields := []string{"goroutines", "memory"}
		for _, field := range systemFields {
			if _, exists := systemSection[field]; !exists {
				t.Errorf("Health response system section missing %s field", field)
			}
		}
	} else {
		t.Error("Health response system section is not a map")
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
