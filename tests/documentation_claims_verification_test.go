package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/health"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/server"
	"github.com/giygas/medicaments-api/validation"
)

var (
	testOnce      sync.Once
	testContainer *data.DataContainer
)

// Comprehensive test to verify all documentation performance claims
type PerformanceClaim struct {
	Description   string
	ClaimedValue  float64
	MeasuredValue float64
	Unit          string
	ClaimType     string // "latency", "throughput", "memory", "coverage"
	Passed        bool
	Tolerance     float64 // Acceptable tolerance percentage
}

var verificationResults []PerformanceClaim

func TestDocumentationClaimsVerification(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping performance verification in CI environment")
	}

	if testing.Short() {
		t.Skip("Skipping performance verification in short mode")
	}

	fmt.Println("=== COMPREHENSIVE DOCUMENTATION CLAIMS VERIFICATION ===")

	logging.ResetForTest(t, "", config.EnvProduction, "", 4, 100*1024*1024)

	container := createFullTestData()
	validator := validation.NewDataValidator()

	testAlgorithmicPerformance(t, container, validator)
	testHTTPPerformance(t, container, validator)
	testMemoryUsage(t, container)
	testParsingPerformance(t)

	generateVerificationReport(t)
}

func testAlgorithmicPerformance(t *testing.T, container *data.DataContainer, validator interfaces.DataValidator) {
	fmt.Println("\n--- ALGORITHMIC PERFORMANCE VERIFICATION ---")

	healthChecker := health.NewHealthChecker(container)
	httpHandler := handlers.NewHTTPHandler(container, validator, healthChecker)

	claims := []struct {
		name       string
		handler    http.HandlerFunc
		setupReq   func() *http.Request
		claimedReq float64
		claimedLat float64
		tolerance  float64
	}{
		{
			name:    "/v1/medicaments/{cis}",
			handler: httpHandler.FindMedicamentByCIS,
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/v1/medicaments/61266250", nil)
				return req
			},
			claimedReq: 400000,
			claimedLat: 3.0,
			tolerance:  20.0,
		},
		{
			name:    "/v1/generiques/{groupID}",
			handler: httpHandler.ServeGeneriquesV1,
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/v1/generiques?group=50", nil)
				return req
			},
			claimedReq: 200000,
			claimedLat: 5.0,
			tolerance:  20.0,
		},
		{
			name:    "/v1/medicaments?page={n}",
			handler: httpHandler.ServeMedicamentsV1,
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/v1/medicaments?page=1", nil)
				return req
			},
			claimedReq: 40000,
			claimedLat: 30.0,
			tolerance:  20.0,
		},
		{
			name:    "/v1/medicaments?search={query}",
			handler: httpHandler.ServeMedicamentsV1,
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/v1/medicaments?search=Medicament", nil)
				return req
			},
			claimedReq: 1600,
			claimedLat: 750.0,
			tolerance:  30.0,
		},
		{
			name:    "/v1/generiques?libelle={nom}",
			handler: httpHandler.ServeGeneriquesV1,
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/v1/generiques?libelle=Paracetamol", nil)
				return req
			},
			claimedReq: 18000,
			claimedLat: 60.0,
			tolerance:  30.0,
		},
		{
			name:    "/v1/presentations?cip={code}",
			handler: httpHandler.ServePresentationsV1,
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/v1/presentations/1234567", nil)
				return req
			},
			claimedReq: 430000,
			claimedLat: 2.0,
			tolerance:  20.0,
		},
		{
			name:    "/v1/medicaments?cip={code}",
			handler: httpHandler.ServeMedicamentsV1,
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/v1/medicaments?cip=1234567", nil)
				return req
			},
			claimedReq: 375000,
			claimedLat: 5.0,
			tolerance:  20.0,
		},
		{
			name:    "/health",
			handler: httpHandler.HealthCheck,
			setupReq: func() *http.Request {
				return httptest.NewRequest("GET", "/health", nil)
			},
			claimedReq: 400000,
			claimedLat: 3.0,
			tolerance:  20.0,
		},
	}

	for _, claim := range claims {
		t.Run(claim.name+" algorithmic", func(t *testing.T) {
			// Benchmark for throughput
			result := testing.Benchmark(func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for b.Loop() {
					req := claim.setupReq()
					w := httptest.NewRecorder()
					claim.handler(w, req)
				}
			})

			measuredReq := float64(result.N) / result.T.Seconds()
			measuredLat := result.T.Nanoseconds() / int64(result.N) / 1000 // microseconds

			// Verify throughput claim
			reqPassed := measuredReq >= claim.claimedReq*(1-claim.tolerance/100)
			verificationResults = append(verificationResults, PerformanceClaim{
				Description:   fmt.Sprintf("%s algorithmic throughput", claim.name),
				ClaimedValue:  claim.claimedReq,
				MeasuredValue: measuredReq,
				Unit:          "req/sec",
				ClaimType:     "throughput",
				Passed:        reqPassed,
				Tolerance:     claim.tolerance,
			})

			// Verify latency claim
			latPassed := float64(measuredLat) <= claim.claimedLat*(1+claim.tolerance/100)
			verificationResults = append(verificationResults, PerformanceClaim{
				Description:   fmt.Sprintf("%s algorithmic latency", claim.name),
				ClaimedValue:  claim.claimedLat,
				MeasuredValue: float64(measuredLat),
				Unit:          "µs",
				ClaimType:     "latency",
				Passed:        latPassed,
				Tolerance:     claim.tolerance,
			})

			fmt.Printf("  %s: %.0f req/sec (claimed: %.0f), %.1fµs (claimed: %.1f)\n",
				claim.name, measuredReq, claim.claimedReq, float64(measuredLat), claim.claimedLat)
		})
	}
}

