package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

func TestETagFunctionality(t *testing.T) {
	// Initialize test data
	dataContainer := data.NewDataContainer()

	// Create test medicament
	testMedicaments := []entities.Medicament{
		{
			Cis:                   123456,
			Denomination:          "Test Medicament",
			FormePharmaceutique:   "comprimé",
			VoiesAdministration:   []string{"orale"},
			StatusAutorisation:    "Autorisation active",
			TypeProcedure:         "Procédure nationale",
			EtatComercialisation:  "Commercialisée",
			DateAMM:               "2023-01-01",
			Titulaire:             "Test Lab",
			SurveillanceRenforcee: "Non",
		},
	}

	// Update data container
	dataContainer.UpdateData(testMedicaments, []entities.GeneriqueList{},
		map[int]entities.Medicament{123456: testMedicaments[0]},
		map[int]entities.Generique{})

	// Test ServeAllMedicaments with ETag
	handler := handlers.ServeAllMedicaments(dataContainer)

	// First request - should return 200 with ETag
	req1 := httptest.NewRequest("GET", "/database", nil)
	w1 := httptest.NewRecorder()
	handler(w1, req1)

	resp1 := w1.Result()
	etag1 := resp1.Header.Get("ETag")

	if etag1 == "" {
		t.Error("Expected ETag header in first response")
	}

	if resp1.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp1.StatusCode)
	}

	// Second request with If-None-Match - should return 304
	req2 := httptest.NewRequest("GET", "/database", nil)
	req2.Header.Set("If-None-Match", etag1)
	w2 := httptest.NewRecorder()
	handler(w2, req2)

	resp2 := w2.Result()

	if resp2.StatusCode != http.StatusNotModified {
		t.Errorf("Expected status 304, got %d", resp2.StatusCode)
	}

	// Verify no body in 304 response
	body2 := w2.Body.String()
	if body2 != "" {
		t.Error("Expected empty body for 304 response")
	}

	// Test with different ETag - should return 200
	req3 := httptest.NewRequest("GET", "/database", nil)
	req3.Header.Set("If-None-Match", `"different-etag"`)
	w3 := httptest.NewRecorder()
	handler(w3, req3)

	resp3 := w3.Result()

	if resp3.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for different ETag, got %d", resp3.StatusCode)
	}

	// Test FindMedicamentByID with ETag
	idHandler := handlers.FindMedicamentByID(dataContainer)

	// First request - we need to set up chi context for URLParam to work
	req4 := httptest.NewRequest("GET", "/medicament/id/123456", nil)

	// Set up chi context to simulate URL parameter
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("cis", "123456")
	req4 = req4.WithContext(context.WithValue(req4.Context(), chi.RouteCtxKey, chiCtx))

	w4 := httptest.NewRecorder()
	idHandler(w4, req4)

	resp4 := w4.Result()
	etag4 := resp4.Header.Get("ETag")

	if etag4 == "" {
		t.Error("Expected ETag header in medicament by ID response")
	}

	// Second request with matching ETag
	req5 := httptest.NewRequest("GET", "/medicament/id/123456", nil)
	req5.Header.Set("If-None-Match", etag4)

	// Set up chi context again
	chiCtx2 := chi.NewRouteContext()
	chiCtx2.URLParams.Add("cis", "123456")
	req5 = req5.WithContext(context.WithValue(req5.Context(), chi.RouteCtxKey, chiCtx2))

	w5 := httptest.NewRecorder()
	idHandler(w5, req5)

	resp5 := w5.Result()

	if resp5.StatusCode != http.StatusNotModified {
		t.Errorf("Expected status 304 for medicament by ID with matching ETag, got %d", resp5.StatusCode)
	}
}
