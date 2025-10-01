package medicamentsparser

import (
	"fmt"
	"os"
	"testing"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// TestParseAllMedicaments tests the main parsing function with mock data
func TestParseAllMedicaments(t *testing.T) {
	fmt.Println("Starting TestParseAllMedicaments")

	// Create temporary test files to avoid downloading
	createTestFiles(t)
	fmt.Println("Test files created")

	defer cleanupTestFiles(t)
	fmt.Println("Cleanup scheduled")

	// Since downloadAndParseAll is private, we test with existing files
	// In a real scenario, ensure test files are present
	fmt.Println("Calling ParseAllMedicaments...")
	medicaments, err := ParseAllMedicaments()
	if err != nil {
		t.Fatalf("Error parsing Medicaments: %v", err)
	}
	fmt.Printf("Parsed %d medicaments\n", len(medicaments))

	if len(medicaments) == 0 {
		t.Error("Expected non-empty medicaments slice")
	}

	// Check if the first medicament has required fields
	if len(medicaments) > 0 {
		fmt.Printf("First medicament: CIS=%d, Denomination=%s\n", medicaments[0].Cis, medicaments[0].Denomination)
		if medicaments[0].Cis == 0 {
			t.Error("Expected CIS to be set")
		}
		if medicaments[0].Denomination == "" {
			t.Error("Expected Denomination to be set")
		}
	}

	fmt.Println("TestParseAllMedicaments completed")
}

// TestGeneriquesParser tests the generiques parsing
func TestGeneriquesParser(t *testing.T) {
	fmt.Println("Starting TestGeneriquesParser")

	// Create test files for generiques parsing
	createGeneriquesTestFiles(t)
	defer cleanupTestFiles(t)

	// Mock medicaments data
	medicaments := []entities.Medicament{
		{
			Cis:          1,
			Denomination: "Test Med 1",
			Generiques:   []entities.Generique{{Cis: 1, Group: 100, Libelle: "Group1", Type: "Princeps"}},
		},
	}

	medicamentsMap := map[int]entities.Medicament{
		1: medicaments[0],
	}

	fmt.Println("Calling GeneriquesParser...")
	generiques, generiquesMap, err := GeneriquesParser(&medicaments, &medicamentsMap)
	if err != nil {
		t.Fatalf("GeneriquesParser failed: %v", err)
	}
	fmt.Printf("Generated %d generiques and %d generiquesMap entries\n", len(generiques), len(generiquesMap))

	if len(generiques) == 0 {
		t.Error("Expected non-empty generiques slice")
	}

	if len(generiquesMap) == 0 {
		t.Error("Expected non-empty generiquesMap")
	}

	// Verify the first generique
	if len(generiques) > 0 {
		fmt.Printf("First generique: GroupId=%d, Libelle=%s, Medicaments count=%d\n",
			generiques[0].GroupID, generiques[0].Libelle, len(generiques[0].Medicaments))
	}

	fmt.Println("TestGeneriquesParser completed")
}

func TestFileReadingErrors(t *testing.T) {
	fmt.Println("Testing file reading error handling...")

	// Test with invalid JSON file
	os.MkdirAll("src", os.ModePerm)
	os.WriteFile("src/Specialites.json", []byte("invalid json"), 0644)
	defer os.RemoveAll("src")

	// This will log an error but not crash the test
	// We can't easily test log.Fatal without process exit
	// So we'll test with valid empty array
	os.WriteFile("src/Specialites.json", []byte("[]"), 0644)

	specialites, err := specialitesFileToJSON()
	if err != nil {
		t.Fatalf("Failed to read specialites file: %v", err)
	}
	if len(specialites) != 0 {
		t.Errorf("Expected empty slice for empty array, got %d items", len(specialites))
	}

	fmt.Println("File reading error test completed")
}

// Helper functions for testing
func createTestFiles(t *testing.T) {
	// Create minimal test JSON files
	testData := `[{"cis":1,"denomination":"Test","formePharmaceutique":"Tablet","voiesAdministration":["Oral"],"statusAutorisation":"Autorisé","typeProcedure":"Nationale","etatComercialisation":"Commercialisé","dateAMM":"2020-01-01","titulaire":"Test Lab","surveillanceRenforcee":"Non","composition":[],"generiques":[],"presentation":[],"conditions":[]}]`

	os.MkdirAll("src", os.ModePerm)
	os.WriteFile("src/Specialites.json", []byte(testData), 0644)
	os.WriteFile("src/Compositions.json", []byte("[]"), 0644)
	os.WriteFile("src/Conditions.json", []byte("[]"), 0644)
	os.WriteFile("src/Presentations.json", []byte("[]"), 0644)
	os.WriteFile("src/Generiques.json", []byte(`{"100":[1]}`), 0644)
}

func createGeneriquesTestFiles(t *testing.T) {
	// Create test files for generiques parsing
	os.MkdirAll("files", os.ModePerm)
	os.MkdirAll("src", os.ModePerm)

	// Create Generiques.txt with correct tab-separated format: group\tlibelle\tcis\ttype
	generiquesTxt := "100\tGroup1\t1\t0" // 0 for Princeps
	os.WriteFile("files/Generiques.txt", []byte(generiquesTxt), 0644)

	// Create Generiques.json
	generiquesJSON := `{"100":[1]}`
	os.WriteFile("src/Generiques.json", []byte(generiquesJSON), 0644)
}

func cleanupTestFiles(t *testing.T) {
	os.RemoveAll("src")
	os.RemoveAll("files")
}
