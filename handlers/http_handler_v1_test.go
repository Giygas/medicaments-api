package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
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
				NewMockHealthCheckerBuilder().Build(),
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
				NewMockHealthCheckerBuilder().Build(),
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
		NewMockHealthCheckerBuilder().Build(),
	).(*Handler)

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

// TestServeGeneriqueByGroupIDV1 tests to v1 path-based group ID lookup
func TestServeGeneriqueByGroupIDV1(t *testing.T) {
	generiquesMap := map[int]entities.GeneriqueList{
		100: {
			GroupID:           100,
			Libelle:           "Test Generique",
			LibelleNormalized: "test generique",
			Medicaments:       []entities.GeneriqueMedicament{},
		},
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid group ID",
			path:           "/v1/generiques/100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid group ID - non-numeric",
			path:           "/v1/generiques/invalid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid group ID",
		},
		{
			name:           "Invalid group ID - not found",
			path:           "/v1/generiques/99999",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Generique group not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().
					WithGeneriquesMap(generiquesMap).
					Build(),
				NewMockDataValidatorBuilder().Build(),
				NewMockHealthCheckerBuilder().Build(),
			)

			router := chi.NewRouter()
			router.Get("/v1/generiques/{groupID}", handler.FindGeneriquesByGroupID)

			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" {
				body := w.Body.String()
				if !strings.Contains(body, tt.expectedBody) {
					t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, body)
				}
			}

			// Verify no deprecation headers on v1 path
			if strings.HasPrefix(tt.path, "/v1/") {
				deprecation := w.Header().Get("Deprecation")
				if deprecation != "" {
					t.Error("V1 endpoint should not have deprecation headers")
				}
			}
		})
	}
}

