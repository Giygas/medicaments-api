package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

// ============================================================================
// V1 ENDPOINTS TESTS
// ============================================================================

// ============================================================================
// PRESENTATIONS V1 TESTS
// ============================================================================

// TestServePresentationsV1_Success tests successful presentation lookups
func TestServePresentationsV1_Success(t *testing.T) {
	presentationCIP7 := entities.Presentation{
		Cis:                  1,
		Cip7:                 1234567,
		Cip13:                1234567890123,
		Libelle:              "Boîte de 8 comprimés",
		StatusAdministratif:  "Présentation commercialisée",
		EtatComercialisation: "Commercialisée",
		DateDeclaration:      "2020-02-01",
	}

	presentationCIP13 := entities.Presentation{
		Cis:                  2,
		Cip7:                 7654321,
		Cip13:                7654321098765,
		Libelle:              "Flacon de 100ml",
		StatusAdministratif:  "Présentation commercialisée",
		EtatComercialisation: "Commercialisée",
		DateDeclaration:      "2020-03-01",
	}

	tests := []struct {
		name        string
		pathParam   string
		cip7Map     map[int]entities.Presentation
		cip13Map    map[int]entities.Presentation
		expectedCIP int
	}{
		{"presentation found via CIP7", "1234567",
			map[int]entities.Presentation{1234567: presentationCIP7},
			map[int]entities.Presentation{},
			1234567},
		{"presentation found via CIP13", "7654321098765",
			map[int]entities.Presentation{},
			map[int]entities.Presentation{7654321098765: presentationCIP13},
			7654321098765},
		{"CIP7 preferred over CIP13", "1234567890123",
			map[int]entities.Presentation{1234567890123: presentationCIP7},
			map[int]entities.Presentation{1234567890123: presentationCIP13},
			1234567890123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().
					WithPresentationsCIP7Map(tt.cip7Map).
					WithPresentationsCIP13Map(tt.cip13Map).
					Build(),
				NewMockDataValidatorBuilder().Build(),
			)

			router := chi.NewRouter()
			router.Get("/v1/presentations/{cip}", handler.ServePresentationsV1)

			req := httptest.NewRequest("GET", "/v1/presentations/"+tt.pathParam, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Simple: only check success case
			if rr.Code != http.StatusOK {
				t.Errorf("Expected 200 OK, got %d", rr.Code)
			}

			// Verify ETag headers
			etag := rr.Header().Get("ETag")
			if etag == "" {
				t.Error("ETag header should be present")
			}
			if !hasQuotedETag(etag) {
				t.Errorf("ETag should be quoted, got: %s", etag)
			}

			// Verify cache headers
			if rr.Header().Get("Cache-Control") != "public, max-age=3600" {
				t.Error("Expected Cache-Control 'public, max-age=3600'")
			}
			if rr.Header().Get("Last-Modified") == "" {
				t.Error("Expected Last-Modified header")
			}

			// Verify response contains expected CIP
			var response entities.Presentation
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if response.Cip7 != tt.expectedCIP && response.Cip13 != tt.expectedCIP {
				t.Errorf("Expected CIP %d, got CIP7: %d, CIP13: %d",
					tt.expectedCIP, response.Cip7, response.Cip13)
			}
		})
	}
}

// TestServePresentationsV1_Errors tests error cases for presentation lookup
func TestServePresentationsV1_Errors(t *testing.T) {
	tests := []struct {
		name         string
		pathParam    string
		expectedCode int
		expectError  string
	}{
		{"non-numeric CIP", "abc123", http.StatusBadRequest, "CIP should have 7 or 13 characters"},
		{"presentation not found", "9999999", http.StatusNotFound, "Presentation not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().Build(),
				NewMockDataValidatorBuilder().Build(),
			)

			router := chi.NewRouter()
			router.Get("/v1/presentations/{cip}", handler.ServePresentationsV1)

			req := httptest.NewRequest("GET", "/v1/presentations/"+tt.pathParam, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Simple: only check error case
			if rr.Code != tt.expectedCode {
				t.Errorf("Expected %d, got %d", tt.expectedCode, rr.Code)
			}

			var response map[string]any
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			message, ok := response["message"].(string)
			if !ok || message != tt.expectError {
				t.Errorf("Expected error '%s', got '%v'", tt.expectError, response["message"])
			}
		})
	}
}

