package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-chi/chi/v5"
)

// BenchmarkResult contains metrics from benchmark runs, exported for documentation tests
type BenchmarkResult struct {
	Iterations   int
	Duration     time.Duration
	Throughput   float64 // req/sec
	Latency      float64 // microseconds
	MemoryBefore runtime.MemStats
	MemoryAfter  runtime.MemStats
	AllocsPerOp  uint64
	BytesPerOp   uint64
}

var (
	// Algorithmic test data (smaller dataset for fast iteration)
	algorithmicContainer *data.DataContainer
	algorithmicDataOnce  sync.Once

	// Real-world test data (full dataset)
	realWorldContainer *data.DataContainer
	realWorldDataOnce  sync.Once

	// Real HTTP server and client for real-world benchmarks
	realWorldServer     *httptest.Server
	realWorldClient     *http.Client
	realWorldServerOnce sync.Once
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// setupAlgorithmicContainer creates a data container with a smaller dataset
// for fast iteration during algorithmic benchmarks (~100-500 items)
func setupAlgorithmicContainer() *data.DataContainer {
	algorithmicDataOnce.Do(func() {
		fmt.Println("Loading algorithmic test data (smaller subset)...")

		// Parse full dataset first
		medicaments, presentationsCIP7Map, presentationsCIP13Map, err := medicamentsparser.ParseAllMedicaments()
		if err != nil {
			panic(fmt.Sprintf("Failed to parse medicaments: %v", err))
		}

		// Take only a subset for algorithmic tests
		subsetSize := 500
		if len(medicaments) < subsetSize {
			subsetSize = len(medicaments)
		}
		algorithmicMedicaments := medicaments[:subsetSize]

		// Create medicaments map from subset
		medicamentsMap := make(map[int]entities.Medicament)
		for i := range algorithmicMedicaments {
			medicamentsMap[algorithmicMedicaments[i].Cis] = algorithmicMedicaments[i]
		}

		// Parse generiques with subset
		generiques, generiquesMap, err := medicamentsparser.GeneriquesParser(&algorithmicMedicaments, &medicamentsMap)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse generiques: %v", err))
		}

		algorithmicContainer = data.NewDataContainer()
		algorithmicContainer.UpdateData(algorithmicMedicaments, generiques, medicamentsMap, generiquesMap,
			presentationsCIP7Map, presentationsCIP13Map, &interfaces.DataQualityReport{
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

		fmt.Printf("Algorithmic test data loaded: %d medicaments, %d generiques\n",
			len(algorithmicMedicaments), len(generiques))
	})

	return algorithmicContainer
}

// setupRealWorldData creates a data container with the full dataset
// for real-world performance benchmarks (~15K+ items)
func setupRealWorldData() *data.DataContainer {
	realWorldDataOnce.Do(func() {
		fmt.Println("Loading real-world test data (full dataset)...")

		// Parse full dataset
		medicaments, presentationsCIP7Map, presentationsCIP13Map, err := medicamentsparser.ParseAllMedicaments()
		if err != nil {
			panic(fmt.Sprintf("Failed to parse medicaments: %v", err))
		}

		// Create medicaments map from full dataset
		medicamentsMap := make(map[int]entities.Medicament)
		for i := range medicaments {
			medicamentsMap[medicaments[i].Cis] = medicaments[i]
		}

		// Parse generiques with full dataset
		generiques, generiquesMap, err := medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse generiques: %v", err))
		}

		realWorldContainer = data.NewDataContainer()
		realWorldContainer.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
			presentationsCIP7Map, presentationsCIP13Map, &interfaces.DataQualityReport{
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

		fmt.Printf("Real-world test data loaded: %d medicaments, %d generiques\n",
			len(medicaments), len(generiques))
	})

	return realWorldContainer
}

// setupRealWorldClient creates an HTTP client with connection pooling
// for realistic benchmark performance
func setupRealWorldClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// setupRealWorldServer creates a real HTTP server with the full dataset
// for HTTP-level performance benchmarks
func setupRealWorldServer() (*httptest.Server, *http.Client) {
	realWorldServerOnce.Do(func() {
		fmt.Println("Setting up real-world HTTP test server...")

		container := setupRealWorldData()
		validator := validation.NewDataValidator()
		httpHandler := handlers.NewHTTPHandler(container, validator)

		// Create router with v1 routes (similar to documentation_claims_verification_test.go)
		router := chi.NewRouter()
		router.Get("/v1/medicaments", httpHandler.ServeMedicamentsV1)
		router.Get("/v1/medicaments/{cis}", httpHandler.FindMedicamentByCIS)
		router.Get("/v1/generiques", httpHandler.ServeGeneriquesV1)
		router.Get("/v1/presentations/{cip}", httpHandler.ServePresentationsV1)
		router.Get("/health", httpHandler.HealthCheck)

		realWorldServer = httptest.NewServer(router)
		realWorldClient = setupRealWorldClient()

		fmt.Printf("Real-world test server ready at %s\n", realWorldServer.URL)
	})

	return realWorldServer, realWorldClient
}

// BenchmarkAlgorithmicPerformance benchmarks algorithmic operations at the handler level
// using httptest with a smaller dataset (~500 items) for fast iteration.
// Usage: go test -bench=BenchmarkAlgorithmicPerformance -benchmem
func BenchmarkAlgorithmicPerformance(b *testing.B) {
	// Initialize with production environment for optimal performance (WARN/ERROR to console only)
	logging.ResetForBenchmark(b, "", config.EnvProduction, "", 4, 100*1024*1024)

	container := setupAlgorithmicContainer()
	validator := validation.NewDataValidator()
	httpHandler := handlers.NewHTTPHandler(container, validator)

	b.Run("CISLookup", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			req := httptest.NewRequest("GET", "/v1/medicaments?cis=500", nil)
			w := httptest.NewRecorder()
			httpHandler.ServeMedicamentsV1(w, req)
		}
	})

	b.Run("GenericGroupLookup", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			req := httptest.NewRequest("GET", "/v1/generiques?group=50", nil)
			w := httptest.NewRecorder()
			httpHandler.ServeGeneriquesV1(w, req)
		}
	})

	b.Run("Pagination", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			req := httptest.NewRequest("GET", "/v1/medicaments?page=1", nil)
			w := httptest.NewRecorder()
			httpHandler.ServeMedicamentsV1(w, req)
		}
	})

	b.Run("Search", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			req := httptest.NewRequest("GET", "/v1/medicaments?search=hexaspray", nil)
			w := httptest.NewRecorder()
			httpHandler.ServeMedicamentsV1(w, req)
		}
	})

	b.Run("PresentationsLookup", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			req := httptest.NewRequest("GET", "/v1/presentations/1234567", nil)
			w := httptest.NewRecorder()
			httpHandler.ServePresentationsV1(w, req)
		}
	})
}