// TestServeGeneriquesV1_Success tests successful generic lookups
func TestServeGeneriquesV1_Success(t *testing.T) {
	genericList := []entities.GeneriqueList{
		{
			GroupID:           1,
			Libelle:           "Paracetamol 500 mg + Codeine (Phosphate) hemihydrate 30 mg",
			LibelleNormalized: strings.ReplaceAll(strings.ToLower("Paracetamol 500 mg + Codeine (Phosphate) hemihydrate 30 mg"), "+", " "),
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 1, Denomination: "PARACETAMOL/CODEINE BIOGARAN"},
				{Cis: 2, Denomination: "DAFALGAN CODEINE"},
			},
		},
		{
			GroupID:           2,
			Libelle:           "Ibuprofene 400 mg",
			LibelleNormalized: strings.ReplaceAll(strings.ToLower("Ibuprofene 400 mg"), "+", " "),
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 3, Denomination: "IBUPROFENE BIOGARAN"},
				{Cis: 4, Denomination: "NUROFEN"},
			},
		},
	}

	tests := []struct {
		name         string
		queryParams  string
		checkLibelle string
	}{
		{"exact libelle match", "?libelle=Paracetamol+500+mg+%2B+Codeine", "Paracetamol 500 mg + Codeine"},
		{"case-insensitive libelle", "?libelle=paracetamol+500+mg+%2B+codeine", "Paracetamol 500 mg + Codeine"},
		{"partial libelle match", "?libelle=Ibuprofene", "Ibuprofene"},
		{"partial libelle lowercase", "?libelle=ibuprofene", "Ibuprofene"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().
					WithGeneriques(genericList).
					Build(),
				NewMockDataValidatorBuilder().Build(),
				NewMockHealthCheckerBuilder().Build(),
			)

			req := httptest.NewRequest("GET", "/v1/generiques"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.ServeGeneriquesV1(rr, req)

			// Simple: only check success case
			if rr.Code != http.StatusOK {
				t.Errorf("Expected 200 OK, got %d", rr.Code)
			}

			// Libelle search: verify array response
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
		{"no parameters", "", false, http.StatusBadRequest, "Needs libelle param"},
		{"empty libelle", "?libelle=", false, http.StatusBadRequest, "Needs libelle param"},
		{"invalid libelle", "?libelle=test@123", true, http.StatusBadRequest, "input must be between"},
		{"not found", "?libelle=xyz123", false, http.StatusNotFound, "No generiques found"},
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
				NewMockHealthCheckerBuilder().Build(),
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

// ============================================================================
// MEDICAMENTS V1 TESTS
// ============================================================================

// TestServeMedicamentsV1_Success tests successful medicament lookups
func TestServeMedicamentsV1_Success(t *testing.T) {
	// Create test medicaments with presentations for CIP search
	med1 := entities.Medicament{
		Cis:                    10000001,
		Denomination:           "PARACETAMOL 500 mg",
		DenominationNormalized: strings.ReplaceAll(strings.ToLower("PARACETAMOL 500 mg"), "+", " "),
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-01-01",
		Titulaire:              "SANOFI",
		Presentation: []entities.Presentation{
			{Cis: 10000001, Cip7: 1234567, Cip13: 1234567890123, Libelle: "Boîte de 8 comprimés"},
		},
	}

	med2 := entities.Medicament{
		Cis:                    10000002,
		Denomination:           "IBUPROFENE 400 mg",
		DenominationNormalized: strings.ReplaceAll(strings.ToLower("IBUPROFENE 400 mg"), "+", " "),
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-02-01",
		Titulaire:              "PFIZER",
		Presentation: []entities.Presentation{
			{Cis: 10000002, Cip7: 7654321, Cip13: 7654321098765, Libelle: "Boîte de 10 comprimés"},
		},
	}

	med3 := entities.Medicament{
		Cis:                    10000003,
		Denomination:           "ASPIRINE 100 mg",
		DenominationNormalized: strings.ReplaceAll(strings.ToLower("ASPIRINE 100 mg"), "+", " "),
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-03-01",
		Titulaire:              "BAYER",
		Presentation:           []entities.Presentation{},
	}

	// Create medicaments array (15 items to test pagination with multiple pages)
	medicaments := []entities.Medicament{med1, med2, med3}
	for i := 4; i <= 15; i++ {
		denom := fmt.Sprintf("MEDICAMENT %d", i)
		medicaments = append(medicaments, entities.Medicament{
			Cis:                    10000000 + i,
			Denomination:           denom,
			DenominationNormalized: strings.ReplaceAll(strings.ToLower(denom), "+", " "),
			FormePharmaceutique:    "Comprimé",
			VoiesAdministration:    []string{"Orale"},
			StatusAutorisation:     "Autorisation active",
			TypeProcedure:          "Procédure nationale",
			EtatComercialisation:   "Commercialisée",
			DateAMM:                "2020-01-01",
			Titulaire:              "TEST",
			Presentation:           []entities.Presentation{},
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
		checkType     string
		expectedValue any
	}{
		{"first page", "?page=1", "page", 10},
		{"second page", "?page=2", "page", 5},
		{"search with results", "?search=paracetamol", "search", "PARACETAMOL 500 mg"},
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
				NewMockHealthCheckerBuilder().Build(),
			).(*Handler)

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
		Cis:                    10000001,
		Denomination:           "PARACETAMOL",
		DenominationNormalized: strings.ReplaceAll(strings.ToLower("PARACETAMOL"), "+", " "),
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
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
		{"multiple parameters", "?page=1&cip=1234567", false, http.StatusBadRequest, "Only one parameter allowed"},
		{"invalid page zero", "?page=0", false, http.StatusBadRequest, "Invalid page number"},
		{"invalid page negative", "?page=-1", false, http.StatusBadRequest, "Invalid page number"},
		{"invalid page non-numeric", "?page=abc", false, http.StatusBadRequest, "Invalid page number"},
		{"page not found", "?page=999", false, http.StatusNotFound, "Page not found"},
		{"invalid CIP length", "?cip=123", false, http.StatusBadRequest, "CIP should have 7 or 13 characters"},
		{"invalid search input", "?search=test@123", true, http.StatusBadRequest, "input must be between"},
		{"search not found", "?search=gripe", false, http.StatusNotFound, "No medicaments found"},
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
				NewMockHealthCheckerBuilder().Build(),
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
		Cis:                    10000001,
		Denomination:           "PARACETAMOL 500 mg",
		DenominationNormalized: strings.ReplaceAll(strings.ToLower("PARACETAMOL 500 mg"), "+", " "),
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-01-01",
		Titulaire:              "SANOFI",
		Presentation: []entities.Presentation{
			{Cis: 10000001, Cip7: 1234567, Cip13: 1234567890123, Libelle: "Boîte de 8 comprimés"},
		},
	}

	med2 := entities.Medicament{
		Cis:                    10000002,
		Denomination:           "IBUPROFENE 400 mg",
		DenominationNormalized: strings.ReplaceAll(strings.ToLower("IBUPROFENE 400 mg"), "+", " "),
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-02-01",
		Titulaire:              "PFIZER",
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
		NewMockHealthCheckerBuilder().Build(),
	).(*Handler)

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

// TestServeMedicamentsV1_MultiWordSearch tests multi-word search functionality
func TestServeMedicamentsV1_MultiWordSearch(t *testing.T) {
	// Create test medicaments with realistic multi-word names
	med1 := entities.Medicament{
		Cis:                    10000001,
		Denomination:           "PARACETAMOL 500 mg",
		DenominationNormalized: strings.ToLower("paracetamol 500 mg"),
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-01-01",
		Titulaire:              "SANOFI",
		Presentation:           []entities.Presentation{},
	}

	med2 := entities.Medicament{
		Cis:                    10000002,
		Denomination:           "IBUPROFENE ARROW CONSEIL 400 mg, caps",
		DenominationNormalized: strings.ToLower("ibuprofene arrow conseil 400 mg, caps"),
		FormePharmaceutique:    "Gélule",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-02-01",
		Titulaire:              "ARROW",
		Presentation:           []entities.Presentation{},
	}

	med3 := entities.Medicament{
		Cis:                    10000003,
		Denomination:           "IBUPROFENE 200 mg, comprimé enrobé",
		DenominationNormalized: "ibuprofene 200 mg, comprime enrobe",
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-03-01",
		Titulaire:              "PFIZER",
		Presentation:           []entities.Presentation{},
	}

	med4 := entities.Medicament{
		Cis:                    10000004,
		Denomination:           "AMOXICILLINE BIOGARAN 1 g",
		DenominationNormalized: "amoxicilline biogaran 1 g",
		FormePharmaceutique:    "Comprimé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisation active",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisée",
		DateAMM:                "2020-04-01",
		Titulaire:              "BIOGARAN",
		Presentation:           []entities.Presentation{},
	}

	medicaments := []entities.Medicament{med1, med2, med3, med4}

	tests := []struct {
		name          string
		queryParams   string
		expectedCount int
		expectedMatch string
		expectError   bool
	}{
		// 2-word searches
		{"2-word: paracetamol 500", "?search=paracetamol+500", 1, "PARACETAMOL 500 mg", false},
		{"2-word: ibuprofene 400", "?search=ibuprofene+400", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},
		{"2-word: ibuprofene 200", "?search=ibuprofene+200", 1, "IBUPROFENE 200 mg, comprimé enrobé", false},
		{"2-word: arrow conseil", "?search=arrow+conseil", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},
		{"2-word: biogaran 1", "?search=biogaran+1", 1, "AMOXICILLINE BIOGARAN 1 g", false},

		// 3-word searches
		{"3-word: arrow 400 caps", "?search=arrow+400+caps", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},
		{"3-word: conseil ibuprofene 400", "?search=conseil+ibuprofene+400", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},
		{"3-word: paracetamol 500 mg", "?search=paracetamol+500+mg", 1, "PARACETAMOL 500 mg", false},
		{"3-word: ibuprofene arrow 400", "?search=ibuprofene+arrow+400", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},

		// 4-5 word searches (realistic medication queries)
		{"4-word: ibuprofene 200 mg enrobé", "?search=ibuprofene+200+mg+comprime", 1, "IBUPROFENE 200 mg, comprimé enrobé", false},
		{"4-word: arrow conseil 400 mg caps", "?search=arrow+conseil+400+mg+caps", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},
		{"5-word: arrow conseil 400 mg caps", "?search=ibuprofene+arrow+conseil+400+mg", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},

		// 6-word searches (maximum allowed)
		{"6-word: arrow conseil 400 mg caps", "?search=ibuprofene+arrow+conseil+400+mg+caps", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},
		{"6-word: arrow conseil 400 mg caps", "?search=ibuprofene+arrow+conseil+400+mg+caps", 1, "IBUPROFENE ARROW CONSEIL 400 mg, caps", false},

		// Single-word backward compatibility
		{"1-word: paracetamol", "?search=paracetamol", 1, "PARACETAMOL 500 mg", false},
		{"1-word: ibuprofene", "?search=ibuprofene", 2, "", false},
		{"1-word: biogaran", "?search=biogaran", 1, "AMOXICILLINE BIOGARAN 1 g", false},

		// No match cases
		{"no match: xyz 123", "?search=xyz+123", 0, "", true},
		{"no match: abc def ghi", "?search=abc+def+ghi", 0, "", true},
		{"no match: aspirin 100", "?search=aspirin+100", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockDataStoreBuilder().WithMedicaments(medicaments).Build()
			mockValidator := NewMockDataValidatorBuilder().Build()
			handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

			req := httptest.NewRequest("GET", "/v1/medicaments"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			handler.ServeMedicamentsV1(w, req)

			if tt.expectError {
				if w.Code != http.StatusNotFound {
					t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
				}

				var errorResponse map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}

				message, ok := errorResponse["message"].(string)
				if !ok || !containsSubstring(message, "No medicaments found") {
					t.Errorf("Expected error message containing 'No medicaments found', got '%s'", message)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				}

				var response []entities.Medicament
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d results, got %d", tt.expectedCount, len(response))
				}

				if tt.expectedMatch != "" && len(response) > 0 {
					found := false
					for _, med := range response {
						if med.Denomination == tt.expectedMatch {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find '%s' in results", tt.expectedMatch)
					}
				}
			}
		})
	}
}

// TestServeMedicamentsV1_MultiWordWordCountLimit tests the 6-word limit
func TestServeMedicamentsV1_MultiWordWordCountLimit(t *testing.T) {
	medicaments := []entities.Medicament{
		{
			Cis:                    10000001,
			Denomination:           "PARACETAMOL 500 mg",
			DenominationNormalized: strings.ToLower("paracetamol 500 mg"),
			FormePharmaceutique:    "Comprimé",
			VoiesAdministration:    []string{"Orale"},
			StatusAutorisation:     "Autorisation active",
			TypeProcedure:          "Procédure nationale",
			EtatComercialisation:   "Commercialisée",
			DateAMM:                "2020-01-01",
			Titulaire:              "SANOFI",
			Presentation:           []entities.Presentation{},
		},
		{
			Cis:                    10000002,
			Denomination:           "IBUPROFENE ARROW CONSEIL 400 mg, caps",
			DenominationNormalized: strings.ToLower("ibuprofene arrow conseil 400 mg, caps"),
			FormePharmaceutique:    "Gélule",
			VoiesAdministration:    []string{"Orale"},
			StatusAutorisation:     "Autorisation active",
			TypeProcedure:          "Procédure nationale",
			EtatComercialisation:   "Commercialisée",
			DateAMM:                "2020-02-01",
			Titulaire:              "ARROW",
			Presentation:           []entities.Presentation{},
		},
	}

	tests := []struct {
		name        string
		queryParams string
		expectError string
	}{
		{"7-word query (should fail)", "?search=a+b+c+d+e+f+g", "maximum 6 words allowed"},
		{"8-word query (should fail)", "?search=a+b+c+d+e+f+g+h", "maximum 6 words allowed"},
		{"9-word query (should fail)", "?search=a+b+c+d+e+f+g+h+i", "maximum 6 words allowed"},
		{"10-word query (should fail)", "?search=a+b+c+d+e+f+g+h+i+j", "maximum 6 words allowed"},
		{"6-word query (should pass)", "?search=ibuprofene+arrow+conseil+400+mg+caps", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockDataStoreBuilder().WithMedicaments(medicaments).Build()
			realValidator := validation.NewDataValidator()
			handler := NewHTTPHandler(mockStore, realValidator, NewMockHealthCheckerBuilder().Build())

			req := httptest.NewRequest("GET", "/v1/medicaments"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			handler.ServeMedicamentsV1(w, req)

			if tt.expectError != "" {
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
				}

				if !strings.Contains(w.Body.String(), tt.expectError) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectError, w.Body.String())
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				}
			}
		})
	}
}