// TestServePresentationsV1_ETagCaching tests ETag caching functionality
func TestServePresentationsV1_ETagCaching(t *testing.T) {
	presentation := entities.Presentation{
		Cis:                  1,
		Cip7:                 1234567,
		Cip13:                1234567890123,
		Libelle:              "Boîte de 8 comprimés",
		StatusAdministratif:  "Présentation commercialisée",
		EtatComercialisation: "Commercialisée",
		DateDeclaration:      "2020-02-01",
	}

	handler := NewHTTPHandler(
		NewMockDataStoreBuilder().
			WithPresentationsCIP7Map(map[int]entities.Presentation{1234567: presentation}).
			Build(),
		NewMockDataValidatorBuilder().Build(),
	).(*HTTPHandlerImpl)

	router := chi.NewRouter()
	router.Get("/v1/presentations/{cip}", handler.ServePresentationsV1)

	// First request - should return full response with ETag
	req1 := httptest.NewRequest("GET", "/v1/presentations/1234567", nil)
	rr1 := httptest.NewRecorder()

	router.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr1.Code)
	}

	etag := rr1.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header should be present")
	}

	if !hasQuotedETag(etag) {
		t.Errorf("ETag should be quoted, got: %s", etag)
	}

	// Second request with matching ETag - should return 304 Not Modified
	req2 := httptest.NewRequest("GET", "/v1/presentations/1234567", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()

	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("Expected 304 Not Modified, got %d", rr2.Code)
	}

	// Third request with different ETag - should return 200
	req3 := httptest.NewRequest("GET", "/v1/presentations/1234567", nil)
	req3.Header.Set("If-None-Match", `"different-etag"`)
	rr3 := httptest.NewRecorder()

	router.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr3.Code)
	}

	if rr3.Body.Len() == 0 {
		t.Error("Response body should be present")
	}
}

// ============================================================================
// GENERIQUES V1 TESTS
// ============================================================================

