package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
	"github.com/go-chi/chi/v5"
)

var (
	realworldServer *httptest.Server
	realworldOnce   sync.Once
)

// Setup real-world test server with full dataset
func setupRealworldServer() *httptest.Server {
	realworldOnce.Do(func() {
		fmt.Println("Setting up real-world performance test server...")

		// Load full dataset
		medicaments, err := medicamentsparser.ParseAllMedicaments()
		if err != nil {
			panic(fmt.Sprintf("Failed to parse medicaments: %v", err))
		}

		medicamentsMap := make(map[int]entities.Medicament)
		for i := range medicaments {
			medicamentsMap[medicaments[i].Cis] = medicaments[i]
		}

		generiques, generiquesMap, err := medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse generiques: %v", err))
		}

		container := data.NewDataContainer()
		container.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap)

		// Create router with all routes
		router := chi.NewRouter()
		validator := validation.NewDataValidator()

		router.Get("/database", handlers.ServeAllMedicaments(container))
		router.Get("/database/{pageNumber}", handlers.ServePagedMedicaments(container))
		router.Get("/medicament/{element}", handlers.FindMedicament(container, validator))
		router.Get("/medicament/id/{cis}", handlers.FindMedicamentByID(container))
		router.Get("/generiques/{libelle}", handlers.FindGeneriques(container, validator))
		router.Get("/generiques/group/{groupId}", handlers.FindGeneriquesByGroupID(container))
		router.Get("/health", handlers.HealthCheck(container))

		realworldServer = httptest.NewServer(router)
		fmt.Printf("Real-world test server ready with %d medicaments, %d generiques\n", len(medicaments), len(generiques))
	})

	return realworldServer
}

// Test realistic concurrent user load
func TestRealWorldConcurrentLoad(t *testing.T) {
	server := setupRealworldServer()
	defer server.Close()

	// Simulate realistic user patterns
	endpoints := []string{
		"/database/1",
		"/database/2",
		"/medicament/paracetamol",
		"/medicament/ibuprofène",
		"/medicament/id/1000",
		"/medicament/id/2000",
		"/generiques/paracetamol",
		"/generiques/group/100",
		"/health",
	}

	// Simulate 50 concurrent users
	numUsers := 50
	requestsPerUser := 20

	var wg sync.WaitGroup
	startTime := time.Now()

	// Channel to collect response times
	responseTimes := make(chan time.Duration, numUsers*requestsPerUser)

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < requestsPerUser; j++ {
				endpoint := endpoints[j%len(endpoints)]

				reqStart := time.Now()
				resp, err := http.Get(server.URL + endpoint)
				reqTime := time.Since(reqStart)

				if err != nil {
					t.Errorf("User %d request failed: %v", userID, err)
					continue
				}

				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					t.Errorf("User %d got status %d for %s", userID, resp.StatusCode, endpoint)
				}

				resp.Body.Close()
				responseTimes <- reqTime
			}
		}(i)
	}

	wg.Wait()
	close(responseTimes)

	totalTime := time.Since(startTime)
	totalRequests := numUsers * requestsPerUser

	// Calculate statistics
	var totalResponseTime time.Duration
	var maxResponseTime time.Duration
	count := 0

	for rt := range responseTimes {
		totalResponseTime += rt
		if rt > maxResponseTime {
			maxResponseTime = rt
		}
		count++
	}

	avgResponseTime := totalResponseTime / time.Duration(count)
	requestsPerSecond := float64(totalRequests) / totalTime.Seconds()

	t.Logf("Real-world concurrent load test results:")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Concurrent users: %d", numUsers)
	t.Logf("  Total time: %v", totalTime)
	t.Logf("  Requests per second: %.2f", requestsPerSecond)
	t.Logf("  Average response time: %v", avgResponseTime)
	t.Logf("  Max response time: %v", maxResponseTime)

	// Performance assertions
	if requestsPerSecond < 100 {
		t.Errorf("Low throughput: %.2f req/s (expected > 100)", requestsPerSecond)
	}

	if avgResponseTime > 100*time.Millisecond {
		t.Errorf("High average response time: %v (expected < 100ms)", avgResponseTime)
	}

	if maxResponseTime > 1*time.Second {
		t.Errorf("Very high max response time: %v (expected < 1s)", maxResponseTime)
	}
}

// Test memory usage under load
func TestRealWorldMemoryUnderLoad(t *testing.T) {
	server := setupRealworldServer()
	defer server.Close()

	// Get initial memory stats
	var initialMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialMem)

	// Simulate sustained load
	numRequests := 1000
	numWorkers := 20

	var wg sync.WaitGroup
	requestChan := make(chan int, numRequests)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestChan {
				resp, err := http.Get(server.URL + "/database/1")
				if err == nil {
					io.Copy(io.Discard, resp.Body) // Read full response
					resp.Body.Close()
				}
			}
		}()
	}

	// Send requests
	for i := 0; i < numRequests; i++ {
		requestChan <- i
	}
	close(requestChan)

	wg.Wait()

	// Get final memory stats
	var finalMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&finalMem)

	// Calculate memory growth
	memGrowthMB := (finalMem.Alloc - initialMem.Alloc) / 1024 / 1024
	totalAllocsMB := (finalMem.TotalAlloc - initialMem.TotalAlloc) / 1024 / 1024

	t.Logf("Memory usage under load:")
	t.Logf("  Memory growth: %d MB", memGrowthMB)
	t.Logf("  Total allocations: %d MB", totalAllocsMB)
	t.Logf("  GC cycles: %d", finalMem.NumGC-initialMem.NumGC)

	// Memory assertions
	if memGrowthMB > 100 {
		t.Errorf("Excessive memory growth: %d MB (expected < 100 MB)", memGrowthMB)
	}
}