// TestServeGeneriquesV1_MultiWordSearch tests multi-word search functionality for generiques
func TestServeGeneriquesV1_MultiWordSearch(t *testing.T) {
	// Create test generiques with realistic multi-word libelles
	gen1 := entities.GeneriqueList{
		GroupID:           1,
		Libelle:           "PARACETAMOL 500 mg, comprimé",
		LibelleNormalized: "paracetamol 500 mg, comprime",
		Medicaments:       []entities.GeneriqueMedicament{},
	}

	gen2 := entities.GeneriqueList{
		GroupID:           2,
		Libelle:           "IBUPROFENE ARROW 400 mg",
		LibelleNormalized: "ibuprofene arrow 400 mg",
		Medicaments:       []entities.GeneriqueMedicament{},
	}

	gen3 := entities.GeneriqueList{
		GroupID:           3,
		Libelle:           "AMOXICILLINE 1 g, comprimé dispersible",
		LibelleNormalized: "amoxicilline 1 g, comprime dispersible",
		Medicaments:       []entities.GeneriqueMedicament{},
	}

	generiques := []entities.GeneriqueList{gen1, gen2, gen3}

	tests := []struct {
		name          string
		queryParams   string
		expectedCount int
		expectedMatch string
		expectError   bool
	}{
		// 2-word searches
		{"2-word: paracetamol 500", "?libelle=paracetamol+500", 1, "PARACETAMOL 500 mg, comprimé", false},
		{"2-word: ibuprofene arrow", "?libelle=ibuprofene+arrow", 1, "IBUPROFENE ARROW 400 mg", false},
		{"2-word: ibuprofene 400", "?libelle=ibuprofene+400", 1, "IBUPROFENE ARROW 400 mg", false},
		{"2-word: amoxicilline 1", "?libelle=amoxicilline+1", 1, "AMOXICILLINE 1 g, comprimé dispersible", false},

		// 3-word searches
		{"3-word: paracetamol 500 comprimé", "?libelle=paracetamol+500+comprime", 1, "PARACETAMOL 500 mg, comprimé", false},
		{"3-word: arrow 400 mg", "?libelle=arrow+400+mg", 1, "IBUPROFENE ARROW 400 mg", false},
		{"3-word: amoxicilline g dispersible", "?libelle=amoxicilline+g+dispersible", 1, "AMOXICILLINE 1 g, comprimé dispersible", false},

		// 4-5 word searches (realistic generique queries)
		{"4-word: paracetamol 500 mg", "?libelle=paracetamol+500+mg+comprime", 1, "PARACETAMOL 500 mg, comprimé", false},

		// 6-word searches (maximum allowed)
		{"6-word: amoxicilline biogaran 1 g", "?libelle=amoxicilline+1+g+comprime+dispersible", 1, "AMOXICILLINE 1 g, comprimé dispersible", false},

		// Single-word backward compatibility
		{"1-word: paracetamol", "?libelle=paracetamol", 1, "PARACETAMOL 500 mg, comprimé", false},
		{"1-word: ibuprofene", "?libelle=ibuprofene", 1, "IBUPROFENE ARROW 400 mg", false},
		{"1-word: amoxicilline", "?libelle=amoxicilline", 1, "AMOXICILLINE 1 g, comprimé dispersible", false},

		// No match cases
		{"no match: xyz 123", "?libelle=xyz+123", 0, "", true},
		{"no match: abc def ghi", "?libelle=abc+def+ghi", 0, "", true},
		{"no match: aspirin 100", "?libelle=aspirin+100", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockDataStoreBuilder().WithGeneriques(generiques).Build()
			mockValidator := NewMockDataValidatorBuilder().Build()
			handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

			req := httptest.NewRequest("GET", "/v1/generiques"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			handler.ServeGeneriquesV1(w, req)

			if tt.expectError {
				if w.Code != http.StatusNotFound {
					t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
				}

				var errorResponse map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}

				message, ok := errorResponse["message"].(string)
				if !ok || !containsSubstring(message, "No generiques found") {
					t.Errorf("Expected error message containing 'No generiques found', got '%s'", message)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				}

				var response []entities.GeneriqueList
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d results, got %d", tt.expectedCount, len(response))
				}

				if tt.expectedMatch != "" && len(response) > 0 {
					found := false
					for _, gen := range response {
						if gen.Libelle == tt.expectedMatch {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find '%s' in results", tt.expectedMatch)
					}
				}
			}
		})
	}
}