// TestServeGeneriquesV1_Success tests successful generic lookups
func TestServeGeneriquesV1_Success(t *testing.T) {
	genericList := []entities.GeneriqueList{
		{
			GroupID: 1,
			Libelle: "Paracetamol 500 mg + Codeine (Phosphate) hemihydrate 30 mg",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 1, Denomination: "PARACETAMOL/CODEINE BIOGARAN"},
				{Cis: 2, Denomination: "DAFALGAN CODEINE"},
			},
		},
		{
			GroupID: 2,
			Libelle: "Ibuprofene 400 mg",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 3, Denomination: "IBUPROFENE BIOGARAN"},
				{Cis: 4, Denomination: "NUROFEN"},
			},
		},
	}

	generiquesMap := map[int]entities.GeneriqueList{
		1: {GroupID: 1, Libelle: "Paracetamol 500 mg + Codeine (Phosphate) hemihydrate 30 mg"},
		2: {GroupID: 2, Libelle: "Ibuprofene 400 mg"},
	}

	tests := []struct {
		name         string
		queryParams  string
		checkGroupID int
		checkLibelle string
	}{
		{"valid group ID", "?group=1", 1, ""},
		{"exact libelle match", "?libelle=Paracetamol+500+mg+%2B+Codeine", 0, "Paracetamol 500 mg + Codeine"},
		{"case-insensitive libelle", "?libelle=paracetamol+500+mg+%2B+codeine", 0, "Paracetamol 500 mg + Codeine"},
		{"partial libelle match", "?libelle=Ibuprofene", 0, "Ibuprofene"},
		{"partial libelle lowercase", "?libelle=ibuprofene", 0, "Ibuprofene"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().
					WithGeneriques(genericList).
					WithGeneriquesMap(generiquesMap).
					Build(),
				NewMockDataValidatorBuilder().Build(),
			)

			req := httptest.NewRequest("GET", "/v1/generiques"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.ServeGeneriquesV1(rr, req)

			// Simple: only check success case
			if rr.Code != http.StatusOK {
				t.Errorf("Expected 200 OK, got %d", rr.Code)
			}

			// Group search: verify ETag headers
			if tt.checkGroupID != 0 {
				etag := rr.Header().Get("ETag")
				if etag == "" {
					t.Error("ETag header should be present for group search")
				}
				if !hasQuotedETag(etag) {
					t.Errorf("ETag should be quoted, got: %s", etag)
				}
				if rr.Header().Get("Cache-Control") != "public, max-age=3600" {
					t.Error("Expected Cache-Control 'public, max-age=3600'")
				}
				if rr.Header().Get("Last-Modified") == "" {
					t.Error("Expected Last-Modified header")
				}

				var response entities.GeneriqueList
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}

				if response.GroupID != tt.checkGroupID {
					t.Errorf("Expected group ID %d, got %d", tt.checkGroupID, response.GroupID)
				}

				if response.Libelle == "" {
					t.Error("Response should contain Libelle field")
				}
			}

			// Libelle search: verify array response
			if tt.checkLibelle != "" {
				var response []entities.GeneriqueList
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}

				if len(response) == 0 {
					t.Error("Expected at least one result, got empty array")
				}

				found := false
				for _, gen := range response {
					if containsSubstring(gen.Libelle, tt.checkLibelle) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find libelle containing '%s', got %v", tt.checkLibelle, response)
				}
			}
		})
	}
}

// TestServeGeneriquesV1_Errors tests error cases for generic lookups
func TestServeGeneriquesV1_Errors(t *testing.T) {
	genericList := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Test", Medicaments: []entities.GeneriqueMedicament{}},
		{GroupID: 9999, Libelle: "Boundary Test", Medicaments: []entities.GeneriqueMedicament{}},
	}

	tests := []struct {
		name          string
		queryParams   string
		setValidation bool
		expectedCode  int
		expectError   string
	}{
		{"no parameters", "", false, http.StatusBadRequest, "Needs libelle or group param"},
		{"empty group", "?group=", false, http.StatusBadRequest, "Needs libelle or group param"},
		{"invalid group", "?group=abc", false, http.StatusBadRequest, "Invalid group ID"},
		{"negative group ID", "?group=-1", false, http.StatusBadRequest, "Group ID should be between 1 and 9999"},
		{"zero group ID", "?group=0", false, http.StatusBadRequest, "Group ID should be between 1 and 9999"},
		{"group ID too high", "?group=10000", false, http.StatusBadRequest, "Group ID should be between 1 and 9999"},
		{"valid boundary group ID", "?group=9999", false, http.StatusNotFound, "Generique group not found"},
		{"not found", "?group=999", false, http.StatusNotFound, "Generique group not found"},
		{"empty libelle", "?libelle=", false, http.StatusBadRequest, "Needs libelle or group param"},
		{"invalid libelle", "?libelle=test@123", true, http.StatusBadRequest, "input must be between"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validator *MockDataValidator
			if tt.setValidation {
				validator = NewMockDataValidatorBuilder().
					WithInputError(fmt.Errorf("input must be between 3 and 50 characters")).
					Build()
			} else {
				validator = NewMockDataValidatorBuilder().Build()
			}

			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().
					WithGeneriques(genericList).
					Build(),
				validator,
			)

			req := httptest.NewRequest("GET", "/v1/generiques"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.ServeGeneriquesV1(rr, req)

			// Simple: only check error case
			if rr.Code != tt.expectedCode {
				t.Errorf("Expected %d, got %d", tt.expectedCode, rr.Code)
			}

			var response map[string]any
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			message, ok := response["message"].(string)
			if !ok || !containsSubstring(message, tt.expectError) {
				t.Errorf("Expected error containing '%s', got '%v'", tt.expectError, message)
			}
		})
	}
}

