package handlers

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

// ============================================================================
// BENCHMARKS
// ============================================================================

// BenchmarkServeAllMedicaments benchmarks medicaments endpoint
func BenchmarkServeAllMedicaments(b *testing.B) {
	factory := NewTestDataFactory()
	medicaments := make([]entities.Medicament, 1000)
	for i := 0; i < 1000; i++ {
		medicaments[i] = factory.CreateMedicament(i, fmt.Sprintf("Test Med %d", i))
	}

	mockStore := NewMockDataStoreBuilder().WithMedicaments(medicaments).Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/database", nil)
		handler.ServeAllMedicaments(rr, req)
	}
}

// BenchmarkFindMedicament benchmarks medicament search
func BenchmarkFindMedicament(b *testing.B) {
	factory := NewTestDataFactory()
	medicaments := make([]entities.Medicament, 1000)
	for i := 0; i < 1000; i++ {
		medicaments[i] = factory.CreateMedicament(i, fmt.Sprintf("Test Med %d", i))
	}

	mockStore := NewMockDataStoreBuilder().WithMedicaments(medicaments).Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("element", "Test Med")
		req := httptest.NewRequest("GET", "/medicament/Test+Med", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		handler.FindMedicament(rr, req)
	}
}
