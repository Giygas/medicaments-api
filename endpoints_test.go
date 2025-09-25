package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/giygas/medicamentsfr/medicamentsparser/entities"
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
		{"Test generiques/group/999999", "/generiques/group/999999", http.StatusNotFound},
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