// TestServeGeneriquesV1_ETagCaching tests ETag caching for group search
func TestServeGeneriquesV1_ETagCaching(t *testing.T) {
	genericList := []entities.GeneriqueList{
		{
			GroupID: 1,
			Libelle: "Paracetamol 500 mg",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 1, Denomination: "PARACETAMOL BIOGARAN"},
			},
		},
	}

	handler := NewHTTPHandler(
		NewMockDataStoreBuilder().
			WithGeneriques(genericList).
			WithGeneriquesMap(map[int]entities.GeneriqueList{
				1: {GroupID: 1, Libelle: "Paracetamol 500 mg"},
			}).
			Build(),
		NewMockDataValidatorBuilder().Build(),
	).(*HTTPHandlerImpl)

	// First request - generate ETag
	req1 := httptest.NewRequest("GET", "/v1/generiques?group=1", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeGeneriquesV1(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr1.Code)
	}

	etag := rr1.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header should be present")
	}

	if !hasQuotedETag(etag) {
		t.Errorf("ETag should be quoted, got: %s", etag)
	}

	// Second request with matching ETag - return 304
	req2 := httptest.NewRequest("GET", "/v1/generiques?group=1", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	handler.ServeGeneriquesV1(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("Expected 304 Not Modified, got %d", rr2.Code)
	}

	// Third request with different ETag - return 200
	req3 := httptest.NewRequest("GET", "/v1/generiques?group=1", nil)
	req3.Header.Set("If-None-Match", `"different-etag"`)
	rr3 := httptest.NewRecorder()
	handler.ServeGeneriquesV1(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr3.Code)
	}

	if rr3.Body.Len() == 0 {
		t.Error("Response body should be present")
	}
}

// ============================================================================
// MEDICAMENTS V1 TESTS
// ============================================================================