// Test response size and compression
func TestRealWorldResponseSizes(t *testing.T) {
	server := setupRealworldServer()
	defer server.Close()

	testCases := []struct {
		endpoint        string
		expectedMaxSize int // bytes
		description     string
	}{
		{"/database", 25 * 1024 * 1024, "Full database (should be large)"},
		{"/database/1", 100 * 1024, "First page (should be small)"},
		{"/medicament/id/1000", 10 * 1024, "Single medicament"},
		{"/health", 5 * 1024, "Health check"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			resp, err := http.Get(server.URL + tc.endpoint)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			t.Logf("Response size for %s: %d bytes", tc.endpoint, len(body))

			// Verify it's valid JSON
			var js json.RawMessage
			if err := json.Unmarshal(body, &js); err != nil {
				t.Errorf("Invalid JSON response: %v", err)
			}

			// Size check (only for endpoints that should be small)
			if tc.expectedMaxSize < 1024*1024 && len(body) > tc.expectedMaxSize {
				t.Errorf("Response too large: %d bytes (expected < %d)", len(body), tc.expectedMaxSize)
			}
		})
	}
}

// Test sustained performance over time
func TestRealWorldSustainedPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained performance test in short mode")
	}

	server := setupRealworldServer()
	defer server.Close()

	duration := 30 * time.Second
	requestInterval := 10 * time.Millisecond

	var (
		totalRequests     int64
		failedRequests    int64
		totalResponseTime time.Duration
		maxResponseTime   time.Duration
		mutex             sync.Mutex
	)

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	// Single goroutine making requests continuously
	go func() {
		ticker := time.NewTicker(requestInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				start := time.Now()
				resp, err := http.Get(server.URL + "/database/1")
				reqTime := time.Since(start)

				mutex.Lock()
				totalRequests++
				totalResponseTime += reqTime
				if reqTime > maxResponseTime {
					maxResponseTime = reqTime
				}
				if err != nil || resp.StatusCode >= 400 {
					failedRequests++
				}
				mutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}
	}()

	<-ctx.Done()

	// Calculate final statistics
	mutex.Lock()
	successRate := float64(totalRequests-failedRequests) / float64(totalRequests) * 100
	avgResponseTime := totalResponseTime / time.Duration(totalRequests)
	requestsPerSecond := float64(totalRequests) / duration.Seconds()
	mutex.Unlock()

	t.Logf("Sustained performance test (%v):", duration)
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Failed requests: %d (%.1f%%)", failedRequests, 100-successRate)
	t.Logf("  Success rate: %.1f%%", successRate)
	t.Logf("  Requests per second: %.2f", requestsPerSecond)
	t.Logf("  Average response time: %v", avgResponseTime)
	t.Logf("  Max response time: %v", maxResponseTime)

	// Performance assertions
	if successRate < 99 {
		t.Errorf("Low success rate: %.1f%% (expected > 99%%)", successRate)
	}

	if avgResponseTime > 50*time.Millisecond {
		t.Errorf("High average response time: %v (expected < 50ms)", avgResponseTime)
	}

	if requestsPerSecond < 50 {
		t.Errorf("Low sustained throughput: %.2f req/s (expected > 50)", requestsPerSecond)
	}
}

// Test different search patterns
func TestRealWorldSearchPatterns(t *testing.T) {
	server := setupRealworldServer()
	defer server.Close()

	searchPatterns := []struct {
		pattern         string
		expectedResults int // approximate expected results
		description     string
	}{
		{"paracetamol", 100, "Common medication"},
		{"ibuprofène", 50, "Common medication with accent"},
		{"amoxicilline", 30, "Antibiotic"},
		{"aspirine", 40, "Very common medication"},
		{"doliprane", 20, "Brand name"},
		{"xyz123", 0, "Non-existent medication"},
		{"a", 1000, "Single character (many results)"},
	}

	for _, sp := range searchPatterns {
		t.Run(sp.description, func(t *testing.T) {
			start := time.Now()
			resp, err := http.Get(server.URL + "/medicament/" + sp.pattern)
			reqTime := time.Since(start)

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			var results []entities.Medicament
			if err := json.Unmarshal(body, &results); err != nil {
				t.Errorf("Invalid JSON response: %v", err)
				return
			}

			t.Logf("Search '%s': %d results in %v", sp.pattern, len(results), reqTime)

			// Performance assertions
			if reqTime > 100*time.Millisecond {
				t.Errorf("Search too slow: %v (expected < 100ms)", reqTime)
			}

			// Reasonable result count check (allow some variance)
			if sp.expectedResults == 0 && len(results) > 0 {
				t.Errorf("Expected 0 results for non-existent pattern, got %d", len(results))
			} else if sp.expectedResults > 0 && len(results) == 0 {
				t.Logf("Warning: Expected some results for '%s', got 0", sp.pattern)
			}
		})
	}
}

// Benchmark real-world request patterns
func BenchmarkRealWorldRequests(b *testing.B) {
	server := setupRealworldServer()
	defer server.Close()

	endpoints := []string{
		"/database/1",
		"/medicament/id/1000",
		"/health",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		endpoint := endpoints[i%len(endpoints)]
		resp, err := http.Get(server.URL + endpoint)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
