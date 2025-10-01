package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

// Mock data for testing
var testMedicaments = []entities.Medicament{
	{
		Cis:          1,
		Denomination: "Test Medicament",
		Generiques:   []entities.Generique{{Cis: 1, Group: 100, Libelle: "Test Group", Type: "Princeps"}},
	},
}

var testGeneriques = []entities.GeneriqueList{
	{
		GroupID: 100,
		Libelle: "Test Group",
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
	1: testMedicaments[0],
}

var testGeneriquesMap = map[int]entities.Generique{
	100: {Cis: 1, Group: 100, Libelle: "Test Group", Type: "Princeps"},
}

func isDatabaseReady() bool {
	return len(GetMedicaments()) > 0
}

func TestMain(m *testing.M) {
	fmt.Println("Initializing test data...")
	// Initialize mock data for tests
	dataContainer.medicaments.Store(testMedicaments)
	dataContainer.generiques.Store(testGeneriques)
	dataContainer.medicamentsMap.Store(testMedicamentsMap)
	dataContainer.generiquesMap.Store(testGeneriquesMap)
	dataContainer.lastUpdated.Store(time.Now())
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

		{"Test database", "/database", http.StatusOK},
		{"Test database with trailing slash", "/database/", http.StatusNotFound}, // Chi doesn't handle trailing slash
		{"Test generiques/Test Group", "/generiques/Test Group", http.StatusOK},
		{"Test generiques/group/100", "/generiques/group/100", http.StatusOK},
		{"Test medicament/Test Medicament", "/medicament/Test Medicament", http.StatusOK},
		{"Test database with a", "/database/a", http.StatusBadRequest},
		{"Test database with 1", "/database/1", http.StatusOK},
		{"Test database with 0", "/database/0", http.StatusBadRequest},
		{"Test database with -1", "/database/-1", http.StatusBadRequest},
		{"Test database with large number", "/database/10000", http.StatusNotFound}, // Only 1 page available
		{"Test generiques", "/generiques", http.StatusNotFound},
		{"Test generiques/aaaaaaaaaaa", "/generiques/aaaaaaaaaaa", http.StatusNotFound},
		{"Test medicament", "/medicament", http.StatusNotFound},
		{"Test medicament/1000000000000000", "/medicament/100000000000000", http.StatusNotFound},
		{"Test medicament/id/1", "/medicament/id/1", http.StatusOK},
		{"Test medicament/id/999999", "/medicament/id/999999", http.StatusNotFound},
		{"Test generiques/group/a", "/generiques/group/a", http.StatusBadRequest},
		{"Test generiques/group/999999", "/generiques/group/999999", http.StatusBadRequest},
		{"Test health", "/health", http.StatusOK},
	}

	router := chi.NewRouter()
	router.Use(rateLimitHandler)

	router.Get("/database/{pageNumber}", servePagedMedicaments)
	router.Get("/database", serveAllMedicaments)
	router.Get("/medicament/{element}", findMedicament)
	router.Get("/medicament/id/{cis}", findMedicamentByID)
	router.Get("/generiques/{libelle}", findGeneriques)
	router.Get("/generiques/group/{groupId}", findGeneriquesByGroupID)
	router.Get("/health", healthCheck)

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

	router := chi.NewRouter()
	router.Use(realIPMiddleware)
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.RemoteAddr))
	})

	// Test with X-Forwarded-For header
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Body.String() != "192.168.1.1:0" {
		t.Errorf("Expected IP 192.168.1.1:0, got %s", rr.Body.String())
	}

	fmt.Println("realIPMiddleware test completed")
}

func TestBlockDirectAccessMiddleware(t *testing.T) {
	fmt.Println("Testing blockDirectAccessMiddleware...")

	router := chi.NewRouter()
	router.Use(blockDirectAccessMiddleware)
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("allowed"))
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

	fmt.Println("blockDirectAccessMiddleware test completed")
}

func TestRateLimiter(t *testing.T) {
	fmt.Println("Testing rate limiter...")

	router := chi.NewRouter()
	router.Use(rateLimitHandler)
	router.Get("/database", serveAllMedicaments)

	// Simulate requests from the same IP
	clientIP := "192.168.1.1:12345"

	// Make 5 requests to /database (each costs 200 tokens, total 1000 tokens)
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/database", nil)
		req.RemoteAddr = clientIP
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: Expected 200, got %d", i+1, rr.Code)
		}
	}

	// 6th request should be rate limited (exceeds 1000 tokens)
	req, _ := http.NewRequest("GET", "/database", nil)
	req.RemoteAddr = clientIP
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("6th request: Expected 429, got %d", rr.Code)
	}

	// Wait for refill (10 tokens/second, need to refill 200 tokens for another request)
	time.Sleep(21 * time.Second) // 200 / 10 = 20 seconds + buffer

	// Now should allow another request
	req, _ = http.NewRequest("GET", "/database", nil)
	req.RemoteAddr = clientIP
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("After refill: Expected 200, got %d", rr.Code)
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
		Env:            "test",
		LogLevel:       "info",
	}

	// Test handler that simply returns 200 OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap the test handler with our middleware
	middleware := requestSizeMiddleware(cfg)
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
		for i := 0; i < 20; i++ {
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