func testHTTPPerformance(t *testing.T, _ *data.DataContainer, _ interfaces.DataValidator) {
	fmt.Println("\n--- HTTP PERFORMANCE VERIFICATION ---")

	srv, baseURL := setupTestServer(t)
	defer func() {
		if err := srv.Shutdown(context.Background()); err != nil {
			t.Logf("Server shutdown error: %v", err)
		}
	}()

	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{Transport: transport}

	for range 100 {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}

	claims := []struct {
		name       string
		endpoint   string
		claimedReq float64
		tolerance  float64
	}{
		{
			name:       "/v1/medicaments/{cis}",
			endpoint:   "/v1/medicaments/61266250",
			claimedReq: 78000,
			tolerance:  25.0,
		},
		{
			name:       "/v1/medicaments?page={n}",
			endpoint:   "/v1/medicaments?page=1",
			claimedReq: 41000,
			tolerance:  25.0,
		},
		{
			name:       "/v1/medicaments?search={query}",
			endpoint:   "/v1/medicaments?search=Test",
			claimedReq: 6100,
			tolerance:  25.0,
		},
		{
			name:       "/v1/generiques?libelle={nom}",
			endpoint:   "/v1/generiques?libelle=Paracetamol",
			claimedReq: 36000,
			tolerance:  25.0,
		},
		{
			name:       "/v1/presentations?cip={code}",
			endpoint:   "/v1/presentations/1234567",
			claimedReq: 77000,
			tolerance:  25.0,
		},
		{
			name:       "/v1/medicaments?cip={code}",
			endpoint:   "/v1/medicaments?cip=1234567",
			claimedReq: 75000,
			tolerance:  25.0,
		},
		{
			name:       "/health",
			endpoint:   "/health",
			claimedReq: 92000,
			tolerance:  25.0,
		},
	}

	const workers = 300
	const duration = 3 * time.Second

	for _, claim := range claims {
		t.Run(claim.name+" HTTP throughput", func(t *testing.T) {
			var successCount atomic.Int64
			var wg sync.WaitGroup
			done := make(chan struct{})

			time.AfterFunc(duration, func() { close(done) })

			for range workers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for {
						select {
						case <-done:
							return
						default:
							resp, err := client.Get(baseURL + claim.endpoint)
							if err == nil {
								_, _ = io.Copy(io.Discard, resp.Body)
								_ = resp.Body.Close()
								successCount.Add(1)
							}
						}
					}
				}()
			}

			wg.Wait()

			measuredReq := float64(successCount.Load()) / duration.Seconds()

			reqPassed := measuredReq >= claim.claimedReq*(1-claim.tolerance/100)

			verificationResults = append(verificationResults, PerformanceClaim{
				Description:   fmt.Sprintf("%s HTTP throughput", claim.name),
				ClaimedValue:  claim.claimedReq,
				MeasuredValue: measuredReq,
				Unit:          "req/sec",
				ClaimType:     "throughput",
				Passed:        reqPassed,
				Tolerance:     claim.tolerance,
			})

			fmt.Printf("  %s: %.2f req/sec (claimed: %.1f)\n",
				claim.name, measuredReq, claim.claimedReq)
		})
	}
}

