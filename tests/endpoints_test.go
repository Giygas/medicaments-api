package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/server"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-chi/chi/v5"
)

// Mock data for testing
var testMedicaments = []entities.Medicament{
	{
		Cis:                    12345678,
		Denomination:           "Test Medicament",
		DenominationNormalized: "test medicament",
		Generiques:             []entities.Generique{{Cis: 12345678, Group: 100, Libelle: "Test Group", Type: "Princeps"}},
	},
}

var testGeneriques = []entities.GeneriqueList{
	{
		GroupID:           100,
		Libelle:           "Test Group",
		LibelleNormalized: "test group",
		Medicaments: []entities.GeneriqueMedicament{
			{
				Cis:                 1,
				Denomination:        "Test Medicament",
				FormePharmaceutique: "Tablet",
				Type:                "Princeps",
				Composition:         []entities.GeneriqueComposition{},
			},
		},
	},
}

var testMedicamentsMap = map[int]entities.Medicament{
	12345678: testMedicaments[0],
}

var testGeneriquesMap = map[int]entities.GeneriqueList{
	100: {GroupID: 100, Libelle: "Test Group"},
}

// Global test data container
var testDataContainer *data.DataContainer

func TestMain(m *testing.M) {
	fmt.Println("Initializing test data...")
	// Initialize mock data for tests
	testDataContainer = data.NewDataContainer()
	testDataContainer.UpdateData(testMedicaments, testGeneriques, testMedicamentsMap, testGeneriquesMap,
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
	fmt.Printf("Mock data initialized: %d medicaments, %d generiques\n", len(testMedicaments), len(testGeneriques))

	fmt.Println("Running tests...")
	exitVal := m.Run()
	fmt.Printf("Tests completed with exit code: %d\n", exitVal)
	os.Exit(exitVal)
}

func TestEndpoints(t *testing.T) {

	testCases := []struct {
		name     string
		endpoint string
		expected int
	}{
		{"Test medicaments export", "/v1/medicaments/export", http.StatusOK},
		{"Test medicaments pagination", "/v1/medicaments?page=1", http.StatusOK},
		{"Test medicaments search", "/v1/medicaments?search=Test", http.StatusOK},
		{"Test medicaments by CIS", "/v1/medicaments/12345678", http.StatusOK},
		{"Test medicaments by invalid CIS", "/v1/medicaments/99999999", http.StatusNotFound},
		{"Test generiques by libelle", "/v1/generiques?libelle=Test Group", http.StatusOK},
		{"Test generiques by group ID", "/v1/generiques/100", http.StatusOK},
		{"Test generiques with no params", "/v1/generiques", http.StatusBadRequest},
		{"Test generiques with invalid group ID", "/v1/generiques/invalid", http.StatusBadRequest},
		{"Test generiques with not found group ID", "/v1/generiques/99999", http.StatusNotFound},
		{"Test medicaments no params", "/v1/medicaments", http.StatusBadRequest},
		{"Test medicaments multiple params", "/v1/medicaments?page=1&search=test", http.StatusBadRequest},
		{"Test medicaments invalid page", "/v1/medicaments?page=0", http.StatusBadRequest},
		{"Test medicaments negative page", "/v1/medicaments?page=-1", http.StatusBadRequest},
		{"Test medicaments invalid CIS", "/v1/medicaments/abc", http.StatusBadRequest},
		{"Test health", "/health", http.StatusOK},
	}

	router := chi.NewRouter()
	// Note: rateLimitHandler is now part of server middleware
	validator := validation.NewDataValidator()
	httpHandler := handlers.NewHTTPHandler(testDataContainer, validator)
	router.Get("/v1/medicaments/export", httpHandler.ExportMedicaments)
	router.Get("/v1/medicaments", httpHandler.ServeMedicamentsV1)
	router.Get("/v1/medicaments/{cis}", httpHandler.FindMedicamentByCIS)
	router.Get("/v1/generiques/{groupID}", httpHandler.FindGeneriquesByGroupID)
	router.Get("/v1/generiques", httpHandler.ServeGeneriquesV1)
	router.Get("/v1/presentations/{cip}", httpHandler.ServePresentationsV1)
	router.Get("/health", httpHandler.HealthCheck)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("Testing %s: %s\n", tt.name, tt.endpoint)
			req, err := http.NewRequest("GET", tt.endpoint, nil)

			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			status := rr.Code
			fmt.Printf("  Status: %d (expected %d)\n", status, tt.expected)
			if status != tt.expected {
				t.Errorf("%v returned wrong status code: got %v want %v", tt.endpoint, status, tt.expected)
			} else {
				fmt.Printf("  ✓ Passed\n")
			}
		})
	}
}

