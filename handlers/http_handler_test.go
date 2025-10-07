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

// MockDataStore implements interfaces.DataStore for testing
type MockDataStore struct {
	medicaments    []entities.Medicament
	generiques     []entities.GeneriqueList
	medicamentsMap map[int]entities.Medicament
	generiquesMap  map[int]entities.Generique
	lastUpdated    time.Time
	updating       bool

	// Method call tracking
	getMedicamentsCalled    bool
	getGeneriquesCalled     bool
	getMedicamentsMapCalled bool
	getGeneriquesMapCalled  bool
	beginUpdateCalled       bool
	endUpdateCalled         bool
	updateDataCalled        bool
}

func (m *MockDataStore) GetMedicaments() []entities.Medicament {
	m.getMedicamentsCalled = true
	return m.medicaments
}

func (m *MockDataStore) GetGeneriques() []entities.GeneriqueList {
	m.getGeneriquesCalled = true
	return m.generiques
}

func (m *MockDataStore) GetMedicamentsMap() map[int]entities.Medicament {
	m.getMedicamentsMapCalled = true
	return m.medicamentsMap
}

func (m *MockDataStore) GetGeneriquesMap() map[int]entities.Generique {
	m.getGeneriquesMapCalled = true
	return m.generiquesMap
}

func (m *MockDataStore) GetLastUpdated() time.Time {
	return m.lastUpdated
}

func (m *MockDataStore) IsUpdating() bool {
	return m.updating
}

func (m *MockDataStore) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList,
	medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.Generique) {
	m.updateDataCalled = true
	m.medicaments = medicaments
	m.generiques = generiques
	m.medicamentsMap = medicamentsMap
	m.generiquesMap = generiquesMap
	m.lastUpdated = time.Now()
}

func (m *MockDataStore) BeginUpdate() bool {
	m.beginUpdateCalled = true
	m.updating = true
	return true
}

func (m *MockDataStore) EndUpdate() {
	m.endUpdateCalled = true
	m.updating = false
}

// MockDataValidator implements interfaces.DataValidator for testing
type MockDataValidator struct {
	validateInputError      error
	validateMedicamentError error

	validateInputCalled bool
	lastValidatedInput  string
}

func (m *MockDataValidator) ValidateMedicament(med *entities.Medicament) error {
	return m.validateMedicamentError
}

func (m *MockDataValidator) ValidateDataIntegrity(medicaments []entities.Medicament, generiques []entities.GeneriqueList) error {
	return nil
}

func (m *MockDataValidator) ValidateInput(input string) error {
	m.validateInputCalled = true
	m.lastValidatedInput = input
	return m.validateInputError
}

// Helper function to create test medicaments
func createTestMedicament(cis int, denomination string) entities.Medicament {
	return entities.Medicament{
		Cis:                  cis,
		Denomination:         denomination,
		FormePharmaceutique:  "Comprimé",
		VoiesAdministration:  []string{"Orale"},
		StatusAutorisation:   "Autorisé",
		TypeProcedure:        "Procédure nationale",
		EtatComercialisation: "Commercialisé",
	}
}

// Helper function to create test generique list
func createTestGeneriqueList(groupID int, libelle string) entities.GeneriqueList {
	return entities.GeneriqueList{
		GroupID: groupID,
		Libelle: libelle,
		Medicaments: []entities.GeneriqueMedicament{
			{
				Cis:          1,
				Denomination: "Test Med",
			},
		},
	}
}

// TestNewHTTPHandler tests handler creation
func TestNewHTTPHandler(t *testing.T) {
	tests := []struct {
		name      string
		dataStore interfaces.DataStore
		validator interfaces.DataValidator
	}{
		{
			name:      "valid dependencies",
			dataStore: &MockDataStore{},
			validator: &MockDataValidator{},
		},
		{
			name:      "nil data store",
			dataStore: nil,
			validator: &MockDataValidator{},
		},
		{
			name:      "nil validator",
			dataStore: &MockDataStore{},
			validator: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(tt.dataStore, tt.validator)

			if handler == nil {
				t.Fatal("Handler should not be nil")
			}

			// Verify it implements the interface
			var _ interfaces.HTTPHandler = handler
		})
	}
}