// TestServeGeneriquesV1_MultiWordWordCountLimit tests the 6-word limit for generiques
func TestServeGeneriquesV1_MultiWordWordCountLimit(t *testing.T) {
	generiques := []entities.GeneriqueList{
		{
			GroupID:           1,
			Libelle:           "PARACETAMOL 500 mg",
			LibelleNormalized: strings.ToLower("paracetamol 500 mg"),
			Medicaments:       []entities.GeneriqueMedicament{},
		},
		{
			GroupID:           2,
			Libelle:           "IBUPROFENE ARROW CONSEIL 400 mg",
			LibelleNormalized: strings.ToLower("ibuprofene arrow conseil 400 mg"),
			Medicaments:       []entities.GeneriqueMedicament{},
		},
	}

	tests := []struct {
		name        string
		queryParams string
		expectError string
	}{
		{"7 words rejected (should fail)", "?libelle=a+b+c+d+e+f+g", "maximum 6 words allowed"},
		{"8 words rejected (should fail)", "?libelle=a+b+c+d+e+f+g+h", "maximum 6 words allowed"},
		{"9 words rejected (should fail)", "?libelle=a+b+c+d+e+f+g+h+i", "maximum 6 words allowed"},
		{"10 words rejected (should fail)", "?libelle=a+b+c+d+e+f+g+h+i+j", "maximum 6 words allowed"},
		{"6 words accepted (should pass)", "?libelle=ibuprofene+arrow+conseil+400+mg", ""},
		{"2 words accepted (should pass)", "?libelle=paracetamol+500", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockDataStoreBuilder().WithGeneriques(generiques).Build()
			realValidator := validation.NewDataValidator()
			handler := NewHTTPHandler(mockStore, realValidator, NewMockHealthCheckerBuilder().Build())

			req := httptest.NewRequest("GET", "/v1/generiques"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			handler.ServeGeneriquesV1(w, req)

			if tt.expectError != "" {
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
				}

				if !strings.Contains(w.Body.String(), tt.expectError) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectError, w.Body.String())
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				}
			}
		})
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// hasQuotedETag checks if an ETag is properly quoted
func hasQuotedETag(etag string) bool {
	return len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"'
}