// BenchmarkHTTPPerformance benchmarks HTTP-level operations with real HTTP server
// and full dataset (~15K+ items) to verify documentation claims.
// Usage: go test -bench=BenchmarkHTTPPerformance -benchmem
func BenchmarkHTTPPerformance(b *testing.B) {
	// Initialize with production environment for optimal performance (WARN/ERROR to console only)
	logging.ResetForBenchmark(b, "", config.EnvProduction, "", 4, 100*1024*1024)

	server, client := setupRealWorldServer()
	// Don't close server here as it's shared across benchmarks

	// Warm up connection
	for range 10 {
		resp, err := client.Get(server.URL + "/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}

	b.Run("CISLookup", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments/61266250")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("GenericGroupLookup", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/generiques?group=50")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("Pagination", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?page=1")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("Search", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?search=hexaspray")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("GenericSearch", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/generiques?libelle=Paracetamol")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("PresentationsLookup", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/presentations/1234567")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("CIPLookup", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?cip=1234567")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/health")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})
}

// BenchmarkRealWorldSearch benchmarks realistic search patterns with full dataset
// to verify search performance under real-world conditions.
// Usage: go test -bench=BenchmarkRealWorldSearch -benchmem
func BenchmarkRealWorldSearch(b *testing.B) {
	// Initialize with production environment for optimal performance (WARN/ERROR to console only)
	logging.ResetForBenchmark(b, "", config.EnvProduction, "", 4, 100*1024*1024)

	server, client := setupRealWorldServer()
	// Don't close server here as it's shared across benchmarks

	b.Run("CommonMedication", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?search=paracetamol")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("BrandName", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?search=Doliprane")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("Antibiotic", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?search=amoxicilline")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("WithAccents", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?search=ibuprofène")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("ShortQuery", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?search=abc")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("NonExistent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?search=xyz123abc")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})
}

// BenchmarkSustainedPerformance benchmarks performance under sustained load
// with concurrent users and mixed endpoint patterns.
// Usage: go test -bench=BenchmarkSustainedPerformance -benchmem
func BenchmarkSustainedPerformance(b *testing.B) {
	// Initialize with production environment for optimal performance (WARN/ERROR to console only)
	logging.ResetForBenchmark(b, "", config.EnvProduction, "", 4, 100*1024*1024)

	server, client := setupRealWorldServer()
	// Don't close server here as it's shared across benchmarks

	b.Run("ConcurrentLoad", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			endpoints := []string{
				"/v1/medicaments?page=1",
				"/v1/medicaments?page=2",
				"/health",
			}
			for pb.Next() {
				endpoint := endpoints[runtime.NumGoroutine()%len(endpoints)]
				resp, err := client.Get(server.URL + endpoint)
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		})
	})

	b.Run("MixedEndpoints", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		endpoints := []string{
			"/v1/medicaments/61266250",
			"/v1/medicaments?page=1",
			"/v1/medicaments?search=paracetamol",
			"/v1/generiques?group=50",
			"/health",
		}
		for b.Loop() {
			endpoint := endpoints[runtime.NumGoroutine()%len(endpoints)]
			resp, err := client.Get(server.URL + endpoint)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	b.Run("MemoryUnderLoad", func(b *testing.B) {
		// Get initial memory stats
		var initialMem runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&initialMem)

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			resp, err := client.Get(server.URL + "/v1/medicaments?page=1")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		// Get final memory stats
		var finalMem runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&finalMem)

		memGrowthMB := int64(finalMem.Sys-initialMem.Sys) / 1024 / 1024
		b.ReportMetric(float64(memGrowthMB), "MB_growth")
	})
}