func TestRealIPMiddleware(t *testing.T) {
	fmt.Println("Testing realIPMiddleware...")

	// Create a test request with X-Forwarded-For header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1")

	// Create a response recorder
	w := httptest.NewRecorder()

	// Create a handler that will check the RemoteAddr
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RemoteAddr != "203.0.113.1" {
			t.Errorf("Expected RemoteAddr to be '203.0.113.1', got '%s'", r.RemoteAddr)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Apply the middleware
	middlewareHandler := server.RealIPMiddleware(handler)
	middlewareHandler.ServeHTTP(w, req)

	fmt.Println("realIPMiddleware test completed")
}

func TestBlockDirectAccessMiddleware(t *testing.T) {
	fmt.Println("Testing blockDirectAccessMiddleware...")

	router := chi.NewRouter()
	router.Use(server.BlockDirectAccessMiddleware)
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("allowed"))
	})

	// Test without nginx headers (should be blocked)
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", rr.Code)
	}

	// Test with X-Forwarded-For header (should be allowed)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	// Test localhost access (should be allowed)
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for localhost, got %d", rr.Code)
	}

	// Test localhost access with hostname (should be allowed)
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "localhost:8002"
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for localhost hostname, got %d", rr.Code)
	}

	fmt.Println("blockDirectAccessMiddleware test completed")
}

func TestRateLimiter(t *testing.T) {
	fmt.Println("Testing rate limiter...")

	router := chi.NewRouter()
	// Apply rate limiting middleware
	router.Use(server.RateLimitHandler)
	validator := validation.NewDataValidator()
	httpHandler := handlers.NewHTTPHandler(testDataContainer, validator)
	router.Get("/v1/medicaments/export", httpHandler.ExportMedicaments)

	// Simulate requests from same IP
	clientIP := "192.168.1.1:12345"

	// Make requests to /v1/medicaments/export until we get rate limited
	// Each costs 200 tokens, so we should be able to make 5 requests (1000 tokens)
	requestCount := 0
	for requestCount = 0; requestCount < 10; requestCount++ {
		req, _ := http.NewRequest("GET", "/v1/medicaments/export", nil)
		req.RemoteAddr = clientIP
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code == http.StatusTooManyRequests {
			break // Rate limited as expected
		}

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: Expected 200 or 429, got %d", requestCount+1, rr.Code)
			break
		}
	}

	// Should have been rate limited by now (after 5 requests)
	if requestCount >= 10 {
		t.Errorf("Expected to be rate limited after 5 requests, but wasn't")
	} else {
		fmt.Printf("Rate limited after %d requests (expected around 5)\n", requestCount)
	}

	fmt.Println("Rate limiter test completed")
}

func TestRequestSizeMiddleware(t *testing.T) {
	fmt.Println("Testing request size middleware...")

	// Create test configuration
	cfg := &config.Config{
		MaxRequestBody: 1024, // 1KB for testing
		MaxHeaderSize:  512,  // 512 bytes for testing
		Port:           "8002",
		Address:        "127.0.0.1",
		Env:            config.EnvTest,
		LogLevel:       "info",
	}

	// Test handler that simply returns 200 OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap the test handler with our middleware
	middleware := server.RequestSizeMiddleware(cfg)
	protectedHandler := middleware(testHandler)

	t.Run("Valid request - small body", func(t *testing.T) {
		body := []byte("small request body")
		req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Valid request - no content length", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Invalid request - body too large via Content-Length", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Content-Length", "2048") // Larger than MaxRequestBody (1024)
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", w.Code)
		}

		// Check response body contains error message
		if w.Body.Len() == 0 {
			t.Error("Expected error response body, got empty")
		}
	})

	t.Run("Invalid request - headers too large", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		// Add many large headers to exceed MaxHeaderSize (512 bytes)
		for i := range 20 {
			req.Header.Set(fmt.Sprintf("X-Large-Header-%d", i), fmt.Sprintf("%0200d", i))
		}

		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusRequestHeaderFieldsTooLarge {
			t.Errorf("Expected status 431, got %d", w.Code)
		}

		// Check response body contains error message
		if w.Body.Len() == 0 {
			t.Error("Expected error response body, got empty")
		}
	})

	t.Run("Valid request - exact size limit", func(t *testing.T) {
		body := make([]byte, 1024) // Exactly MaxRequestBody
		req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req.Header.Set("Content-Length", "1024")
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Invalid request - body just over limit", func(t *testing.T) {
		body := make([]byte, 1025) // Just over MaxRequestBody
		req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req.Header.Set("Content-Length", "1025")
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", w.Code)
		}
	})

	t.Run("Invalid Content-Length header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Content-Length", "invalid")
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		// Should pass through when Content-Length is invalid (can't parse)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for invalid Content-Length, got %d", w.Code)
		}
	})

	fmt.Println("Request size middleware test completed")
}

func TestCompressionOptimization(t *testing.T) {
	fmt.Println("Testing compression optimization...")

	// Test basic JSON response functionality
	t.Run("Basic JSON response", func(t *testing.T) {
		w := httptest.NewRecorder()

		// Create handler and test actual endpoint compression
		validator := validation.NewDataValidator()
		httpHandler := handlers.NewHTTPHandler(testDataContainer, validator)
		req := httptest.NewRequest("GET", "/v1/medicaments/export", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		httpHandler.ExportMedicaments(w, req)

		// Check that response was written correctly
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
		}
	})

	fmt.Println("Compression optimization test completed")
}
