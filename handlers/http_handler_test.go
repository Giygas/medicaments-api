package handlers

import (
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
		name          string
		dataStore     interfaces.DataStore
		validator     interfaces.DataValidator
		healthChecker interfaces.HealthChecker
	}{
		{
			name:          "valid dependencies",
			dataStore:     NewMockDataStoreBuilder().Build(),
			validator:     NewMockDataValidatorBuilder().Build(),
			healthChecker: NewMockHealthCheckerBuilder().Build(),
		},
		{
			name:          "nil data store",
			dataStore:     nil,
			validator:     NewMockDataValidatorBuilder().Build(),
			healthChecker: NewMockHealthCheckerBuilder().Build(),
		},
		{
			name:          "nil validator",
			dataStore:     NewMockDataStoreBuilder().Build(),
			validator:     nil,
			healthChecker: NewMockHealthCheckerBuilder().Build(),
		},
		{
			name:          "nil health checker",
			dataStore:     NewMockDataStoreBuilder().Build(),
			validator:     NewMockDataValidatorBuilder().Build(),
			healthChecker: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHTTPHandler(tt.dataStore, tt.validator, tt.healthChecker)

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
	mockHealthChecker := NewMockHealthCheckerBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, mockHealthChecker).(*Handler)

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
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build()).(*Handler)

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
			handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

			req := httptest.NewRequest("GET", "/v1/medicaments/export", nil)
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

			// Check for weak ETag format: W/"hash"
			actualETag := strings.TrimPrefix(etag, "W/")
			if !strings.HasPrefix(actualETag, "\"") || !strings.HasSuffix(actualETag, "\"") {
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

// TestLegacyEndpointsReturn410 verifies that the removed legacy endpoints
// (sunset 2026-07-31) respond 410 Gone with deprecation headers pointing to
// their v1 successors.
func TestLegacyEndpointsReturn410(t *testing.T) {
	handler := NewHTTPHandler(
		NewMockDataStoreBuilder().Build(),
		NewMockDataValidatorBuilder().Build(),
		NewMockHealthCheckerBuilder().Build(),
	)

	tests := []struct {
		name        string
		pattern     string
		requestPath string
		successor   string
		handler     func(http.ResponseWriter, *http.Request)
	}{
		{"database export", "/database", "/database", "/v1/medicaments/export", handler.ExportMedicaments},
		{"database page", "/database/{pageNumber}", "/database/2", "/v1/medicaments?page=2", handler.ServePagedMedicaments},
		{"medicament search", "/medicament/{element}", "/medicament/paracetamol", "/v1/medicaments?search=paracetamol", handler.FindMedicament},
		{"medicament by CIS", "/medicament/id/{cis}", "/medicament/id/12345678", "/v1/medicaments/12345678", handler.FindMedicamentByCIS},
		{"medicament by CIP", "/medicament/cip/{cip}", "/medicament/cip/1234567", "/v1/medicaments?cip=1234567", handler.FindMedicamentByCIP},
		{"generiques by libelle", "/generiques/{libelle}", "/generiques/paracetamol", "/v1/generiques?libelle=paracetamol", handler.FindGeneriques},
		{"generiques by group", "/generiques/group/{groupId}", "/generiques/group/100", "/v1/generiques/100", handler.FindGeneriquesByGroupID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a chi router with the legacy route, mirroring server.go registration
			router := chi.NewRouter()
			router.Get(tt.pattern, tt.handler)

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusGone {
				t.Fatalf("Expected status 410 Gone, got %d", rr.Code)
			}

			// Verify deprecation headers
			if rr.Header().Get("Deprecation") != "true" {
				t.Error("Expected Deprecation header 'true'")
			}

			if rr.Header().Get("Sunset") != "2026-07-31T23:59:59Z" {
				t.Errorf("Expected Sunset header '2026-07-31T23:59:59Z', got %q", rr.Header().Get("Sunset"))
			}

			link := rr.Header().Get("Link")
			if !strings.Contains(link, tt.successor) || !strings.Contains(link, `rel="successor-version"`) {
				t.Errorf("Expected Link header with successor %s and rel=successor-version, got %q", tt.successor, link)
			}

			if xDeprecated := rr.Header().Get("X-Deprecated"); !strings.Contains(xDeprecated, tt.successor) {
				t.Errorf("Expected X-Deprecated header to contain %s, got %q", tt.successor, xDeprecated)
			}

			if warning := rr.Header().Get("Warning"); !strings.Contains(warning, "299") {
				t.Errorf("Expected Warning header 299, got %q", warning)
			}

			// Verify JSON error body mentions the successor
			var response map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}
			if message, ok := response["message"].(string); !ok || !strings.Contains(message, tt.successor) {
				t.Errorf("Expected error message to contain %s, got %v", tt.successor, response["message"])
			}
		})
	}
}