// ============================================================================
// DIAGNOSTICS V1 TESTS
// ============================================================================

// TestServeDiagnosticsV1_Success tests that diagnostics endpoint returns successful response
func TestServeDiagnosticsV1_Success(t *testing.T) {
	handler := NewHTTPHandler(
		NewMockDataStoreBuilder().Build(),
		NewMockDataValidatorBuilder().Build(),
		NewMockHealthCheckerBuilder().Build(),
	)

	req := httptest.NewRequest("GET", "/v1/diagnostics", nil)
	rr := httptest.NewRecorder()

	handler.ServeDiagnosticsV1(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Verify cache headers
	if rr.Header().Get("Cache-Control") != "public, max-age=10" {
		t.Errorf("Expected Cache-Control 'public, max-age=10', got '%s'", rr.Header().Get("Cache-Control"))
	}
}

// TestServeDiagnosticsV1_ResponseStructure tests that diagnostics response has correct structure
func TestServeDiagnosticsV1_ResponseStructure(t *testing.T) {
	handler := NewHTTPHandler(
		NewMockDataStoreBuilder().Build(),
		NewMockDataValidatorBuilder().Build(),
		NewMockHealthCheckerBuilder().Build(),
	)

	req := httptest.NewRequest("GET", "/v1/diagnostics", nil)
	rr := httptest.NewRecorder()

	handler.ServeDiagnosticsV1(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v", err)
	}

	// Check required top-level fields
	requiredFields := []string{
		"timestamp", "uptime_seconds", "next_update",
		"data_age_hours", "system", "data_integrity",
	}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Response missing required field: %s", field)
		}
	}

	// Check timestamp is valid RFC3339
	if timestamp, ok := response["timestamp"].(string); ok {
		if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
			t.Errorf("Invalid timestamp format: %v", err)
		}
	} else {
		t.Error("timestamp should be a string")
	}

	// Check uptime_seconds is a non-negative float
	if uptime, ok := response["uptime_seconds"].(float64); ok {
		if uptime < 0 {
			t.Errorf("uptime_seconds should be non-negative, got %f", uptime)
		}
	} else {
		t.Error("uptime_seconds should be a float")
	}

	// Check next_update is valid RFC3339
	if nextUpdate, ok := response["next_update"].(string); ok {
		if _, err := time.Parse(time.RFC3339, nextUpdate); err != nil {
			t.Errorf("Invalid next_update format: %v", err)
		}
	} else {
		t.Error("next_update should be a string")
	}

	// Check data_age_hours is a non-negative float
	if dataAge, ok := response["data_age_hours"].(float64); ok {
		if dataAge < 0 {
			t.Errorf("data_age_hours should be non-negative, got %f", dataAge)
		}
	} else {
		t.Error("data_age_hours should be a float")
	}

	// Check system field structure
	system, ok := response["system"].(map[string]any)
	if !ok {
		t.Fatal("system should be a map")
	}

	if goroutines, ok := system["goroutines"].(float64); ok {
		if goroutines <= 0 {
			t.Errorf("goroutines should be positive, got %f", goroutines)
		}
	} else {
		t.Error("system.goroutines should be a number")
	}

	memory, ok := system["memory"].(map[string]any)
	if !ok {
		t.Fatal("system.memory should be a map")
	}

	if allocMB, ok := memory["alloc_mb"].(float64); ok {
		if allocMB < 0 {
			t.Errorf("memory.alloc_mb should be non-negative, got %f", allocMB)
		}
	} else {
		t.Error("memory.alloc_mb should be a number")
	}

	if sysMB, ok := memory["sys_mb"].(float64); ok {
		if sysMB < 0 {
			t.Errorf("memory.sys_mb should be non-negative, got %f", sysMB)
		}
	} else {
		t.Error("memory.sys_mb should be a number")
	}

	if numGC, ok := memory["num_gc"].(float64); ok {
		if numGC < 0 {
			t.Errorf("memory.num_gc should be non-negative, got %f", numGC)
		}
	} else {
		t.Error("memory.num_gc should be a number")
	}
}

