package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-chi/chi/v5"
)

var (
	benchmarkContainer *data.DataContainer
	benchmarkOnce      sync.Once
)

// Create realistic test data for benchmarks using full medicaments database
// Cache the data to avoid re-downloading/parsing for each benchmark
func createBenchmarkData() *data.DataContainer {
	benchmarkOnce.Do(func() {
		fmt.Println("Loading full medicaments database for benchmarks...")

		// Parse the full medicaments database for realistic performance testing
		medicaments, err := medicamentsparser.ParseAllMedicaments()
		if err != nil {
			panic(fmt.Sprintf("Failed to parse medicaments for benchmarks: %v", err))
		}

		// Create medicaments map as done in production
		medicamentsMap := make(map[int]entities.Medicament)
		for i := range medicaments {
			medicamentsMap[medicaments[i].Cis] = medicaments[i]
		}

		// Parse generiques with real data
		generiques, generiquesMap, err := medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse generiques for benchmarks: %v", err))
		}

		benchmarkContainer = data.NewDataContainer()
		benchmarkContainer.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap)

		fmt.Printf("Benchmark data loaded: %d medicaments, %d generiques\n", len(medicaments), len(generiques))
	})

	return benchmarkContainer
}

// Benchmark database endpoint (full data)
func BenchmarkDatabase(b *testing.B) {
	container := createBenchmarkData()
	handler := handlers.ServeAllMedicaments(container)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/database", nil)
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

// Benchmark paginated database endpoint
func BenchmarkDatabasePage(b *testing.B) {
	container := createBenchmarkData()
	validator := validation.NewDataValidator()
	handler := handlers.NewHTTPHandler(container, validator)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/database/1", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		// Create chi router context to properly extract URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("pageNumber", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.ServePagedMedicaments(w, req)
	}
}

// Benchmark medicament search by name
func BenchmarkMedicamentSearch(b *testing.B) {
	container := createBenchmarkData()
	validator := validation.NewDataValidator()
	handler := handlers.NewHTTPHandler(container, validator)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/medicament/Medicament", nil)
		w := httptest.NewRecorder()

		// Create chi router context to properly extract URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("element", "Medicament")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.FindMedicament(w, req)
	}
}

// Benchmark medicament lookup by CIS
func BenchmarkMedicamentByID(b *testing.B) {
	container := createBenchmarkData()
	handler := handlers.NewHTTPHandler(container, validation.NewDataValidator())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/medicament/id/500", nil)
		w := httptest.NewRecorder()

		// Create chi router context to properly extract URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("cis", "500")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.FindMedicamentByID(w, req)
	}
}

// Benchmark generiques search
func BenchmarkGeneriquesSearch(b *testing.B) {
	container := createBenchmarkData()
	validator := validation.NewDataValidator()
	handler := handlers.NewHTTPHandler(container, validator)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/generiques/Groupe", nil)
		w := httptest.NewRecorder()

		// Create chi router context to properly extract URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("libelle", "Groupe")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.FindGeneriques(w, req)
	}
}

// Benchmark generiques by group ID
func BenchmarkGeneriquesByID(b *testing.B) {
	container := createBenchmarkData()
	handler := handlers.NewHTTPHandler(container, validation.NewDataValidator())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/generiques/group/50", nil)
		w := httptest.NewRecorder()

		// Create chi router context to properly extract URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("groupId", "50")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.FindGeneriquesByGroupID(w, req)
	}
}

// Benchmark health check
func BenchmarkHealth(b *testing.B) {
	container := createBenchmarkData()
	handler := handlers.NewHTTPHandler(container, validation.NewDataValidator())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		handler.HealthCheck(w, req)
	}
}

// Benchmark full router with middleware
func BenchmarkFullRouter(b *testing.B) {
	container := createBenchmarkData()
	validator := validation.NewDataValidator()

	router := chi.NewRouter()
	router.Get("/database", handlers.ServeAllMedicaments(container))
	router.Get("/database/{pageNumber}", handlers.ServePagedMedicaments(container))
	router.Get("/medicament/{element}", handlers.FindMedicament(container, validator))
	router.Get("/medicament/id/{cis}", handlers.FindMedicamentByID(container))
	router.Get("/generiques/{libelle}", handlers.FindGeneriques(container, validator))
	router.Get("/generiques/group/{groupId}", handlers.FindGeneriquesByGroupID(container))
	router.Get("/health", handlers.HealthCheck(container))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/medicament/id/500", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// Benchmark concurrent requests
func BenchmarkConcurrentRequests(b *testing.B) {
	container := createBenchmarkData()
	handler := handlers.FindMedicamentByID(container)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/medicament/id/500", nil)
			w := httptest.NewRecorder()
			handler(w, req)
		}
	})
}

// Memory allocation benchmark
func BenchmarkMemoryUsage(b *testing.B) {
	container := createBenchmarkData()
	handler := handlers.ServeAllMedicaments(container)

	b.ResetTimer()
	b.ReportAllocs()

	var responses [][]byte
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/database", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		responses = append(responses, w.Body.Bytes())
	}

	// Prevent compiler optimization
	_ = responses
}

// Benchmark with realistic response sizes
func BenchmarkRealisticResponse(b *testing.B) {
	container := createBenchmarkData()
	handler := handlers.NewHTTPHandler(container, validation.NewDataValidator())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/database/1", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		// Create chi router context to properly extract URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("pageNumber", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.ServePagedMedicaments(w, req)

		// Simulate response processing time
		_ = w.Body.Len()
	}
}
