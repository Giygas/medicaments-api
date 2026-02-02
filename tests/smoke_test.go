package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
)

// TestApplicationStartupSmoke tests basic application startup and functionality
// This is a fast sanity check that should run before expensive integration tests
// Expected duration: < 2 seconds
func TestApplicationStartupSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping smoke test in short mode")
	}

	t.Log("Starting application smoke test...")

	// 0. Initialize logging first
	logging.InitLogger("logs")

	// 1. Create minimal data container with sample data
	// This ensures data is available for endpoints
	sampleMedicament := entities.Medicament{
		Cis:                  12345678,
		Denomination:         "Smoke Test Medicament",
		FormePharmaceutique:  "Tablet",
		StatusAutorisation:   "Autorisée",
		EtatComercialisation: "Commercialisée",
		Conditions:           []string{"Smoke condition"},
		Presentation: []entities.Presentation{
			{
				Cip7:    7001001,
				Cip13:   3400912345678,
				Libelle: "Boîte de 10 comprimés",
			},
		},
		Composition: []entities.Composition{
			{
				CodeSubstance:         1,
				DenominationSubstance: "Test substance",
			},
		},
		Generiques: []entities.Generique{
			{
				Cis:   12345678,
				Group: 100,
			},
		},
	}

	sampleGenerique := entities.GeneriqueList{
		GroupID: 100,
		Libelle: "Smoke Test Group",
		Medicaments: []entities.GeneriqueMedicament{
			{Cis: 12345678, Denomination: "Smoke Test Medicament"},
		},
	}

	dataContainer := data.NewDataContainer()
	dataContainer.UpdateData(
		[]entities.Medicament{sampleMedicament},
		[]entities.GeneriqueList{sampleGenerique},
		map[int]entities.Medicament{sampleMedicament.Cis: sampleMedicament},
		map[int]entities.GeneriqueList{sampleGenerique.GroupID: sampleGenerique},
		map[int]entities.Presentation{},
		map[int]entities.Presentation{},
		&interfaces.DataQualityReport{},
	)

	// 2. Create validator and handler
	validator := validation.NewDataValidator()
	httpHandler := handlers.NewHTTPHandler(dataContainer, validator)

	// 3. Test health endpoint
	t.Log("Testing health endpoint...")
	healthRR := httptest.NewRequest("GET", "/health", nil)
	healthRecorder := httptest.NewRecorder()
	httpHandler.HealthCheck(healthRecorder, healthRR)

	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("Health endpoint returned status %d, expected %d", healthRecorder.Code, http.StatusOK)
	}

	var healthResp map[string]any
	if err := json.Unmarshal(healthRecorder.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	// Verify required health endpoint fields
	if status, ok := healthResp["status"].(string); !ok {
		t.Error("Health response missing status field")
	} else if status != "healthy" {
		t.Errorf("Unexpected health status: %s, expected healthy", status)
	}

	if _, ok := healthResp["data"]; !ok {
		t.Error("Health response missing data field")
	}

	t.Log("Health endpoint responded correctly")

	// 4. Test database export endpoint
	t.Log("Testing database export endpoint...")
	exportRR := httptest.NewRequest("GET", "/v1/medicaments/export", nil)
	exportRecorder := httptest.NewRecorder()
	httpHandler.ExportMedicaments(exportRecorder, exportRR)

	if exportRecorder.Code != http.StatusOK {
		t.Fatalf("Database export endpoint returned status %d, expected %d", exportRecorder.Code, http.StatusOK)
	}

	var exportResp []entities.Medicament
	if err := json.Unmarshal(exportRecorder.Body.Bytes(), &exportResp); err != nil {
		t.Fatalf("Failed to decode export response: %v", err)
	}

	if len(exportResp) != 1 {
		t.Errorf("Expected 1 medicament in export, got %d", len(exportResp))
	}

	t.Log("Database export endpoint responded correctly")

	t.Log("Smoke test completed successfully")
}