// RunAlgorithmicBenchmark runs algorithmic benchmarks with handler-level testing
// and returns result for documentation verification.
// Exported for use by documentation_claims_verification_test.go
func RunAlgorithmicBenchmark(endpoint string, iterations int) BenchmarkResult {
	container := setupAlgorithmicContainer()
	validator := validation.NewDataValidator()
	httpHandler := handlers.NewHTTPHandler(container, validator)

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	successCount := 0

	for range iterations {
		req := httptest.NewRequest("GET", endpoint, nil)
		w := httptest.NewRecorder()

		switch {
		case strings.Contains(endpoint, "/v1/medicaments"):
			httpHandler.ServeMedicamentsV1(w, req)
		case strings.Contains(endpoint, "/v1/generiques"):
			httpHandler.ServeGeneriquesV1(w, req)
		case strings.Contains(endpoint, "/v1/presentations"):
			httpHandler.ServePresentationsV1(w, req)
		case endpoint == "/health":
			httpHandler.HealthCheck(w, req)
		}

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	duration := time.Since(start)
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	return BenchmarkResult{
		Iterations:   iterations,
		Duration:     duration,
		Throughput:   float64(successCount) / duration.Seconds(),
		Latency:      float64(duration.Nanoseconds() / int64(iterations) / 1000), // microseconds
		MemoryBefore: memBefore,
		MemoryAfter:  memAfter,
	}
}

// RunHTTPBenchmark runs HTTP benchmarks with real HTTP server
// and returns the result for documentation verification.
// Exported for use by documentation_claims_verification_test.go
func RunHTTPBenchmark(endpoint string, duration time.Duration) BenchmarkResult {
	server, client := setupRealWorldServer()

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	successCount := 0
	requestCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			goto done
		default:
			resp, err := client.Get(server.URL + endpoint)
			requestCount++
			if err == nil && resp.StatusCode == http.StatusOK {
				successCount++
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		}
	}

done:
	elapsed := time.Since(start)
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	return BenchmarkResult{
		Iterations:   requestCount,
		Duration:     elapsed,
		Throughput:   float64(successCount) / elapsed.Seconds(),
		Latency:      float64(elapsed.Nanoseconds() / int64(requestCount) / 1000), // microseconds
		MemoryBefore: memBefore,
		MemoryAfter:  memAfter,
	}
}

// RunMemoryBenchmark runs memory benchmarks and returns memory metrics.
// Exported for use by documentation_claims_verification_test.go
func RunMemoryBenchmark(benchmarkName string) BenchmarkResult {
	container := setupRealWorldData()

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Run some operations based on benchmark name
	switch benchmarkName {
	case "DataContainerInit":
		// Already loaded above, just capture the state
	case "UnderLoad":
		// Simulate load by accessing data
		medicaments := container.GetMedicaments()
		limit := min(1000, len(medicaments))
		for i := range limit {
			_ = medicaments[i]
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	allocMB := float64(memAfter.Alloc) / 1024 / 1024
	sysMB := float64(memAfter.Sys) / 1024 / 1024

	return BenchmarkResult{
		Iterations:   1,
		Duration:     time.Duration(memAfter.Sys - memBefore.Sys),
		MemoryBefore: memBefore,
		MemoryAfter:  memAfter,
		Latency:      allocMB, // Reuse field for alloc MB
		Throughput:   sysMB,   // Reuse field for sys MB
	}
}
