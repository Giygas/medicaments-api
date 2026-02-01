package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

// ============================================================================
// CORE HANDLER TESTS
// ============================================================================

// TestNewHTTPHandler tests handler creation
func TestNewHTTPHandler(t *testing.T) {
	tests := []struct {
		name      string
		dataStore interfaces.DataStore
		validator interfaces.DataValidator
	}{
		{
			name:      "valid dependencies",
			dataStore: NewMockDataStoreBuilder().Build(),
			validator: NewMockDataValidatorBuilder().Build(),
		},
		{
			name:      "nil data store",
			dataStore: nil,
			validator: NewMockDataValidatorBuilder().Build(),
		},
		{
			name:      "nil validator",
			dataStore: NewMockDataStoreBuilder().Build(),
			validator: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(tt.dataStore, tt.validator)

			if handler == nil {
				t.Fatal("Handler should not be nil")
			}

			// Verify it implements interface
			var _ = handler
		})
	}
}

// TestRespondWithJSON tests JSON response formatting
func TestRespondWithJSON(t *testing.T) {
	mockStore := NewMockDataStoreBuilder().Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator).(*HTTPHandlerImpl)

	tests := []struct {
		name           string
		code           int
		payload        any
		expectedStatus int
		expectedJSON   string
	}{
		{
			name:           "successful response",
			code:           http.StatusOK,
			payload:        map[string]string{"message": "success"},
			expectedStatus: http.StatusOK,
			expectedJSON:   `{"message":"success"}`,
		},
		{
			name:           "empty payload",
			code:           http.StatusOK,
			payload:        nil,
			expectedStatus: http.StatusOK,
			expectedJSON:   `null`,
		},
		{
			name:           "array payload",
			code:           http.StatusOK,
			payload:        []string{"item1", "item2"},
			expectedStatus: http.StatusOK,
			expectedJSON:   `["item1","item2"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			handler.RespondWithJSON(rr, tt.code, tt.payload)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Expected Content-Type application/json; charset=utf-8, got %s", ct)
			}

			if !strings.Contains(rr.Body.String(), tt.expectedJSON) {
				t.Errorf("Expected body to contain %s, got %s", tt.expectedJSON, rr.Body.String())
			}
		})
	}
}

// TestRespondWithError tests error response formatting
func TestRespondWithError(t *testing.T) {
	mockStore := NewMockDataStoreBuilder().Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator).(*HTTPHandlerImpl)

	tests := []struct {
		name           string
		code           int
		message        string
		expectedStatus int
		expectedJSON   string
	}{
		{
			name:           "bad request error",
			code:           http.StatusBadRequest,
			message:        "Invalid input",
			expectedStatus: http.StatusBadRequest,
			expectedJSON:   `"message":"Invalid input"`,
		},
		{
			name:           "not found error",
			code:           http.StatusNotFound,
			message:        "Resource not found",
			expectedStatus: http.StatusNotFound,
			expectedJSON:   `"message":"Resource not found"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			handler.RespondWithError(rr, tt.code, tt.message)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Expected Content-Type application/json; charset=utf-8, got %s", ct)
			}

			if !strings.Contains(rr.Body.String(), tt.expectedJSON) {
				t.Errorf("Expected body to contain %s, got %s", tt.expectedJSON, rr.Body.String())
			}
		})
	}
}

// TestExportMedicaments tests medicaments export endpoint
func TestExportMedicaments(t *testing.T) {
	factory := NewTestDataFactory()

	tests := []struct {
		name         string
		medicaments  []entities.Medicament
		expectedCode int
		expectArray  bool
	}{
		{
			name: "with medicaments",
			medicaments: []entities.Medicament{
				factory.CreateMedicament(1, "Test Med 1"),
				factory.CreateMedicament(2, "Test Med 2"),
			},
			expectedCode: http.StatusOK,
			expectArray:  true,
		},
		{
			name:         "empty medicaments",
			medicaments:  []entities.Medicament{},
			expectedCode: http.StatusOK,
			expectArray:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &MockDataStore{medicaments: tt.medicaments}
			mockValidator := &MockDataValidator{}
			handler := NewHTTPHandler(mockStore, mockValidator)

			req := httptest.NewRequest("GET", "/database", nil)
			rr := httptest.NewRecorder()

			handler.ExportMedicaments(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Expected Content-Type application/json; charset=utf-8, got %s", ct)
			}

			var response []entities.Medicament
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to unmarshal JSON: %v", err)
			}

			if len(response) != len(tt.medicaments) {
				t.Errorf("Expected %d medicaments, got %d", len(tt.medicaments), len(response))
			}

			// Verify ETag headers are present
			etag := rr.Header().Get("ETag")
			if etag == "" {
				t.Error("ETag header should be present")
			}

			if !strings.HasPrefix(etag, "\"") || !strings.HasSuffix(etag, "\"") {
				t.Errorf("ETag should be quoted, got: %s", etag)
			}

			if rr.Header().Get("Cache-Control") != "public, max-age=3600" {
				t.Error("Expected Cache-Control 'public, max-age=3600'")
			}

			if rr.Header().Get("Last-Modified") == "" {
				t.Error("Expected Last-Modified header")
			}

			if !mockStore.getMedicamentsCalled {
				t.Error("GetMedicaments should have been called")
			}
		})
	}
}