// TestRespondWithJSON tests JSON response formatting
func TestRespondWithJSON(t *testing.T) {
	mockStore := &MockDataStore{}
	mockValidator := &MockDataValidator{}
	handler := NewHTTPHandler(mockStore, mockValidator).(*HTTPHandlerImpl)

	tests := []struct {
		name           string
		code           int
		payload        interface{}
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
	mockStore := &MockDataStore{}
	mockValidator := &MockDataValidator{}
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

// TestServeAllMedicaments tests the medicaments endpoint
func TestServeAllMedicaments(t *testing.T) {
	tests := []struct {
		name         string
		medicaments  []entities.Medicament
		expectedCode int
		expectArray  bool
	}{
		{
			name: "with medicaments",
			medicaments: []entities.Medicament{
				createTestMedicament(1, "Test Med 1"),
				createTestMedicament(2, "Test Med 2"),
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

			handler.ServeAllMedicaments(rr, req)

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

			if !mockStore.getMedicamentsCalled {
				t.Error("GetMedicaments should have been called")
			}
		})
	}
}

// TestServePagedMedicaments tests pagination
func TestServePagedMedicaments(t *testing.T) {
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
			medicaments:  []entities.Medicament{createTestMedicament(1, "Test Med 1")},
			expectedCode: http.StatusOK,
		},
		{
			name:         "valid page 2",
			pageNumber:   "2",
			medicaments:  []entities.Medicament{createTestMedicament(1, "Test Med 1")},
			expectedCode: http.StatusNotFound,
			expectError:  "Page not found",
		},
		{
			name:         "invalid page number",
			pageNumber:   "invalid",
			medicaments:  []entities.Medicament{createTestMedicament(1, "Test Med 1")},
			expectedCode: http.StatusBadRequest,
			expectError:  "Invalid page number",
		},
		{
			name:         "negative page number",
			pageNumber:   "-1",
			medicaments:  []entities.Medicament{createTestMedicament(1, "Test Med 1")},
			expectedCode: http.StatusBadRequest,
			expectError:  "Invalid page number",
		},
		{
			name:         "zero page number",
			pageNumber:   "0",
			medicaments:  []entities.Medicament{createTestMedicament(1, "Test Med 1")},
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
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if message, ok := response["message"].(string); !ok || message != tt.expectError {
					t.Errorf("Expected error %s, got %v", tt.expectError, response["message"])
				}
			} else {
				// Verify pagination metadata
				var response map[string]interface{}
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
				createTestMedicament(1, "Doliprane"),
				createTestMedicament(2, "Ibuprofène"),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "empty search term",
			element:      "",
			medicaments:  []entities.Medicament{createTestMedicament(1, "Test Med")},
			expectedCode: http.StatusBadRequest,
			expectError:  "Missing search term",
		},
		{
			name:    "no results",
			element: "NonExistent",
			medicaments: []entities.Medicament{
				createTestMedicament(1, "Doliprane"),
			},
			expectedCode: http.StatusNotFound,
			expectError:  "No medicaments found",
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
				var response map[string]interface{}
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

// TestFindMedicamentByID tests medicament lookup by CIS
func TestFindMedicamentByID(t *testing.T) {
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
			cis:  "1",
			medicaments: []entities.Medicament{
				createTestMedicament(1, "Doliprane"),
			},
			medicamentsMap: map[int]entities.Medicament{
				1: createTestMedicament(1, "Doliprane"),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:           "invalid CIS (non-numeric)",
			cis:            "invalid",
			medicaments:    []entities.Medicament{createTestMedicament(1, "Test Med")},
			medicamentsMap: map[int]entities.Medicament{},
			expectedCode:   http.StatusBadRequest,
			expectError:    "Invalid CIS",
		},
		{
			name:           "non-existent CIS",
			cis:            "999",
			medicaments:    []entities.Medicament{createTestMedicament(1, "Test Med")},
			medicamentsMap: map[int]entities.Medicament{},
			expectedCode:   http.StatusNotFound,
			expectError:    "Medicament not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &MockDataStore{
				medicaments:    tt.medicaments,
				medicamentsMap: tt.medicamentsMap,
			}
			mockValidator := &MockDataValidator{}
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
				var response map[string]interface{}
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
				createTestGeneriqueList(1, "Paracetamol"),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "empty libelle",
			libelle:      "",
			generiques:   []entities.GeneriqueList{createTestGeneriqueList(1, "Test")},
			expectedCode: http.StatusBadRequest,
			expectError:  "Missing libelle",
		},
		{
			name:    "no results",
			libelle: "NonExistent",
			generiques: []entities.GeneriqueList{
				createTestGeneriqueList(1, "Test"),
			},
			expectedCode: http.StatusNotFound,
			expectError:  "No generiques found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &MockDataStore{generiques: tt.generiques}
			mockValidator := &MockDataValidator{}
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
				var response map[string]interface{}
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
	tests := []struct {
		name          string
		groupID       string
		generiques    []entities.GeneriqueList
		generiquesMap map[int]entities.Generique
		expectedCode  int
		expectError   string
	}{
		{
			name:    "valid group ID",
			groupID: "1",
			generiques: []entities.GeneriqueList{
				createTestGeneriqueList(1, "Test Group"),
			},
			generiquesMap: map[int]entities.Generique{
				1: {Group: 1, Libelle: "Test Group"},
			},
			expectedCode: http.StatusOK,
		},
		{
			name:          "invalid group ID (non-numeric)",
			groupID:       "invalid",
			generiques:    []entities.GeneriqueList{createTestGeneriqueList(1, "Test")},
			generiquesMap: map[int]entities.Generique{},
			expectedCode:  http.StatusBadRequest,
			expectError:   "Invalid group ID",
		},
		{
			name:          "non-existent group ID",
			groupID:       "999",
			generiques:    []entities.GeneriqueList{createTestGeneriqueList(1, "Test")},
			generiquesMap: map[int]entities.Generique{},
			expectedCode:  http.StatusNotFound,
			expectError:   "Generique group not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &MockDataStore{
				generiques:    tt.generiques,
				generiquesMap: tt.generiquesMap,
			}
			mockValidator := &MockDataValidator{}
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
				var response map[string]interface{}
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

// TestHealthCheck tests health check endpoint
func TestHealthCheck(t *testing.T) {
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
			medicaments:    []entities.Medicament{createTestMedicament(1, "Test Med")},
			generiques:     []entities.GeneriqueList{createTestGeneriqueList(1, "Test Group")},
			lastUpdated:    time.Now().Add(-1 * time.Hour),
			updating:       false,
			expectedCode:   http.StatusOK,
			expectedStatus: "healthy",
		},
		{
			name:           "system during update",
			medicaments:    []entities.Medicament{createTestMedicament(1, "Test Med")},
			generiques:     []entities.GeneriqueList{createTestGeneriqueList(1, "Test Group")},
			lastUpdated:    time.Now().Add(-1 * time.Hour),
			updating:       true,
			expectedCode:   http.StatusOK,
			expectedStatus: "healthy",
		},
		{
			name:           "stale data",
			medicaments:    []entities.Medicament{createTestMedicament(1, "Test Med")},
			generiques:     []entities.GeneriqueList{createTestGeneriqueList(1, "Test Group")},
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
			mockStore := &MockDataStore{
				medicaments: tt.medicaments,
				generiques:  tt.generiques,
				lastUpdated: tt.lastUpdated,
				updating:    tt.updating,
			}
			mockValidator := &MockDataValidator{}
			handler := NewHTTPHandler(mockStore, mockValidator)

			req := httptest.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()

			handler.HealthCheck(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			// Verify response structure
			var response map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to unmarshal JSON: %v", err)
			}

			// Check status
			if status, ok := response["status"].(string); !ok || status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, response["status"])
			}

			// Check required fields
			requiredFields := []string{"status", "last_update", "data_age_hours", "uptime_seconds", "data", "system"}
			for _, field := range requiredFields {
				if _, ok := response[field]; !ok {
					t.Errorf("Response should contain '%s' field", field)
				}
			}

			// Verify data field contains expected keys
			if data, ok := response["data"].(map[string]interface{}); ok {
				expectedDataKeys := []string{"api_version", "medicaments", "generiques", "is_updating", "next_update"}
				for _, key := range expectedDataKeys {
					if _, ok := data[key]; !ok {
						t.Errorf("Data should contain '%s' key", key)
					}
				}
			}

			// Verify system field contains expected keys
			if system, ok := response["system"].(map[string]interface{}); ok {
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

// BenchmarkServeAllMedicaments benchmarks the medicaments endpoint
func BenchmarkServeAllMedicaments(b *testing.B) {
	medicaments := make([]entities.Medicament, 1000)
	for i := 0; i < 1000; i++ {
		medicaments[i] = createTestMedicament(i, "Test Med "+string(rune(i)))
	}

	mockStore := &MockDataStore{medicaments: medicaments}
	mockValidator := &MockDataValidator{}
	handler := NewHTTPHandler(mockStore, mockValidator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/database", nil)
		handler.ServeAllMedicaments(rr, req)
	}
}

// BenchmarkFindMedicament benchmarks the medicament search
func BenchmarkFindMedicament(b *testing.B) {
	medicaments := make([]entities.Medicament, 1000)
	for i := 0; i < 1000; i++ {
		medicaments[i] = createTestMedicament(i, "Test Med "+string(rune(i)))
	}

	mockStore := &MockDataStore{medicaments: medicaments}
	mockValidator := &MockDataValidator{}
	handler := NewHTTPHandler(mockStore, mockValidator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("element", "Test Med")
		req := httptest.NewRequest("GET", "/medicament/Test Med", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		handler.FindMedicament(rr, req)
	}
}

// Phase 1: Utility Function Tests
// These tests target the utility functions in handlers.go to increase coverage from 36.8% to ~52%

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
			result := generateETag(tt.data)
			if !strings.HasPrefix(result, `"`) {
				t.Errorf("ETag should be quoted, got %s", result)
			}
			if !strings.HasSuffix(result, `"`) {
				t.Errorf("ETag should be quoted, got %s", result)
			}
			if len(strings.Trim(result, `"`)) != 16 {
				t.Errorf("ETag hash should be 16 hex characters (8 bytes), got %d", len(strings.Trim(result, `"`)))
			}
		})
	}

	// Test consistency - same data should always generate same ETag
	data := []byte("consistency test")
	etag1 := generateETag(data)
	etag2 := generateETag(data)
	if etag1 != etag2 {
		t.Errorf("ETag should be consistent for same data, got %s and %s", etag1, etag2)
	}

	// Test uniqueness - different data should generate different ETags
	data2 := []byte("consistency test modified")
	etag3 := generateETag(data2)
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
			ifNoneMatch:   `""`,
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

			match := checkETag(req, tt.currentETag)
			if match != tt.expectedMatch {
				t.Errorf("Expected match %v, got %v", tt.expectedMatch, match)
			}
		})
	}
}

