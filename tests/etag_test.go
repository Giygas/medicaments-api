package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/health"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-chi/chi/v5"
)

func TestETagFunctionality(t *testing.T) {
	// NOTE: ETag functionality is implemented in most v1 endpoints and ExportMedicaments:
	// - ExportMedicaments (old /database endpoint)
	// - FindMedicamentByCIP (both old and v1)
	// - ServePresentationsV1
	// - ServeGeneriquesV1 (group and libelle)
	// - ServeMedicamentsV1 (page and search)
	// This test demonstrates ETag functionality using FindMedicamentByCIP as an example

	// Initialize test data
	dataContainer := data.NewDataContainer()

	// Create test medicament with presentation (for CIP testing)
	testMedicaments := []entities.Medicament{
		{
			Cis:                   10000001, // Fixed to valid 8-digit CIS
			Denomination:          "Test Medicament",
			FormePharmaceutique:   "comprimé",
			VoiesAdministration:   []string{"orale"},
			StatusAutorisation:    "Autorisation active",
			TypeProcedure:         "Procédure nationale",
			EtatComercialisation:  "Commercialisée",
			DateAMM:               "2023-01-01",
			Titulaire:             "Test Lab",
			SurveillanceRenforcee: "Non",
			Presentation: []entities.Presentation{
				{
					Cis:                  10000001, // Presentation CIS must match medicament CIS
					Cip7:                 1234567,
					Cip13:                1234567890123,
					Libelle:              "Test Presentation",
					StatusAdministratif:  "Présentation active",
					EtatComercialisation: "Commercialisée",
					DateDeclaration:      "2023-01-01",
				},
			},
		},
		{
			Cis:                   20000002, // Second medicament for CIP test
			Denomination:          "Test Medicament 2",
			FormePharmaceutique:   "comprimé",
			VoiesAdministration:   []string{"orale"},
			StatusAutorisation:    "Autorisation active",
			TypeProcedure:         "Procédure nationale",
			EtatComercialisation:  "Commercialisée",
			DateAMM:               "2023-01-01",
			Titulaire:             "Test Lab",
			SurveillanceRenforcee: "Non",
			Presentation: []entities.Presentation{
				{
					Cis:                  20000002, // Presentation CIS must match medicament CIS
					Cip7:                 7654321,
					Cip13:                7654321098765,
					Libelle:              "Test Presentation 2",
					StatusAdministratif:  "Présentation active",
					EtatComercialisation: "Commercialisée",
					DateDeclaration:      "2023-01-01",
				},
			},
		},
	}

	// Update data container with all test data
	medicamentsMap := make(map[int]entities.Medicament)
	for _, med := range testMedicaments {
		medicamentsMap[med.Cis] = med
	}

	presentationsCIP7Map := make(map[int]entities.Presentation)
	presentationsCIP13Map := make(map[int]entities.Presentation)
	for _, med := range testMedicaments {
		for _, pres := range med.Presentation {
			presentationsCIP7Map[pres.Cip7] = pres
			presentationsCIP13Map[pres.Cip13] = pres
		}
	}

	dataContainer.UpdateData(testMedicaments, []entities.GeneriqueList{},
		medicamentsMap,
		map[int]entities.GeneriqueList{},
		presentationsCIP7Map,
		presentationsCIP13Map, &interfaces.DataQualityReport{
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

	validator := validation.NewDataValidator()
	healthChecker := health.NewHealthChecker(dataContainer)
	httpHandler := handlers.NewHTTPHandler(dataContainer, validator, healthChecker)

	// Test FindMedicamentByCIP with ETag (this endpoint supports ETag)
	cipHandler := httpHandler.FindMedicamentByCIP

	// Create a chi router to properly set path values
	router := chi.NewRouter()
	router.Get("/medicament/cip/{cip}", cipHandler)

	// First request - should return 200 with ETag
	req1 := httptest.NewRequest("GET", "/medicament/cip/1234567", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	resp1 := w1.Result()
	etag1 := resp1.Header.Get("ETag")

	if etag1 == "" {
		t.Error("Expected ETag header in first response")
	}

	if resp1.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp1.StatusCode)
	}

	// Second request with If-None-Match - should return 304
	req2 := httptest.NewRequest("GET", "/medicament/cip/1234567", nil)
	req2.Header.Set("If-None-Match", etag1)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	resp2 := w2.Result()

	if resp2.StatusCode != http.StatusNotModified {
		t.Errorf("Expected status 304 for CIP with matching ETag, got %d", resp2.StatusCode)
	}

	// Verify no body in 304 response
	body2 := w2.Body.String()
	if body2 != "" {
		t.Error("Expected empty body for 304 response")
	}

	// Test with different ETag - should return 200
	req3 := httptest.NewRequest("GET", "/medicament/cip/1234567", nil)
	req3.Header.Set("If-None-Match", `"different-etag"`)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	resp3 := w3.Result()

	if resp3.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for different ETag, got %d", resp3.StatusCode)
	}
}
