package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-chi/chi/v5"
)

// Create realistic test data for benchmarks
func createBenchmarkData() *data.DataContainer {
	// Create 1000 mock medicaments for realistic performance testing
	medicaments := make([]entities.Medicament, 1000)
	medicamentsMap := make(map[int]entities.Medicament)
	generiques := make([]entities.GeneriqueList, 100)
	generiquesMap := make(map[int]entities.Generique)

	for i := 0; i < 1000; i++ {
		cis := i + 1
		medicaments[i] = entities.Medicament{
			Cis:                   cis,
			Denomination:          fmt.Sprintf("Medicament %d", cis),
			FormePharmaceutique:   "Comprimé",
			VoiesAdministration:   []string{"orale"},
			StatusAutorisation:    "Autorisation active",
			TypeProcedure:         "Procédure nationale",
			EtatComercialisation:  "Commercialisée",
			DateAMM:               "2020-01-01",
			Titulaire:             "Laboratoire Test",
			SurveillanceRenforcee: "Non",
			Composition:           []entities.Composition{},
			Generiques:            []entities.Generique{},
			Presentation:          []entities.Presentation{},
			Conditions:            []string{},
		}
		medicamentsMap[cis] = medicaments[i]
	}

	for i := 0; i < 100; i++ {
		groupID := i + 1
		generiques[i] = entities.GeneriqueList{
			GroupID:     groupID,
			Libelle:     fmt.Sprintf("Groupe Générique %d", groupID),
			Medicaments: []entities.GeneriqueMedicament{},
		}
		generiquesMap[groupID] = entities.Generique{
			Cis:     1,
			Group:   groupID,
			Libelle: fmt.Sprintf("Groupe Générique %d", groupID),
			Type:    "Générique",
		}
	}

	container := data.NewDataContainer()
	container.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap)
	return container
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