// TestServeDiagnosticsV1_DataIntegrityCategories tests all data integrity categories are present
func TestServeDiagnosticsV1_DataIntegrityCategories(t *testing.T) {
	handler := NewHTTPHandler(
		NewMockDataStoreBuilder().Build(),
		NewMockDataValidatorBuilder().Build(),
		NewMockHealthCheckerBuilder().Build(),
	)

	req := httptest.NewRequest("GET", "/v1/diagnostics", nil)
	rr := httptest.NewRecorder()

	handler.ServeDiagnosticsV1(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v", err)
	}

	dataIntegrity, ok := response["data_integrity"].(map[string]any)
	if !ok {
		t.Fatal("data_integrity should be a map")
	}

	// Check all 6 required categories
	requiredCategories := []string{
		"medicaments_without_conditions",
		"medicaments_without_generiques",
		"medicaments_without_presentations",
		"medicaments_without_compositions",
		"generique_only_cis",
		"presentations_with_orphaned_cis",
	}
	for _, category := range requiredCategories {
		if _, ok := dataIntegrity[category]; !ok {
			t.Errorf("data_integrity missing required category: %s", category)
		}
	}

	// Check each category has count and sample_cis/sample_cip
	countCategories := map[string]string{
		"medicaments_without_conditions":    "sample_cis",
		"medicaments_without_generiques":    "sample_cis",
		"medicaments_without_presentations": "sample_cis",
		"medicaments_without_compositions":  "sample_cis",
		"generique_only_cis":                "sample_cis",
		"presentations_with_orphaned_cis":   "sample_cip",
	}
	for category, sampleField := range countCategories {
		cat, ok := dataIntegrity[category].(map[string]any)
		if !ok {
			t.Errorf("Category %s should be a map", category)
			continue
		}
		if _, ok := cat["count"]; !ok {
			t.Errorf("Category %s missing 'count' field", category)
		}
		if _, ok := cat[sampleField]; !ok {
			t.Errorf("Category %s missing '%s' field", category, sampleField)
		}
	}
}