// TestServePagedMedicaments tests pagination
func TestServePagedMedicaments(t *testing.T) {
	factory := NewTestDataFactory()

	tests := []struct {
		name         string
		pageNumber   string
		medicaments  []entities.Medicament
		expectedCode int
		expectError  string
	}{
		{
			name:         "valid page 1",
			pageNumber:   "1",
			medicaments:  []entities.Medicament{factory.CreateMedicament(1, "Test Med 1")},
			expectedCode: http.StatusOK,
		},
		{
			name:         "valid page 2",
			pageNumber:   "2",
			medicaments:  []entities.Medicament{factory.CreateMedicament(1, "Test Med 1")},
			expectedCode: http.StatusNotFound,
			expectError:  "Page not found",
		},
		{
			name:         "invalid page number",
			pageNumber:   "invalid",
			medicaments:  []entities.Medicament{factory.CreateMedicament(1, "Test Med 1")},
			expectedCode: http.StatusBadRequest,
			expectError:  "Invalid page number",
		},
		{
			name:         "negative page number",
			pageNumber:   "-1",
			medicaments:  []entities.Medicament{factory.CreateMedicament(1, "Test Med 1")},
			expectedCode: http.StatusBadRequest,
			expectError:  "Invalid page number",
		},
		{
			name:         "zero page number",
			pageNumber:   "0",
			medicaments:  []entities.Medicament{factory.CreateMedicament(1, "Test Med 1")},
			expectedCode: http.StatusBadRequest,
			expectError:  "Invalid page number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &MockDataStore{medicaments: tt.medicaments}
			mockValidator := &MockDataValidator{}
			handler := NewHTTPHandler(mockStore, mockValidator)

			// Create a request with chi URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("pageNumber", tt.pageNumber)

			req := httptest.NewRequest("GET", "/database/"+tt.pageNumber, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			handler.ServePagedMedicaments(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			if tt.expectError != "" {
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if message, ok := response["message"].(string); !ok || message != tt.expectError {
					t.Errorf("Expected error %s, got %v", tt.expectError, response["message"])
				}
			} else {
				// Verify pagination metadata
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if _, ok := response["data"]; !ok {
					t.Error("Response should contain 'data' field")
				}

				if _, ok := response["page"]; !ok {
					t.Error("Response should contain 'page' field")
				}

				if _, ok := response["pageSize"]; !ok {
					t.Error("Response should contain 'pageSize' field")
				}

				if _, ok := response["totalItems"]; !ok {
					t.Error("Response should contain 'totalItems' field")
				}

				if _, ok := response["maxPage"]; !ok {
					t.Error("Response should contain 'maxPage' field")
				}
			}
		})
	}
}

// TestFindMedicament tests medicament search
func TestFindMedicament(t *testing.T) {
	factory := NewTestDataFactory()

	tests := []struct {
		name         string
		element      string
		medicaments  []entities.Medicament
		expectedCode int
		expectError  string
	}{
		{
			name:    "valid search term",
			element: "Doliprane",
			medicaments: []entities.Medicament{
				factory.CreateMedicament(1, "Doliprane"),
				factory.CreateMedicament(2, "Ibuprofène"),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "empty search term",
			element:      "",
			medicaments:  []entities.Medicament{factory.CreateMedicament(1, "Test Med")},
			expectedCode: http.StatusBadRequest,
			expectError:  "Missing search term",
		},
		{
			name:         "no results",
			element:      "NonExistent",
			medicaments:  []entities.Medicament{factory.CreateMedicament(1, "Doliprane")},
			expectedCode: http.StatusOK,
			expectError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &MockDataStore{medicaments: tt.medicaments}
			mockValidator := &MockDataValidator{}
			handler := NewHTTPHandler(mockStore, mockValidator)

			// Create a request with chi URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("element", tt.element)

			req := httptest.NewRequest("GET", "/medicament/"+tt.element, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			handler.FindMedicament(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			if tt.expectError != "" {
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if message, ok := response["message"].(string); !ok || message != tt.expectError {
					t.Errorf("Expected error %s, got %v", tt.expectError, response["message"])
				}
			} else {
				// For successful responses, expect JSON array
				var response []entities.Medicament
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON array: %v", err)
				}

				// For "no results" case, expect empty array
				if tt.name == "no results" && len(response) != 0 {
					t.Errorf("Expected empty array for no results, got %d items", len(response))
				}
			}
		})
	}
}

// TestFindMedicamentByID tests medicament lookup by CIS
func TestFindMedicamentByID(t *testing.T) {
	factory := NewTestDataFactory()

	tests := []struct {
		name           string
		cis            string
		medicaments    []entities.Medicament
		medicamentsMap map[int]entities.Medicament
		expectedCode   int
		expectError    string
	}{
		{
			name: "valid CIS",
			cis:  "00000001",
			medicaments: []entities.Medicament{
				factory.CreateMedicament(1, "Doliprane"),
			},
			medicamentsMap: map[int]entities.Medicament{
				1: factory.CreateMedicament(1, "Doliprane"),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:           "invalid CIS (non-numeric)",
			cis:            "invalid",
			medicaments:    []entities.Medicament{factory.CreateMedicament(1, "Test Med")},
			medicamentsMap: map[int]entities.Medicament{},
			expectedCode:   http.StatusBadRequest,
			expectError:    "CIS should have 8 digits",
		},
		{
			name:           "non-existent CIS",
			cis:            "99999999",
			medicaments:    []entities.Medicament{factory.CreateMedicament(1, "Test Med")},
			medicamentsMap: map[int]entities.Medicament{},
			expectedCode:   http.StatusNotFound,
			expectError:    "Medicament not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockDataStoreBuilder().
				WithMedicaments(tt.medicaments).
				Build()
			// Manually set the medicaments map for this specific test
			mockStore.medicamentsMap = tt.medicamentsMap
			mockValidator := NewMockDataValidatorBuilder().Build()
			handler := NewHTTPHandler(mockStore, mockValidator)

			// Create a request with chi URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("cis", tt.cis)

			req := httptest.NewRequest("GET", "/medicament/id/"+tt.cis, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			handler.FindMedicamentByID(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			if tt.expectError != "" {
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if message, ok := response["message"].(string); !ok || message != tt.expectError {
					t.Errorf("Expected error %s, got %v", tt.expectError, response["message"])
				}
			}
		})
	}
}

// TestFindGeneriques tests generique search
func TestFindGeneriques(t *testing.T) {
	factory := NewTestDataFactory()

	tests := []struct {
		name         string
		libelle      string
		generiques   []entities.GeneriqueList
		expectedCode int
		expectError  string
	}{
		{
			name:    "valid libelle search",
			libelle: "Paracetamol",
			generiques: []entities.GeneriqueList{
				factory.CreateGeneriqueList(1, "Paracetamol", []int{1}),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "empty libelle",
			libelle:      "",
			generiques:   []entities.GeneriqueList{factory.CreateGeneriqueList(1, "Test", []int{1})},
			expectedCode: http.StatusBadRequest,
			expectError:  "Missing libelle",
		},
		{
			name:         "no results",
			libelle:      "NonExistent",
			generiques:   []entities.GeneriqueList{factory.CreateGeneriqueList(1, "Test", []int{1})},
			expectedCode: http.StatusNotFound,
			expectError:  "No generiques found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockDataStoreBuilder().WithGeneriques(tt.generiques).Build()
			mockValidator := NewMockDataValidatorBuilder().Build()
			handler := NewHTTPHandler(mockStore, mockValidator)

			// Create a request with chi URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("libelle", tt.libelle)

			req := httptest.NewRequest("GET", "/generiques/"+tt.libelle, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			handler.FindGeneriques(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			if tt.expectError != "" {
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if message, ok := response["message"].(string); !ok || message != tt.expectError {
					t.Errorf("Expected error %s, got %v", tt.expectError, response["message"])
				}
			}
		})
	}
}

// TestFindGeneriquesByGroupID tests generique lookup by group ID
func TestFindGeneriquesByGroupID(t *testing.T) {
	factory := NewTestDataFactory()

	tests := []struct {
		name          string
		groupID       string
		generiques    []entities.GeneriqueList
		generiquesMap map[int]entities.GeneriqueList
		expectedCode  int
		expectError   string
	}{
		{
			name:    "valid group ID",
			groupID: "1",
			generiques: []entities.GeneriqueList{
				factory.CreateGeneriqueList(1, "Test Group", []int{1}),
			},
			generiquesMap: map[int]entities.GeneriqueList{
				1: {GroupID: 1, Libelle: "Test Group"},
			},
			expectedCode: http.StatusOK,
		},
		{
			name:          "invalid group ID (non-numeric)",
			groupID:       "invalid",
			generiques:    []entities.GeneriqueList{factory.CreateGeneriqueList(1, "Test", []int{1})},
			generiquesMap: map[int]entities.GeneriqueList{},
			expectedCode:  http.StatusBadRequest,
			expectError:   "Invalid group ID",
		},
		{
			name:          "non-existent group ID",
			groupID:       "999",
			generiques:    []entities.GeneriqueList{factory.CreateGeneriqueList(1, "Test", []int{1})},
			generiquesMap: map[int]entities.GeneriqueList{},
			expectedCode:  http.StatusNotFound,
			expectError:   "Generique group not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockDataStoreBuilder().
				WithGeneriques(tt.generiques).
				Build()
			// Manually set the generiques map for this specific test
			mockStore.generiquesMap = tt.generiquesMap
			mockValidator := NewMockDataValidatorBuilder().Build()
			handler := NewHTTPHandler(mockStore, mockValidator)

			// Create a request with chi URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("groupId", tt.groupID)

			req := httptest.NewRequest("GET", "/generiques/group/"+tt.groupID, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			handler.FindGeneriquesByGroupID(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			if tt.expectError != "" {
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if message, ok := response["message"].(string); !ok || message != tt.expectError {
					t.Errorf("Expected error %s, got %v", tt.expectError, response["message"])
				}
			}
		})
	}
}

// TestFindMedicamentByCIP tests medicament lookup by CIP code (CIP7 or CIP13)
func TestFindMedicamentByCIP(t *testing.T) {
	factory := NewTestDataFactory()

	// Create test data with presentations
	medWithPresentation := factory.CreateMedicament(1, "Doliprane")
	medWithPresentation.Presentation = []entities.Presentation{
		{Cis: 1, Cip7: 1234567, Cip13: 1234567890123, Libelle: "Boîte de 8 comprimés"},
	}

	medWithDifferentPresentation := factory.CreateMedicament(2, "Ibuprofène")
	medWithDifferentPresentation.Presentation = []entities.Presentation{
		{Cis: 2, Cip7: 7654321, Cip13: 7654321098765, Libelle: "Boîte de 20 gélules"},
	}

	tests := []struct {
		name          string
		cip           string
		medicaments   []entities.Medicament
		expectedCode  int
		expectError   string
		checkCipMatch int // The CIP that should match (0 if none)
	}{
		{
			name: "valid CIP7 code found",
			cip:  "1234567",
			medicaments: []entities.Medicament{
				medWithPresentation,
				medWithDifferentPresentation,
			},
			expectedCode:  http.StatusOK,
			checkCipMatch: 1234567,
		},
		{
			name: "valid CIP13 code found",
			cip:  "1234567890123",
			medicaments: []entities.Medicament{
				medWithPresentation,
				medWithDifferentPresentation,
			},
			expectedCode:  http.StatusOK,
			checkCipMatch: 1234567890123,
		},
		{
			name: "empty CIP code",
			cip:  "",
			medicaments: []entities.Medicament{
				medWithPresentation,
			},
			expectedCode: http.StatusBadRequest,
			expectError:  "input cannot be empty",
		},
		{
			name: "non-numeric CIP code",
			cip:  "abcd123",
			medicaments: []entities.Medicament{
				medWithPresentation,
			},
			expectedCode: http.StatusBadRequest,
			expectError:  "input contains invalid characters. Only numeric characters are allowed",
		},
		{
			name: "CIP code not found in any presentation",
			cip:  "9999999",
			medicaments: []entities.Medicament{
				medWithPresentation,
			},
			expectedCode: http.StatusNotFound,
			expectError:  "Medicament not found",
		},
		{
			name:         "medicament with presentation but CIP doesn't match",
			cip:          "9999999",
			medicaments:  []entities.Medicament{factory.CreateMedicament(1, "Different CIP")},
			expectedCode: http.StatusNotFound,
			expectError:  "Medicament not found",
		},
		{
			name: "CIP code found in CIP13 but not CIP7",
			cip:  "7654321098765",
			medicaments: []entities.Medicament{
				medWithPresentation,
				medWithDifferentPresentation,
			},
			expectedCode:  http.StatusOK,
			checkCipMatch: 7654321098765,
		},
		{
			name: "CIP code with leading zeros",
			cip:  "0123456",
			medicaments: []entities.Medicament{
				medWithPresentation,
			},
			expectedCode: http.StatusNotFound,
			expectError:  "Medicament not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build presentation maps for O(1) lookups
			presentationsCIP7Map := make(map[int]entities.Presentation)
			presentationsCIP13Map := make(map[int]entities.Presentation)
			for _, med := range tt.medicaments {
				for _, pres := range med.Presentation {
					presentationsCIP7Map[pres.Cip7] = pres
					presentationsCIP13Map[pres.Cip13] = pres
				}
			}

			mockStore := NewMockDataStoreBuilder().
				WithMedicaments(tt.medicaments).
				WithPresentationsCIP7Map(presentationsCIP7Map).
				WithPresentationsCIP13Map(presentationsCIP13Map).
				Build()
			mockValidator := NewMockDataValidatorBuilder().Build()
			handler := NewHTTPHandler(mockStore, mockValidator)

			// Create a request with chi URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("cip", tt.cip)

			req := httptest.NewRequest("GET", "/medicament/cip/"+tt.cip, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			handler.FindMedicamentByCIP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			if tt.expectError != "" {
				var response map[string]any
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if message, ok := response["message"].(string); !ok || message != tt.expectError {
					t.Errorf("Expected error %s, got %v", tt.expectError, response["message"])
				}
			} else {
				// For successful responses, expect JSON object
				var response entities.Medicament
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON object: %v", err)
				}

				// Verify that the returned medicament contains the expected CIP
				if tt.checkCipMatch != 0 {
					found := false
					for _, pres := range response.Presentation {
						if pres.Cip7 == tt.checkCipMatch || pres.Cip13 == tt.checkCipMatch {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find CIP %d in presentation, got %+v", tt.checkCipMatch, response.Presentation)
					}
				}
			}
		})
	}
}

// TestHealthCheck tests health check endpoint
func TestHealthCheck(t *testing.T) {
	factory := NewTestDataFactory()

	tests := []struct {
		name           string
		medicaments    []entities.Medicament
		generiques     []entities.GeneriqueList
		lastUpdated    time.Time
		updating       bool
		expectedCode   int
		expectedStatus string
	}{
		{
			name:           "healthy system",
			medicaments:    []entities.Medicament{factory.CreateMedicament(1, "Test Med")},
			generiques:     []entities.GeneriqueList{factory.CreateGeneriqueList(1, "Test Group", []int{1})},
			lastUpdated:    time.Now().Add(-1 * time.Hour),
			updating:       false,
			expectedCode:   http.StatusOK,
			expectedStatus: "healthy",
		},
		{
			name:           "system during update",
			medicaments:    []entities.Medicament{factory.CreateMedicament(1, "Test Med")},
			generiques:     []entities.GeneriqueList{factory.CreateGeneriqueList(1, "Test Group", []int{1})},
			lastUpdated:    time.Now().Add(-1 * time.Hour),
			updating:       true,
			expectedCode:   http.StatusOK,
			expectedStatus: "healthy",
		},
		{
			name:           "stale data",
			medicaments:    []entities.Medicament{factory.CreateMedicament(1, "Test Med")},
			generiques:     []entities.GeneriqueList{factory.CreateGeneriqueList(1, "Test Group", []int{1})},
			lastUpdated:    time.Now().Add(-25 * time.Hour),
			updating:       false,
			expectedCode:   http.StatusOK,
			expectedStatus: "degraded",
		},
		{
			name:           "unhealthy system (no data)",
			medicaments:    []entities.Medicament{},
			generiques:     []entities.GeneriqueList{},
			lastUpdated:    time.Time{},
			updating:       false,
			expectedCode:   http.StatusServiceUnavailable,
			expectedStatus: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockDataStoreBuilder().
				WithMedicaments(tt.medicaments).
				WithGeneriques(tt.generiques).
				WithLastUpdated(tt.lastUpdated).
				WithUpdating(tt.updating).
				Build()
			mockValidator := NewMockDataValidatorBuilder().Build()
			handler := NewHTTPHandler(mockStore, mockValidator)

			req := httptest.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()

			handler.HealthCheck(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			// Verify response structure
			var response map[string]any
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to unmarshal JSON: %v", err)
			}

			// Check status
			if status, ok := response["status"].(string); !ok || status != tt.expectedStatus {
				t.Errorf("Status mismatch: expected %s, got %s", tt.expectedStatus, response["status"])
			}

			// Check required fields
			requiredFields := []string{"status", "last_update", "data_age_hours", "uptime_seconds", "data", "system"}
			for _, field := range requiredFields {
				if _, ok := response[field]; !ok {
					t.Errorf("Response should contain '%s' field", field)
				}
			}

			// Verify data field contains expected keys
			if data, ok := response["data"].(map[string]any); ok {
				expectedDataKeys := []string{"api_version", "medicaments", "generiques", "is_updating", "next_update"}
				for _, key := range expectedDataKeys {
					if _, ok := data[key]; !ok {
						t.Errorf("Data should contain '%s' key", key)
					}
				}
			}

			// Verify system field contains expected keys
			if system, ok := response["system"].(map[string]any); ok {
				expectedSystemKeys := []string{"goroutines", "memory"}
				for _, key := range expectedSystemKeys {
					if _, ok := system[key]; !ok {
						t.Errorf("System should contain '%s' key", key)
					}
				}
			}
		})
	}
}

// ============================================================================
// ETag UTILITY FUNCTION TESTS
// ============================================================================

func TestGenerateETag(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte(""),
		},
		{
			name: "simple data",
			data: []byte("hello world"),
		},
		{
			name: "json data",
			data: []byte(`{"test": "data", "number": 123}`),
		},
		{
			name: "binary data",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateETag(tt.data)
			if !strings.HasPrefix(result, `"`) {
				t.Errorf("ETag should be quoted, got %s", result)
			}
			if !strings.HasSuffix(result, `"`) {
				t.Errorf("ETag should be quoted, got %s", result)
			}
			// ETag hash should be 16 hex characters (8 bytes) after trimming quotes
			etagContent := string(result[1 : len(result)-1])
			if len(etagContent) != 16 {
				t.Errorf("ETag hash should be 16 hex characters (8 bytes), got %d", len(etagContent))
			}
		})
	}

	// Test consistency - same data should always generate same ETag
	data := []byte("consistency test")
	etag1 := GenerateETag(data)
	etag2 := GenerateETag(data)
	if etag1 != etag2 {
		t.Errorf("ETag should be consistent for same data, got %s and %s", etag1, etag2)
	}

	// Test uniqueness - different data should generate different ETags
	data2 := []byte("consistency test modified")
	etag3 := GenerateETag(data2)
	if etag1 == etag3 {
		t.Errorf("Different data should generate different ETags, got same %s", etag1)
	}
}

func TestCheckETag(t *testing.T) {
	tests := []struct {
		name          string
		ifNoneMatch   string
		currentETag   string
		expectedMatch bool
	}{
		{
			name:          "no If-None-Match header",
			ifNoneMatch:   "",
			currentETag:   `"test-etag"`,
			expectedMatch: false,
		},
		{
			name:          "matching ETag",
			ifNoneMatch:   `"test-etag"`,
			currentETag:   `"test-etag"`,
			expectedMatch: true,
		},
		{
			name:          "non-matching ETag",
			ifNoneMatch:   `"different-etag"`,
			currentETag:   `"test-etag"`,
			expectedMatch: false,
		},
		{
			name:          "wildcard ETag",
			ifNoneMatch:   `*`,
			currentETag:   `"test-etag"`,
			expectedMatch: false, // Current implementation only does exact match
		},
		{
			name:          "empty ETag header",
			ifNoneMatch:   ``,
			currentETag:   `"test-etag"`,
			expectedMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.ifNoneMatch != "" {
				req.Header.Set("If-None-Match", tt.ifNoneMatch)
			}

			match := CheckETag(req, tt.currentETag)
			if match != tt.expectedMatch {
				t.Errorf("Expected match %v, got %v", tt.expectedMatch, match)
			}
		})
	}
}