func testMemoryUsage(t *testing.T, _ *data.DataContainer) {
	t.Helper()
	fmt.Println("\n--- MEMORY USAGE VERIFICATION ---")

	srv, _ := setupTestServer(t)
	defer func() {
		if err := srv.Shutdown(context.Background()); err != nil {
			t.Logf("Server shutdown error: %v", err)
		}
	}()

	time.Sleep(2 * time.Second)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	claimedMin := 60.0
	claimedMax := 90.0

	memoryPassed := allocMB >= claimedMin && allocMB <= claimedMax
	verificationResults = append(verificationResults, PerformanceClaim{
		Description:   "Application memory usage",
		ClaimedValue:  (claimedMin + claimedMax) / 2,
		MeasuredValue: allocMB,
		Unit:          "MB",
		ClaimType:     "memory",
		Passed:        memoryPassed,
		Tolerance:     0.0,
	})

	fmt.Printf("  Application memory: %.1f MB alloc, %.1f MB sys (claimed: %.1f-%.1f MB)\n",
		allocMB, sysMB, claimedMin, claimedMax)
}

func testParsingPerformance(t *testing.T) {
	fmt.Println("\n--- PARSING PERFORMANCE VERIFICATION ---")

	start := time.Now()

	// Parse full medicaments database
	medicaments, _, _, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("Failed to parse medicaments: %v", err)
	}

	// Create maps
	medicamentsMap := make(map[int]entities.Medicament)
	for i := range medicaments {
		medicamentsMap[medicaments[i].Cis] = medicaments[i]
	}

	// Parse generiques
	_, _, err = medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
	if err != nil {
		t.Fatalf("Failed to parse generiques: %v", err)
	}

	duration := time.Since(start).Seconds()
	claimedDuration := 0.7

	parsingPassed := duration <= claimedDuration*2.0 // 100% tolerance for CI environments
	verificationResults = append(verificationResults, PerformanceClaim{
		Description:   "Concurrent TSV parsing",
		ClaimedValue:  claimedDuration,
		MeasuredValue: duration,
		Unit:          "seconds",
		ClaimType:     "latency",
		Passed:        parsingPassed,
		Tolerance:     100.0,
	})

	fmt.Printf("  Parsing time: %.2f seconds (claimed: %.1f)\n", duration, claimedDuration)
}

func createFullTestData() *data.DataContainer {
	testOnce.Do(func() {
		fmt.Println("Loading full medicaments database for verification...")

		// Parse of full medicaments database
		medicaments, presentationsCIP7Map, presentationsCIP13Map, err := medicamentsparser.ParseAllMedicaments()
		if err != nil {
			panic(fmt.Sprintf("Failed to parse medicaments: %v", err))
		}

		// Create medicaments map
		medicamentsMap := make(map[int]entities.Medicament)
		for i := range medicaments {
			medicamentsMap[medicaments[i].Cis] = medicaments[i]
		}

		// Parse generiques
		generiques, generiquesMap, err := medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse generiques: %v", err))
		}

		testContainer = data.NewDataContainer()
		testContainer.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
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

		fmt.Printf("Loaded: %d medicaments, %d generiques\n", len(medicaments), len(generiques))
	})

	return testContainer
}

func setupTestServer(t *testing.T) (*server.Server, string) {
	logging.ResetForTest(t, "", config.EnvProduction, "", 4, 100*1024*1024)

	container := createFullTestData()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Logf("Failed to close listener: %v", err)
	}

	cfg := &config.Config{
		Port:           fmt.Sprintf("%d", port),
		Address:        "127.0.0.1",
		Env:            config.EnvTest,
		LogLevel:       "error",
		MaxRequestBody: 1048576,
		MaxHeaderSize:  1048576,
	}

	srv := server.NewServer(cfg, container)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	baseURL := "http://" + cfg.Address + ":" + fmt.Sprintf("%d", port)

	maxRetries := 50
	for range maxRetries {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			time.Sleep(100 * time.Millisecond)
			return srv, baseURL
		}
		time.Sleep(200 * time.Millisecond)
	}

	select {
	case err := <-serverErr:
		t.Fatalf("Server failed to start: %v", err)
	default:
		t.Fatal("Server failed to become ready after 10 seconds")
	}

	return srv, baseURL
}

func generateVerificationReport(t *testing.T) {
	fmt.Println("\n=== VERIFICATION REPORT ===")

	passed := 0
	total := len(verificationResults)

	for _, result := range verificationResults {
		status := "❌ FAIL"
		if result.Passed {
			status = "✅ PASS"
			passed++
		}

		diff := ((result.MeasuredValue - result.ClaimedValue) / result.ClaimedValue) * 100

		fmt.Printf("%s %s: %.1f %s (claimed: %.1f %s, diff: %.1f%%)\n",
			status, result.Description, result.MeasuredValue, result.Unit,
			result.ClaimedValue, result.Unit, diff)
	}

	fmt.Printf("\nSUMMARY: %d/%d claims verified (%.1f%%)\n",
		passed, total, float64(passed)/float64(total)*100)

	if passed < int(float64(total)*0.8) { // At least 80% should pass
		t.Errorf("Too many performance claims failed verification: %d/%d passed", passed, total)
	}
}