// TestServeDiagnosticsV1_DataIntegrityValues tests data integrity values match report
func TestServeDiagnosticsV1_DataIntegrityValues(t *testing.T) {
	report := &interfaces.DataQualityReport{
		DuplicateCIS:                        []int{},
		DuplicateGroupIDs:                   []int{},
		MedicamentsWithoutConditions:        5,
		MedicamentsWithoutGeneriques:        10,
		MedicamentsWithoutPresentations:     3,
		MedicamentsWithoutCompositions:      2,
		GeneriqueOnlyCIS:                    4,
		PresentationsWithOrphanedCIS:        1,
		MedicamentsWithoutConditionsCIS:     []int{1, 2, 3, 4, 5},
		MedicamentsWithoutGeneriquesCIS:     []int{6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		MedicamentsWithoutPresentationsCIS:  []int{16, 17, 18},
		MedicamentsWithoutCompositionsCIS:   []int{19, 20},
		GeneriqueOnlyCISList:                []int{21, 22, 23, 24},
		PresentationsWithOrphanedCISCIPList: []int{100, 200, 300},
	}

	handler := NewHTTPHandler(
		NewMockDataStoreBuilder().WithDataQualityReport(report).Build(),
		NewMockDataValidatorBuilder().Build(),
		NewMockHealthCheckerBuilder().Build(),
	)

	req := httptest.NewRequest("GET", "/v1/diagnostics", nil)
	rr := httptest.NewRecorder()

	handler.ServeDiagnosticsV1(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v", err)
	}

	dataIntegrity := response["data_integrity"].(map[string]any)

	// Verify counts match report
	counts := map[string]int{
		"medicaments_without_conditions":    5,
		"medicaments_without_generiques":    10,
		"medicaments_without_presentations": 3,
		"medicaments_without_compositions":  2,
		"generique_only_cis":                4,
		"presentations_with_orphaned_cis":   1,
	}
	for category, expectedCount := range counts {
		cat := dataIntegrity[category].(map[string]any)
		if count, ok := cat["count"].(float64); ok {
			if int(count) != expectedCount {
				t.Errorf("Category %s: expected count %d, got %f", category, expectedCount, count)
			}
		} else {
			t.Errorf("Category %s: count should be a number", category)
		}
	}

	// Verify sample CIS lists
	sampleCISCategories := map[string][]int{
		"medicaments_without_conditions":    {1, 2, 3, 4, 5},
		"medicaments_without_generiques":    {6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		"medicaments_without_presentations": {16, 17, 18},
		"medicaments_without_compositions":  {19, 20},
		"generique_only_cis":                {21, 22, 23, 24},
	}
	for category, expectedCIS := range sampleCISCategories {
		cat := dataIntegrity[category].(map[string]any)
		sampleCIS, ok := cat["sample_cis"].([]any)
		if !ok {
			t.Errorf("Category %s: sample_cis should be an array", category)
			continue
		}
		if len(sampleCIS) != len(expectedCIS) {
			t.Errorf("Category %s: expected %d sample CIS, got %d", category, len(expectedCIS), len(sampleCIS))
		}
	}

	// Verify sample CIP list for orphaned presentations
	orphanedCat := dataIntegrity["presentations_with_orphaned_cis"].(map[string]any)
	sampleCIP, ok := orphanedCat["sample_cip"].([]any)
	if !ok {
		t.Error("presentations_with_orphaned_cis: sample_cip should be an array")
	} else if len(sampleCIP) != 3 {
		t.Errorf("presentations_with_orphaned_cis: expected 3 sample CIP, got %d", len(sampleCIP))
	}
}

// TestServeDiagnosticsV1_EdgeCases tests edge cases and boundary conditions
func TestServeDiagnosticsV1_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		serverStartTime  time.Time
		lastUpdated      time.Time
		verifyAssertions func(*testing.T, map[string]any)
	}{
		{
			name:            "zero server start time",
			serverStartTime: time.Time{},
			lastUpdated:     time.Now(),
			verifyAssertions: func(t *testing.T, response map[string]any) {
				uptime := response["uptime_seconds"].(float64)
				if uptime != 0 {
					t.Errorf("Expected uptime 0 for zero start time, got %f", uptime)
				}
			},
		},
		{
			name:            "recent data",
			serverStartTime: time.Now().Add(-1 * time.Hour),
			lastUpdated:     time.Now().Add(-30 * time.Minute),
			verifyAssertions: func(t *testing.T, response map[string]any) {
				dataAge := response["data_age_hours"].(float64)
				if dataAge < 0 || dataAge > 1 {
					t.Errorf("Recent data: expected data_age_hours < 1, got %f", dataAge)
				}
			},
		},
		{
			name:            "old data",
			serverStartTime: time.Now().Add(-24 * time.Hour),
			lastUpdated:     time.Now().Add(-3 * time.Hour),
			verifyAssertions: func(t *testing.T, response map[string]any) {
				dataAge := response["data_age_hours"].(float64)
				if dataAge < 2 || dataAge > 4 {
					t.Errorf("Old data: expected data_age_hours between 2 and 4, got %f", dataAge)
				}
			},
		},
		{
			name:            "empty data quality report",
			serverStartTime: time.Now().Add(-1 * time.Hour),
			lastUpdated:     time.Now(),
			verifyAssertions: func(t *testing.T, response map[string]any) {
				dataIntegrity := response["data_integrity"].(map[string]any)
				categories := []string{
					"medicaments_without_conditions",
					"medicaments_without_generiques",
					"medicaments_without_presentations",
					"medicaments_without_compositions",
					"generique_only_cis",
					"presentations_with_orphaned_cis",
				}
				for _, category := range categories {
					cat := dataIntegrity[category].(map[string]any)
					count := cat["count"].(float64)
					if count != 0 {
						t.Errorf("Category %s: expected count 0, got %f", category, count)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var report *interfaces.DataQualityReport
			if tt.name == "empty data quality report" {
				report = &interfaces.DataQualityReport{
					DuplicateCIS:                        []int{},
					DuplicateGroupIDs:                   []int{},
					MedicamentsWithoutConditions:        0,
					MedicamentsWithoutGeneriques:        0,
					MedicamentsWithoutPresentations:     0,
					MedicamentsWithoutCompositions:      0,
					GeneriqueOnlyCIS:                    0,
					PresentationsWithOrphanedCIS:        0,
					MedicamentsWithoutConditionsCIS:     []int{},
					MedicamentsWithoutGeneriquesCIS:     []int{},
					MedicamentsWithoutPresentationsCIS:  []int{},
					MedicamentsWithoutCompositionsCIS:   []int{},
					GeneriqueOnlyCISList:                []int{},
					PresentationsWithOrphanedCISCIPList: []int{},
				}
			}

			builder := NewMockDataStoreBuilder().
				WithServerStartTime(tt.serverStartTime).
				WithLastUpdated(tt.lastUpdated)

			if report != nil {
				builder = builder.WithDataQualityReport(report)
			}

			handler := NewHTTPHandler(
				builder.Build(),
				NewMockDataValidatorBuilder().Build(),
				NewMockHealthCheckerBuilder().Build(),
			)

			req := httptest.NewRequest("GET", "/v1/diagnostics", nil)
			rr := httptest.NewRecorder()

			handler.ServeDiagnosticsV1(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", rr.Code)
			}

			var response map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal JSON response: %v", err)
			}

			tt.verifyAssertions(t, response)
		})
	}
}

// TestServeDiagnosticsV1_Comprehensive provides comprehensive table-driven tests
func TestServeDiagnosticsV1_Comprehensive(t *testing.T) {
	now := time.Now()

	report := &interfaces.DataQualityReport{
		DuplicateCIS:                        []int{},
		DuplicateGroupIDs:                   []int{},
		MedicamentsWithoutConditions:        2,
		MedicamentsWithoutGeneriques:        5,
		MedicamentsWithoutPresentations:     3,
		MedicamentsWithoutCompositions:      1,
		GeneriqueOnlyCIS:                    4,
		PresentationsWithOrphanedCIS:        2,
		MedicamentsWithoutConditionsCIS:     []int{100, 200},
		MedicamentsWithoutGeneriquesCIS:     []int{300, 400, 500, 600, 700},
		MedicamentsWithoutPresentationsCIS:  []int{800, 900, 1000},
		MedicamentsWithoutCompositionsCIS:   []int{1100},
		GeneriqueOnlyCISList:                []int{1200, 1300, 1400, 1500},
		PresentationsWithOrphanedCISCIPList: []int{1234567, 7654321, 9999999},
	}

	tests := []struct {
		name           string
		serverStart    time.Time
		lastUpdated    time.Time
		nextUpdate     time.Time
		expectedStatus int
		validate       func(*testing.T, map[string]any)
	}{
		{
			name:           "healthy system with recent data",
			serverStart:    now.Add(-1 * time.Hour),
			lastUpdated:    now.Add(-30 * time.Minute),
			nextUpdate:     now.Add(2 * time.Hour),
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]any) {
				dataAge := response["data_age_hours"].(float64)
				if dataAge < 0 || dataAge > 1 {
					t.Errorf("Expected data_age_hours < 1, got %f", dataAge)
				}

				uptime := response["uptime_seconds"].(float64)
				if uptime < 3000 || uptime > 4000 { // ~1 hour
					t.Errorf("Expected uptime ~3600s, got %f", uptime)
				}
			},
		},
		{
			name:           "system with data quality issues",
			serverStart:    now.Add(-24 * time.Hour),
			lastUpdated:    now.Add(-2 * time.Hour),
			nextUpdate:     now.Add(4 * time.Hour),
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]any) {
				dataIntegrity := response["data_integrity"].(map[string]any)

				expectedCounts := map[string]int{
					"medicaments_without_conditions":    2,
					"medicaments_without_generiques":    5,
					"medicaments_without_presentations": 3,
					"medicaments_without_compositions":  1,
					"generique_only_cis":                4,
					"presentations_with_orphaned_cis":   2,
				}

				for category, expectedCount := range expectedCounts {
					cat := dataIntegrity[category].(map[string]any)
					count := cat["count"].(float64)
					if int(count) != expectedCount {
						t.Errorf("Category %s: expected count %d, got %f", category, expectedCount, count)
					}
				}

				// Check that sample lists are not empty when count > 0
				withoutConditions := dataIntegrity["medicaments_without_conditions"].(map[string]any)
				sampleCIS := withoutConditions["sample_cis"].([]any)
				if len(sampleCIS) != 2 {
					t.Errorf("Expected 2 sample CIS, got %d", len(sampleCIS))
				}
			},
		},
		{
			name:           "zero server start time",
			serverStart:    time.Time{},
			lastUpdated:    now,
			nextUpdate:     now.Add(1 * time.Hour),
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]any) {
				uptime := response["uptime_seconds"].(float64)
				if uptime != 0 {
					t.Errorf("Expected uptime 0 for zero start time, got %f", uptime)
				}
			},
		},
		{
			name:           "long running server",
			serverStart:    now.Add(-7 * 24 * time.Hour),
			lastUpdated:    now.Add(-30 * time.Minute),
			nextUpdate:     now.Add(2 * time.Hour),
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]any) {
				uptime := response["uptime_seconds"].(float64)
				// ~7 days in seconds
				if uptime < 600000 || uptime > 610000 {
					t.Errorf("Expected uptime ~604800s (7 days), got %f", uptime)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(
				NewMockDataStoreBuilder().
					WithServerStartTime(tt.serverStart).
					WithLastUpdated(tt.lastUpdated).
					WithDataQualityReport(report).
					Build(),
				NewMockDataValidatorBuilder().Build(),
				NewMockHealthCheckerBuilder().
					WithNextUpdate(tt.nextUpdate).
					Build(),
			)

			req := httptest.NewRequest("GET", "/v1/diagnostics", nil)
			rr := httptest.NewRecorder()

			handler.ServeDiagnosticsV1(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			var response map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal JSON response: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, response)
			}
		})
	}
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