// TestServeMedicamentsV1_Success tests successful medicament lookups
func TestServeMedicamentsV1_Success(t *testing.T) {
	// Create test medicaments with presentations for CIP search
	med1 := entities.Medicament{
		Cis:                  10000001,
		Denomination:         "PARACETAMOL 500 mg",
		FormePharmaceutique:  "Comprimé",
		VoiesAdministration:  []string{"Orale"},
		StatusAutorisation:   "Autorisation active",
		TypeProcedure:        "Procédure nationale",
		EtatComercialisation: "Commercialisée",
		DateAMM:              "2020-01-01",
		Titulaire:            "SANOFI",
		Presentation: []entities.Presentation{
			{Cis: 10000001, Cip7: 1234567, Cip13: 1234567890123, Libelle: "Boîte de 8 comprimés"},
		},
	}

	med2 := entities.Medicament{
		Cis:                  10000002,
		Denomination:         "IBUPROFENE 400 mg",
		FormePharmaceutique:  "Comprimé",
		VoiesAdministration:  []string{"Orale"},
		StatusAutorisation:   "Autorisation active",
		TypeProcedure:        "Procédure nationale",
		EtatComercialisation: "Commercialisée",
		DateAMM:              "2020-02-01",
		Titulaire:            "PFIZER",
		Presentation: []entities.Presentation{
			{Cis: 10000002, Cip7: 7654321, Cip13: 7654321098765, Libelle: "Boîte de 10 comprimés"},
		},
	}

	med3 := entities.Medicament{
		Cis:                  10000003,
		Denomination:         "ASPIRINE 100 mg",
		FormePharmaceutique:  "Comprimé",
		VoiesAdministration:  []string{"Orale"},
		StatusAutorisation:   "Autorisation active",
		TypeProcedure:        "Procédure nationale",
		EtatComercialisation: "Commercialisée",
		DateAMM:              "2020-03-01",
		Titulaire:            "BAYER",
		Presentation:         []entities.Presentation{},
	}

	// Create medicaments array (15 items to test pagination with multiple pages)
	medicaments := []entities.Medicament{med1, med2, med3}
	for i := 4; i <= 15; i++ {
		medicaments = append(medicaments, entities.Medicament{
			Cis:                  10000000 + i,
			Denomination:         fmt.Sprintf("MEDICAMENT %d", i),
			FormePharmaceutique:  "Comprimé",
			VoiesAdministration:  []string{"Orale"},
			StatusAutorisation:   "Autorisation active",
			TypeProcedure:        "Procédure nationale",
			EtatComercialisation: "Commercialisée",
			DateAMM:              "2020-01-01",
			Titulaire:            "TEST",
			Presentation:         []entities.Presentation{},
		})
	}

	// Create presentation maps for O(1) lookups (needed for CIP search)
	presentationsCIP7Map := map[int]entities.Presentation{
		1234567: med1.Presentation[0],
		7654321: med2.Presentation[0],
	}

	presentationsCIP13Map := map[int]entities.Presentation{
		1234567890123: med1.Presentation[0],
		7654321098765: med2.Presentation[0],
	}

	tests := []struct {
		name          string
		queryParams   string
		checkType     string // "page", "search", "cis", "cip"
		expectedValue any
	}{
		{"first page", "?page=1", "page", 10},
		{"second page", "?page=2", "page", 5},
		{"search with results", "?search=paracetamol", "search", "PARACETAMOL 500 mg"},
		{"search no results", "?search=unknown", "search", nil},
		{"lookup by CIS", "?cis=10000001", "cis", "PARACETAMOL 500 mg"},
		{"lookup by CIP7", "?cip=1234567", "cip", "PARACETAMOL 500 mg"},
		{"lookup by CIP13", "?cip=7654321098765", "cip", "IBUPROFENE 400 mg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().
					WithMedicaments(medicaments).
					WithPresentationsCIP7Map(presentationsCIP7Map).
					WithPresentationsCIP13Map(presentationsCIP13Map).
					Build(),
				NewMockDataValidatorBuilder().Build(),
			).(*HTTPHandlerImpl)

			req := httptest.NewRequest("GET", "/v1/medicaments"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.ServeMedicamentsV1(rr, req)

			// Simple: only check success case
			if rr.Code != http.StatusOK {
				t.Errorf("Expected 200 OK, got %d", rr.Code)
			}

			// Verify based on type
			switch tt.checkType {
			case "export":
				var response []entities.Medicament
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}
				if len(response) != tt.expectedValue.(int) {
					t.Errorf("Expected %d medicaments, got %d", tt.expectedValue, len(response))
				}

				// Verify ETag headers
				etag := rr.Header().Get("ETag")
				if etag == "" {
					t.Error("ETag header should be present for export")
				}
				if !hasQuotedETag(etag) {
					t.Errorf("ETag should be quoted, got: %s", etag)
				}
				if rr.Header().Get("Cache-Control") != "public, max-age=3600" {
					t.Error("Expected Cache-Control 'public, max-age=3600'")
				}
				if rr.Header().Get("Last-Modified") == "" {
					t.Error("Expected Last-Modified header")
				}

			case "page":
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}

				data, ok := response["data"].([]any)
				if !ok {
					t.Error("Expected 'data' field to be an array")
				}
				if len(data) != tt.expectedValue.(int) {
					t.Errorf("Expected %d items on page, got %d", tt.expectedValue, len(data))
				}

			case "search":
				var response []entities.Medicament
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}

				if tt.expectedValue == nil {
					// Search with no results
					if len(response) != 0 {
						t.Errorf("Expected empty array, got %d results", len(response))
					}
				} else {
					// Search with results
					if len(response) == 0 {
						t.Error("Expected at least one result, got empty array")
					}
					found := false
					for _, med := range response {
						if containsSubstring(med.Denomination, tt.expectedValue.(string)) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find medicament containing '%s', got %v", tt.expectedValue, response)
					}

					// Verify ETag headers for search with results
					etag := rr.Header().Get("ETag")
					if etag == "" {
						t.Error("ETag header should be present for search with results")
					}
					if !hasQuotedETag(etag) {
						t.Errorf("ETag should be quoted, got: %s", etag)
					}
					if rr.Header().Get("Cache-Control") != "public, max-age=3600" {
						t.Error("Expected Cache-Control 'public, max-age=3600'")
					}
					if rr.Header().Get("Last-Modified") == "" {
						t.Error("Expected Last-Modified header")
					}
				}

			case "cis":
				// CIS lookup (O(1) map lookup)
				var response entities.Medicament
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}

				if response.Denomination != tt.expectedValue.(string) {
					t.Errorf("Expected denomination '%s', got '%s'", tt.expectedValue, response.Denomination)
				}
				if rr.Header().Get("ETag") != "" {
					t.Error("ETag header should not be present for CIS lookup")
				}

			case "cip":
				// CIP lookup (O(1) map lookup)
				var response entities.Medicament
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}

				if response.Denomination != tt.expectedValue.(string) {
					t.Errorf("Expected denomination '%s', got '%s'", tt.expectedValue, response.Denomination)
				}
				if rr.Header().Get("ETag") != "" {
					t.Error("ETag header should not be present for CIP lookup")
				}
			}
		})
	}
}

