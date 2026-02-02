package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
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
	medicaments, presentationsCIP7Map, presentationsCIP13Map, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("Failed to parse medicaments: %v", err)
	}

	// Verify presentation maps are populated
	if len(presentationsCIP7Map) == 0 {
		t.Error("CIP7 presentation map should not be empty")
	}
	if len(presentationsCIP13Map) == 0 {
		t.Error("CIP13 presentation map should not be empty")
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
	if len(generiques) < 20 {
		t.Errorf("Expected at least 20 generique groups, got %d", len(generiques))
	}

	if len(generiquesMap) < 20 {
		t.Errorf("Expected at least 20 generiques in map, got %d", len(generiquesMap))
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
	medicaments1, _, _, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("First parse failed: %v", err)
	}

	// Wait a bit to ensure different timestamps
	time.Sleep(2 * time.Second)

	// Second parse (simulating concurrent update)
	medicaments2, _, _, err := medicamentsparser.ParseAllMedicaments()
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
			_ = os.WriteFile(file, data, 0644)
		}
	}()

	// Test parsing with missing files - the parser auto-downloads missing files
	for _, file := range testFiles {
		// Remove file
		_ = os.Remove(file)

		// Try to parse - should succeed by auto-downloading
		_, _, _, err := medicamentsparser.ParseAllMedicaments()
		if err != nil {
			t.Errorf("Expected success when %s is missing (should auto-download), but got error: %v", file, err)
		}

		// Restore file for next iteration
		if data, exists := backups[file]; exists {
			_ = os.WriteFile(file, data, 0644)
		}
	}

	fmt.Println("Error handling test completed successfully")
}

// TestIntegrationMemoryUsage tests memory usage during parsing
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

func verifyDataIntegrity(t *testing.T, medicaments []entities.Medicament, generiques []entities.GeneriqueList, medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.GeneriqueList) {
	// Use existing validator to generate data quality report (eliminates redundant validation logic)
	validator := validation.NewDataValidator()
	report := validator.ReportDataQuality(medicaments, generiques)

	// Log data quality issues found (real-world data issues, not test failures)
	if report.MedicamentsWithoutCompositions > 0 {
		t.Logf("Data quality: %d medicaments without compositions (sample CIS: %v)",
			report.MedicamentsWithoutCompositions, report.MedicamentsWithoutCompositionsCIS[:min(10, len(report.MedicamentsWithoutCompositionsCIS))])
	}
	if report.MedicamentsWithoutPresentations > 0 {
		t.Logf("Data quality: %d medicaments without presentations (sample CIS: %v)",
			report.MedicamentsWithoutPresentations, report.MedicamentsWithoutPresentationsCIS[:min(10, len(report.MedicamentsWithoutPresentationsCIS))])
	}
	if len(report.DuplicateCIS) > 0 {
		t.Logf("Data quality: %d duplicate CIS codes found", len(report.DuplicateCIS))
	}
	if len(report.DuplicateGroupIDs) > 0 {
		t.Logf("Data quality: %d duplicate generique group IDs found", len(report.DuplicateGroupIDs))
	}
	if report.GeneriqueOnlyCIS > 0 {
		t.Logf("Data quality: %d generique-only CIS (sample: %v)",
			report.GeneriqueOnlyCIS, report.GeneriqueOnlyCISList[:min(10, len(report.GeneriqueOnlyCISList))])
	}

	// Integration-test-specific structural checks (not covered by validator)

	// Test 1: Verify medicaments map size matches slice size
	if len(medicamentsMap) != len(medicaments) {
		t.Errorf("Medicaments map size mismatch: %d vs %d", len(medicamentsMap), len(medicaments))
	}

	// Test 2: Verify all medicaments in map exist in slice
	for cis, med := range medicamentsMap {
		if med.Cis != cis {
			t.Errorf("Map key mismatch: key %d, medicament CIS %d", cis, med.Cis)
		}
	}

	// Test 3: Log empty generique libelle as INFO
	for _, gen := range generiques {
		if gen.Libelle == "" {
			// Some generique groups may have empty libelle in source data
			t.Logf("Found generique group with empty libelle: ID %d", gen.GroupID)
		}
	}

	// Test 4: Verify generiques map size matches slice size
	if len(generiquesMap) != len(generiques) {
		t.Errorf("Generiques map size mismatch: %d vs %d", len(generiquesMap), len(generiques))
	}

	// Test 5: Verify all generiques in map exist in slice
	for groupID, gen := range generiquesMap {
		if gen.GroupID != groupID {
			t.Errorf("Generiques map key mismatch: key %d, generique GroupID %d", groupID, gen.GroupID)
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
	// Note: In real tests, we'd get presentation maps from ParseAllMedicaments
	// For now, using empty maps for this test
	dataContainer.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
		map[int]entities.Presentation{}, map[int]entities.Presentation{}, &interfaces.DataQualityReport{
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

	// Create HTTP handler
	validator := validation.NewDataValidator()
	httpHandler := handlers.NewHTTPHandler(dataContainer, validator)

	// Add routes using v1 handlers
	router.Get("/v1/medicaments/export", httpHandler.ExportMedicaments)
	router.Get("/v1/medicaments", httpHandler.ServeMedicamentsV1)
	router.Get("/v1/medicaments/{cis}", httpHandler.FindMedicamentByCIS)
	router.Get("/v1/generiques", httpHandler.ServeGeneriquesV1)
	router.Get("/v1/presentations/{cip}", httpHandler.ServePresentationsV1)
	router.Get("/health", httpHandler.HealthCheck)

	// Test database endpoint (export all)
	req := httptest.NewRequest("GET", "/v1/medicaments/export", nil)
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
	req = httptest.NewRequest("GET", "/v1/medicaments?page=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Paged database endpoint returned status %d, expected %d", w.Code, http.StatusOK)
	}

	// Verify pagination response
	var pagedResponse map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &pagedResponse); err != nil {
		t.Errorf("Failed to unmarshal paged response: %v", err)
	}

	// Test medicament by CIS endpoint (use first medicament)
	if len(medicaments) > 0 {
		firstCIS := medicaments[0].Cis
		req = httptest.NewRequest("GET", fmt.Sprintf("/v1/medicaments/%d", firstCIS), nil)
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
	var healthResponse map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &healthResponse); err != nil {
		t.Errorf("Failed to unmarshal health response: %v", err)
	}

	// Check for top-level fields (simplified health endpoint)
	topLevelFields := []string{"status", "data"}
	for _, field := range topLevelFields {
		if _, exists := healthResponse[field]; !exists {
			t.Errorf("Health response missing %s field", field)
		}
	}

	// Check data section fields
	if dataSection, ok := healthResponse["data"].(map[string]any); ok {
		dataFields := []string{"last_update", "medicaments", "generiques", "is_updating"}
		for _, field := range dataFields {
			if _, exists := dataSection[field]; !exists {
				t.Errorf("Health response data section missing %s field", field)
			}
		}
	} else {
		t.Error("Health response data section is not a map")
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