func TestFormatUptimeHuman(t *testing.T) {
	tests := []struct {
		name     string
		uptime   time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			uptime:   0,
			expected: "0s",
		},
		{
			name:     "seconds only",
			uptime:   45 * time.Second,
			expected: "45s",
		},
		{
			name:     "one minute",
			uptime:   60 * time.Second,
			expected: "1m 0s",
		},
		{
			name:     "minutes and seconds",
			uptime:   2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "one hour",
			uptime:   60 * time.Minute,
			expected: "1h 0m 0s",
		},
		{
			name:     "hours and minutes",
			uptime:   2*time.Hour + 30*time.Minute,
			expected: "2h 30m 0s",
		},
		{
			name:     "hours, minutes, seconds",
			uptime:   2*time.Hour + 30*time.Minute + 45*time.Second,
			expected: "2h 30m 45s",
		},
		{
			name:     "one day",
			uptime:   24 * time.Hour,
			expected: "1d 0h 0m 0s",
		},
		{
			name:     "days and hours",
			uptime:   2*24*time.Hour + 6*time.Hour,
			expected: "2d 6h 0m 0s",
		},
		{
			name:     "complex duration",
			uptime:   3*24*time.Hour + 12*time.Hour + 30*time.Minute + 15*time.Second,
			expected: "3d 12h 30m 15s",
		},
		{
			name:     "very long duration",
			uptime:   30*24*time.Hour + 12*time.Hour + 45*time.Minute,
			expected: "30d 12h 45m 0s",
		},
		{
			name:     "milliseconds only",
			uptime:   500 * time.Millisecond,
			expected: "0s",
		},
		{
			name:     "with milliseconds",
			uptime:   2*time.Minute + 500*time.Millisecond,
			expected: "2m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUptimeHuman(tt.uptime)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}

	// Test with current time
	startTime := time.Now()
	time.Sleep(10 * time.Millisecond)
	uptime := time.Since(startTime)
	result := formatUptimeHuman(uptime)
	if !strings.Contains(result, "s") {
		t.Errorf("Should contain seconds, got %s", result)
	}
	if result == "" {
		t.Error("Should not be empty")
	}
}

func TestCalculateNextUpdate(t *testing.T) {
	// Since calculateNextUpdate() uses time.Now() internally, we can't test exact times
	// but we can test the logic and ensure it returns a valid time

	nextUpdate := calculateNextUpdate()

	// Should always return a valid time
	if nextUpdate.IsZero() {
		t.Error("Next update should not be zero time")
	}

	// Should be in the future or very close to now
	now := time.Now()
	timeDiff := nextUpdate.Sub(now)
	if timeDiff < 0 {
		t.Errorf("Next update should be in the future or now, got %v", timeDiff)
	}
	if timeDiff > 24*time.Hour {
		t.Errorf("Next update should be within 24 hours, got %v", timeDiff)
	}

	// Should be at either 6:00 or 18:00
	hour := nextUpdate.Hour()
	if hour != 6 && hour != 18 {
		t.Errorf("Next update should be at 6:00 or 18:00, got %d:00", hour)
	}

	// Minutes and seconds should be 0
	if nextUpdate.Minute() != 0 {
		t.Errorf("Minutes should be 0, got %d", nextUpdate.Minute())
	}
	if nextUpdate.Second() != 0 {
		t.Errorf("Seconds should be 0, got %d", nextUpdate.Second())
	}
}

// Phase 2: Response Function Tests
// These tests target the response functions in handlers.go to increase coverage from 44.3% to ~65%

func TestRespondWithJSONAndETag(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		payload        interface{}
		ifNoneMatch    string
		expectedStatus int
		expectBody     bool
		expectETag     bool
	}{
		{
			name:           "successful response with data",
			code:           http.StatusOK,
			payload:        map[string]string{"test": "data"},
			expectedStatus: http.StatusOK,
			expectBody:     true,
			expectETag:     true,
		},
		{
			name:           "created response with data",
			code:           http.StatusCreated,
			payload:        map[string]int{"id": 123},
			expectedStatus: http.StatusCreated,
			expectBody:     true,
			expectETag:     true,
		},
		{
			name:           "empty payload",
			code:           http.StatusOK,
			payload:        nil,
			expectedStatus: http.StatusOK,
			expectBody:     true,
			expectETag:     true,
		},
		{
			name:           "array payload",
			code:           http.StatusOK,
			payload:        []string{"item1", "item2"},
			expectedStatus: http.StatusOK,
			expectBody:     true,
			expectETag:     true,
		},
		{
			name:           "matching ETag returns 304",
			code:           http.StatusOK,
			payload:        map[string]string{"test": "data"},
			ifNoneMatch:    `"test-etag"`, // This will need to match the generated ETag
			expectedStatus: http.StatusNotModified,
			expectBody:     false,
			expectETag:     true,
		},
		{
			name:           "non-matching ETag returns full response",
			code:           http.StatusOK,
			payload:        map[string]string{"test": "data"},
			ifNoneMatch:    `"different-etag"`,
			expectedStatus: http.StatusOK,
			expectBody:     true,
			expectETag:     true,
		},
		{
			name:           "non-OK status with matching ETag returns full response",
			code:           http.StatusBadRequest,
			payload:        map[string]string{"error": "bad request"},
			ifNoneMatch:    `"test-etag"`,
			expectedStatus: http.StatusBadRequest,
			expectBody:     true,
			expectETag:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)

			if tt.ifNoneMatch != "" {
				req.Header.Set("If-None-Match", tt.ifNoneMatch)
			}

			// For the matching ETag test, we need to generate the ETag first
			if tt.name == "matching ETag returns 304" {
				data, _ := json.Marshal(tt.payload)
				expectedETag := generateETag(data)
				req.Header.Set("If-None-Match", expectedETag)
			}

			RespondWithJSONAndETag(rr, req, tt.code, tt.payload)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check headers
			if contentType := rr.Header().Get("Content-Type"); tt.expectBody && contentType != "application/json; charset=utf-8" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			etag := rr.Header().Get("ETag")
			if tt.expectETag && etag == "" {
				t.Error("Expected ETag header, but it's empty")
			}
			if !tt.expectETag && etag != "" {
				t.Error("Did not expect ETag header, but got one")
			}

			// Check cache headers
			if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "public, max-age=3600" {
				t.Errorf("Expected Cache-Control 'public, max-age=3600', got %s", cacheControl)
			}

			// Check Cloudflare headers (only set for non-304 responses)
			if tt.expectedStatus != http.StatusNotModified {
				if cfCacheStatus := rr.Header().Get("CF-Cache-Status"); cfCacheStatus != "DYNAMIC" {
					t.Errorf("Expected CF-Cache-Status 'DYNAMIC', got %s", cfCacheStatus)
				}
			} else {
				// For 304 responses, CF-Cache-Status should not be set
				if cfCacheStatus := rr.Header().Get("CF-Cache-Status"); cfCacheStatus != "" {
					t.Errorf("Did not expect CF-Cache-Status for 304 response, got %s", cfCacheStatus)
				}
			}

			// Check body
			body := rr.Body.String()
			if tt.expectBody && body == "" {
				t.Error("Expected response body, but it's empty")
			}
			if !tt.expectBody && body != "" {
				t.Errorf("Did not expect response body, but got: %s", body)
			}

			// Validate JSON if body is expected
			if tt.expectBody && body != "" {
				var result interface{}
				if err := json.Unmarshal([]byte(body), &result); err != nil {
					t.Errorf("Response body is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestRespondWithJSONAndETag_MarshalError(t *testing.T) {
	// Test with a payload that cannot be marshaled to JSON
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	// Use a function as payload (cannot be marshaled to JSON)
	RespondWithJSONAndETag(rr, req, http.StatusOK, func() {})

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}

	// Should not have Content-Type header on error
	if contentType := rr.Header().Get("Content-Type"); contentType != "" {
		t.Errorf("Did not expect Content-Type header on error, got %s", contentType)
	}
}

func TestRespondWithJSON_Standalone(t *testing.T) {
	tests := []struct {
		name       string
		code       int
		payload    interface{}
		expectBody bool
	}{
		{
			name:       "successful response with data",
			code:       http.StatusOK,
			payload:    map[string]string{"test": "data"},
			expectBody: true,
		},
		{
			name:       "error response with data",
			code:       http.StatusBadRequest,
			payload:    map[string]string{"error": "bad request"},
			expectBody: true,
		},
		{
			name:       "empty payload",
			code:       http.StatusNoContent,
			payload:    nil,
			expectBody: true,
		},
		{
			name:       "array payload",
			code:       http.StatusOK,
			payload:    []string{"item1", "item2"},
			expectBody: true,
		},
		{
			name:       "numeric payload",
			code:       http.StatusOK,
			payload:    42,
			expectBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			RespondWithJSON(rr, tt.code, tt.payload)

			// Check status code
			if rr.Code != tt.code {
				t.Errorf("Expected status %d, got %d", tt.code, rr.Code)
			}

			// Check headers
			if contentType := rr.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Check Last-Modified header
			if lastModified := rr.Header().Get("Last-Modified"); lastModified == "" {
				t.Error("Expected Last-Modified header, but it's empty")
			}

			// Check body
			body := rr.Body.String()
			if tt.expectBody && body == "" {
				t.Error("Expected response body, but it's empty")
			}

			// Validate JSON if body is expected
			if tt.expectBody && body != "" {
				var result interface{}
				if err := json.Unmarshal([]byte(body), &result); err != nil {
					t.Errorf("Response body is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestRespondWithJSON_MarshalError_Standalone(t *testing.T) {
	// Test with a payload that cannot be marshaled to JSON
	rr := httptest.NewRecorder()

	// Use a function as payload (cannot be marshaled to JSON)
	RespondWithJSON(rr, http.StatusOK, func() {})

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}

	// Should not have Content-Type header on error
	if contentType := rr.Header().Get("Content-Type"); contentType != "" {
		t.Errorf("Did not expect Content-Type header on error, got %s", contentType)
	}
}

func TestRespondWithError_Standalone(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		message        string
		expectedStatus int
	}{
		{
			name:           "bad request error",
			code:           http.StatusBadRequest,
			message:        "Invalid input",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found error",
			code:           http.StatusNotFound,
			message:        "Resource not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "internal server error",
			code:           http.StatusInternalServerError,
			message:        "Something went wrong",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "unauthorized error",
			code:           http.StatusUnauthorized,
			message:        "Access denied",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty error message",
			code:           http.StatusBadRequest,
			message:        "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			RespondWithError(rr, tt.code, tt.message)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check headers
			if contentType := rr.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Check body
			body := rr.Body.String()
			if body == "" {
				t.Error("Expected response body, but it's empty")
			}

			// Parse and validate JSON response
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				t.Errorf("Response body is not valid JSON: %v", err)
			}

			// Check error field
			if errorField, exists := response["error"]; !exists {
				t.Error("Expected 'error' field in response")
			} else if errorStr, ok := errorField.(string); !ok {
				t.Error("Expected 'error' field to be a string")
			} else if errorStr == "" {
				t.Error("Expected 'error' field to not be empty")
			}

			// Check message field
			if messageField, exists := response["message"]; !exists {
				t.Error("Expected 'message' field in response")
			} else if messageStr, ok := messageField.(string); !ok {
				t.Error("Expected 'message' field to be a string")
			} else if messageStr != tt.message {
				t.Errorf("Expected message '%s', got '%s'", tt.message, messageStr)
			}

			// Check code field
			if codeField, exists := response["code"]; !exists {
				t.Error("Expected 'code' field in response")
			} else if codeFloat, ok := codeField.(float64); !ok {
				t.Error("Expected 'code' field to be a number")
			} else if int(codeFloat) != tt.code {
				t.Errorf("Expected code %d, got %d", tt.code, int(codeFloat))
			}
		})
	}
}