// TestServeMedicamentsV1_Errors tests error cases for medicament lookup
func TestServeMedicamentsV1_Errors(t *testing.T) {
	// Create minimal test data
	med1 := entities.Medicament{
		Cis:                 10000001,
		Denomination:        "PARACETAMOL",
		FormePharmaceutique: "Comprimé",
		VoiesAdministration: []string{"Orale"},
		Presentation: []entities.Presentation{
			{Cis: 10000001, Cip7: 1234567, Cip13: 1234567890123, Libelle: "Boîte de 8"},
		},
	}

	medicaments := []entities.Medicament{med1}

	tests := []struct {
		name          string
		queryParams   string
		setValidation bool
		expectedCode  int
		expectError   string
	}{
		{"no parameters", "", false, http.StatusBadRequest, "Needs at least one param"},
		{"multiple parameters", "?page=1&cis=10000001", false, http.StatusBadRequest, "Only one parameter allowed"},
		{"invalid page zero", "?page=0", false, http.StatusBadRequest, "Invalid page number"},
		{"invalid page negative", "?page=-1", false, http.StatusBadRequest, "Invalid page number"},
		{"invalid page non-numeric", "?page=abc", false, http.StatusBadRequest, "Invalid page number"},
		{"page not found", "?page=999", false, http.StatusNotFound, "Page not found"},
		{"invalid CIS non-numeric", "?cis=abc12345", false, http.StatusBadRequest, "input contains invalid characters"},
		{"invalid CIP length", "?cip=123", false, http.StatusBadRequest, "CIP should have 7 or 13 characters"},
		{"invalid search input", "?search=test@123", true, http.StatusBadRequest, "input must be between"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validator *MockDataValidator
			if tt.setValidation {
				validator = NewMockDataValidatorBuilder().
					WithInputError(fmt.Errorf("input must be between 3 and 50 characters")).
					Build()
			} else {
				validator = NewMockDataValidatorBuilder().Build()
			}

			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().
					WithMedicaments(medicaments).
					Build(),
				validator,
			)

			req := httptest.NewRequest("GET", "/v1/medicaments"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.ServeMedicamentsV1(rr, req)

			// Simple: only check error case
			if rr.Code != tt.expectedCode {
				t.Errorf("Expected %d, got %d", tt.expectedCode, rr.Code)
			}

			var response map[string]any
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			message, ok := response["message"].(string)
			if !ok || !containsSubstring(message, tt.expectError) {
				t.Errorf("Expected error containing '%s', got '%v'", tt.expectError, message)
			}
		})
	}
}