// TestFindMedicamentByCIS tests medicament lookup by CIS
func TestFindMedicamentByCIS(t *testing.T) {
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
			handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

			// Create a chi router with the v1 route (shares the handler with the removed legacy route)
			router := chi.NewRouter()
			router.Get("/v1/medicaments/{cis}", handler.FindMedicamentByCIS)

			req := httptest.NewRequest("GET", "/v1/medicaments/"+tt.cis, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

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
			handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

			// Create a chi router with the v1 route (shares the handler with the removed legacy route)
			router := chi.NewRouter()
			router.Get("/v1/generiques/{groupID}", handler.FindGeneriquesByGroupID)

			req := httptest.NewRequest("GET", "/v1/generiques/"+tt.groupID, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

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
			expectedCode:   http.StatusServiceUnavailable,
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
			data := map[string]any{
				"last_update":    tt.lastUpdated.Format(time.RFC3339),
				"data_age_hours": time.Since(tt.lastUpdated).Hours(),
				"medicaments":    len(tt.medicaments),
				"generiques":     len(tt.generiques),
				"is_updating":    tt.updating,
			}
			mockHealthChecker := NewMockHealthCheckerBuilder().
				WithStatus(tt.expectedStatus).
				WithData(data).
				WithHTTPStatus(tt.expectedCode).
				Build()
			mockValidator := NewMockDataValidatorBuilder().Build()
			mockStore := NewMockDataStoreBuilder().
				WithMedicaments(tt.medicaments).
				WithGeneriques(tt.generiques).
				WithLastUpdated(tt.lastUpdated).
				WithUpdating(tt.updating).
				Build()
			handler := NewHTTPHandler(mockStore, mockValidator, mockHealthChecker)

			req := httptest.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()

			handler.HealthCheck(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			if !mockHealthChecker.WasHealthCalled() {
				t.Error("HealthChecker.HealthCheckHTTP() should have been called")
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
			requiredFields := []string{"status", "data"}
			for _, field := range requiredFields {
				if _, ok := response[field]; !ok {
					t.Errorf("Response should contain '%s' field", field)
				}
			}

			// Verify data field contains expected keys
			if data, ok := response["data"].(map[string]any); ok {
				expectedDataKeys := []string{"last_update", "data_age_hours", "medicaments", "generiques", "is_updating"}
				for _, key := range expectedDataKeys {
					if _, ok := data[key]; !ok {
						t.Errorf("Data should contain '%s' key", key)
					}
				}
			}

			// Verify removed fields are not present
			removedFields := []string{"uptime_seconds", "system"}
			for _, field := range removedFields {
				if _, ok := response[field]; ok {
					t.Errorf("Response should not contain '%s' field (removed)", field)
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

			// Check for weak ETag format: W/"hash"
			actualETag := strings.TrimPrefix(result, "W/")
			if !strings.HasPrefix(actualETag, `"`) {
				t.Errorf("ETag should be quoted, got %s", result)
			}
			if !strings.HasSuffix(actualETag, `"`) {
				t.Errorf("ETag should be quoted, got %s", result)
			}
			// ETag hash should be 16 hex characters (8 bytes) after trimming quotes and W/ prefix
			etagContent := string(actualETag[1 : len(actualETag)-1])
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