// TestServeMedicamentsV1_ETagCaching tests ETag caching for export and search endpoints
func TestServeMedicamentsV1_ETagCaching(t *testing.T) {
	// Create test medicaments with presentations for CIP search
	med1 := entities.Medicament{
		Cis:                  10000001,
		Denomination:         "PARACETAMOL 500 mg",
		FormePharmaceutique:  "Comprimé",
		VoiesAdministration:  []string{"Orale"},
		StatusAutorisation:   "Autorisation active",
		TypeProcedure:        "Procédure nationale",
		EtatComercialisation: "Commercialisée",
		DateAMM:              "2020-01-01",
		Titulaire:            "SANOFI",
		Presentation: []entities.Presentation{
			{Cis: 10000001, Cip7: 1234567, Cip13: 1234567890123, Libelle: "Boîte de 8 comprimés"},
		},
	}

	med2 := entities.Medicament{
		Cis:                  10000002,
		Denomination:         "IBUPROFENE 400 mg",
		FormePharmaceutique:  "Comprimé",
		VoiesAdministration:  []string{"Orale"},
		StatusAutorisation:   "Autorisation active",
		TypeProcedure:        "Procédure nationale",
		EtatComercialisation: "Commercialisée",
		DateAMM:              "2020-02-01",
		Titulaire:            "PFIZER",
		Presentation: []entities.Presentation{
			{Cis: 10000002, Cip7: 7654321, Cip13: 7654321098765, Libelle: "Boîte de 10 comprimés"},
		},
	}

	medicaments := []entities.Medicament{med1, med2}

	handler := NewHTTPHandler(
		NewMockDataStoreBuilder().
			WithMedicaments(medicaments).
			Build(),
		NewMockDataValidatorBuilder().Build(),
	).(*HTTPHandlerImpl)

	// Test Search ETag caching
	t.Run("search ETag caching", func(t *testing.T) {
		// First request - generate ETag
		req1 := httptest.NewRequest("GET", "/v1/medicaments?search=paracetamol", nil)
		rr1 := httptest.NewRecorder()
		handler.ServeMedicamentsV1(rr1, req1)

		if rr1.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rr1.Code)
		}

		etag := rr1.Header().Get("ETag")
		if etag == "" {
			t.Error("ETag header should be present")
		}

		if !hasQuotedETag(etag) {
			t.Errorf("ETag should be quoted, got: %s", etag)
		}

		// Second request with matching ETag - return 304
		req2 := httptest.NewRequest("GET", "/v1/medicaments?search=paracetamol", nil)
		req2.Header.Set("If-None-Match", etag)
		rr2 := httptest.NewRecorder()
		handler.ServeMedicamentsV1(rr2, req2)

		if rr2.Code != http.StatusNotModified {
			t.Errorf("Expected 304 Not Modified, got %d", rr2.Code)
		}

		// Third request with different ETag - return 200
		req3 := httptest.NewRequest("GET", "/v1/medicaments?search=paracetamol", nil)
		req3.Header.Set("If-None-Match", `"different-etag"`)
		rr3 := httptest.NewRecorder()
		handler.ServeMedicamentsV1(rr3, req3)

		if rr3.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rr3.Code)
		}

		if rr3.Body.Len() == 0 {
			t.Error("Response body should be present")
		}
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// hasQuotedETag checks if an ETag is properly quoted
func hasQuotedETag(etag string) bool {
	return len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"'
}

// containsSubstring checks if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

// findSubstring finds a substring in a string
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
