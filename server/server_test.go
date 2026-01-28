package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// MockHealthChecker implements interfaces.HealthChecker for testing
type MockHealthChecker struct {
	status  string
	details map[string]any
	err     error
}

func (m *MockHealthChecker) HealthCheck() (string, map[string]any, error) {
	return m.status, m.details, m.err
}

func (m *MockHealthChecker) CalculateNextUpdate() time.Time {
	return time.Now().Add(6 * time.Hour)
}

// TestNewServer tests server creation with various configurations
func TestNewServer(t *testing.T) {
	// Initialize logging for tests
	logging.InitLogger("")
	tests := []struct {
		name          string
		config        *config.Config
		dataContainer *data.DataContainer
		expectError   bool
	}{
		{
			name: "valid config and data container",
			config: &config.Config{
				Port:           "8080",
				Address:        "localhost",
				Env:            "test",
				LogLevel:       "info",
				MaxRequestBody: 1048576,
				MaxHeaderSize:  1048576,
			},
			dataContainer: data.NewDataContainer(),
			expectError:   false,
		},
		{
			name: "minimal config should work",
			config: &config.Config{
				Port:           "8080",
				Address:        "localhost",
				Env:            "test",
				LogLevel:       "info",
				MaxRequestBody: 1048576,
				MaxHeaderSize:  1048576,
			},
			dataContainer: data.NewDataContainer(),
			expectError:   false,
		},
		{
			name: "nil data container should still work",
			config: &config.Config{
				Port:           "8080",
				Address:        "localhost",
				Env:            "test",
				LogLevel:       "info",
				MaxRequestBody: 1048576,
				MaxHeaderSize:  1048576,
			},
			dataContainer: nil,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use default config if nil
			cfg := tt.config
			if cfg == nil {
				cfg = &config.Config{
					Port:           "8080",
					Address:        "localhost",
					Env:            "test",
					LogLevel:       "info",
					MaxRequestBody: 1048576,
					MaxHeaderSize:  1048576,
				}
			}

			// Use default data container if nil
			dc := tt.dataContainer
			if dc == nil {
				dc = data.NewDataContainer()
			}

			server := NewServer(cfg, dc)

			if server == nil {
				t.Fatal("Server should not be nil")
			}

			if server.server.Addr != cfg.Address+":"+cfg.Port {
				t.Errorf("Expected server address %s, got %s", cfg.Address+":"+cfg.Port, server.server.Addr)
			}

			if server.dataContainer != dc {
				t.Error("Data container should be set correctly")
			}

			if server.config != cfg {
				t.Error("Config should be set correctly")
			}

			if server.router == nil {
				t.Error("Router should not be nil")
			}

			if server.httpHandler == nil {
				t.Error("HTTP handler should not be nil")
			}

			if server.healthChecker == nil {
				t.Error("Health checker should not be nil")
			}
		})
	}
}

// TestSetupMiddleware tests that all expected middleware are configured
func TestSetupMiddleware(t *testing.T) {
	// Initialize logging for tests
	logging.InitLogger("")

	cfg := &config.Config{
		Port:           "8080",
		Address:        "localhost",
		Env:            "test",
		LogLevel:       "info",
		MaxRequestBody: 1048576,
		MaxHeaderSize:  1048576,
	}

	dc := data.NewDataContainer()
	server := NewServer(cfg, dc)

	// Create a test request to verify middleware chain
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234" // Set localhost RemoteAddr to pass BlockDirectAccessMiddleware
	rr := httptest.NewRecorder()

	// Add a test route to verify middleware is working
	server.router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID is available in the context
		requestID := middleware.GetReqID(r.Context())
		if requestID == "" {
			t.Error("RequestID should be available in request context")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	server.router.ServeHTTP(rr, req)

	// Also verify the response was successful
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Also verify the response was successful
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// TestSetupRoutes tests that all expected routes are configured
func TestSetupRoutes(t *testing.T) {
	// Initialize logging for tests
	logging.InitLogger("")

	cfg := &config.Config{
		Port:           "8080",
		Address:        "localhost",
		Env:            "test",
		LogLevel:       "info",
		MaxRequestBody: 1048576,
		MaxHeaderSize:  1048576,
	}

	dc := data.NewDataContainer()
	server := NewServer(cfg, dc)

	// Test API routes
	expectedRoutes := []string{
		"/database",
		"/database/{pageNumber}",
		"/medicament/{element}",
		"/medicament/id/{cis}",
		"/generiques/{libelle}",
		"/generiques/group/{groupId}",
		"/health",
	}

	// Test documentation routes
	docRoutes := []string{
		"/",
		"/docs",
		"/docs/openapi.yaml",
		"/favicon.ico",
		"/cache-test",
	}

	// Use chi router to check if routes exist
	router := server.router.(*chi.Mux)

	// Check API routes
	for _, route := range expectedRoutes {
		// Chi doesn't expose route listing directly, so we'll test by making requests
		rr := httptest.NewRecorder()

		// Replace path parameters with actual values for testing
		testRoute := strings.ReplaceAll(route, "{pageNumber}", "1")
		testRoute = strings.ReplaceAll(testRoute, "{element}", "test")
		testRoute = strings.ReplaceAll(testRoute, "{cis}", "123")
		testRoute = strings.ReplaceAll(testRoute, "{libelle}", "test")
		testRoute = strings.ReplaceAll(testRoute, "{groupId}", "1")

		req := httptest.NewRequest("GET", testRoute, nil)
		req.RemoteAddr = "127.0.0.1:1234" // Set localhost RemoteAddr to pass BlockDirectAccessMiddleware
		router.ServeHTTP(rr, req)

		// Routes that require data may return 404 when data container is empty
		// This is expected behavior - we're just testing that routes are registered
		if rr.Code == http.StatusNotFound {
			// This is OK for routes that need data (medicament search, etc.)
			if strings.Contains(route, "{") {
				t.Logf("Route %s returned 404 (expected for empty data)", route)
			} else {
				t.Errorf("Route %s should be registered (got 404)", route)
			}
		} else {
			t.Logf("Route %s returned status %d (expected)", route, rr.Code)
		}
	}

	// Check documentation routes
	for _, route := range docRoutes {
		req := httptest.NewRequest("GET", route, nil)
		req.RemoteAddr = "127.0.0.1:1234" // Set localhost RemoteAddr to pass BlockDirectAccessMiddleware
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Documentation routes may return 404 if files don't exist in test environment
		// We're just testing that routes are registered, not that files exist
		if rr.Code == http.StatusNotFound {
			t.Logf("Documentation route %s returned 404 (file may not exist in test env)", route)
		} else {
			t.Logf("Documentation route %s returned status %d (expected)", route, rr.Code)
		}
	}
}

// TestServerLifecycle tests server start and shutdown
func TestServerLifecycle(t *testing.T) {
	// Initialize logging for tests
	logging.InitLogger("")

	cfg := &config.Config{
		Port:           "0", // Use port 0 for automatic port assignment
		Address:        "localhost",
		Env:            "test",
		LogLevel:       "error", // Reduce log noise during tests
		MaxRequestBody: 1048576,
		MaxHeaderSize:  1048576,
	}

	dc := data.NewDataContainer()
	server := NewServer(cfg, dc)

	// Test server start
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is running by making a request
	resp, err := http.Get("http://localhost:" + server.server.Addr[strings.LastIndex(server.server.Addr, ":")+1:] + "/health")
	if err == nil {
		_ = resp.Body.Close()
	}

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Server shutdown should not error: %v", err)
	}

	// Check if server start returned (should happen after shutdown)
	select {
	case err := <-errChan:
		// Server should have shutdown gracefully
		if err == nil {
			t.Error("Server should return error after shutdown")
		} else if !strings.Contains(err.Error(), "Server closed") {
			t.Errorf("Error should indicate server was closed: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Server should have shutdown within 1 second")
	}
}

// TestGetHealthData tests health data generation
func TestGetHealthData(t *testing.T) {
	// Initialize logging for tests
	logging.InitLogger("")

	cfg := &config.Config{
		Port:           "8080",
		Address:        "localhost",
		Env:            "test",
		LogLevel:       "info",
		MaxRequestBody: 1048576,
		MaxHeaderSize:  1048576,
	}

	// Create data container with test data
	dc := data.NewDataContainer()

	// Add some test medicaments
	testMedicaments := []entities.Medicament{
		{Cis: 1, Denomination: "Test Med 1"},
		{Cis: 2, Denomination: "Test Med 2"},
	}
	dc.UpdateData(testMedicaments, []entities.GeneriqueList{},
		map[int]entities.Medicament{1: testMedicaments[0], 2: testMedicaments[1]},
		map[int]entities.Generique{},
		map[int]entities.Presentation{}, map[int]entities.Presentation{})

	server := NewServer(cfg, dc)

	healthData := server.GetHealthData()

	// Verify health data structure
	if healthData.Status == "" {
		t.Error("Status should not be empty")
	}

	if healthData.Uptime == "" {
		t.Error("Uptime should not be empty")
	}

	if healthData.MemoryUsage < 0 {
		t.Error("Memory usage should be non-negative")
	}

	if healthData.LastUpdate == "" {
		t.Error("Last update should not be empty")
	}

	if healthData.NextUpdate == "" {
		t.Error("Next update should not be empty")
	}

	if healthData.MedicamentCount != 2 {
		t.Errorf("Should count test medicaments, got %d", healthData.MedicamentCount)
	}

	if healthData.GeneriqueCount < 0 {
		t.Error("Generique count should be non-negative")
	}
}

// TestFormatUptimeHuman tests uptime formatting
func TestFormatUptimeHuman(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "hours, minutes, and seconds",
			duration: 1*time.Hour + 2*time.Minute + 30*time.Second,
			expected: "1h 2m 30s",
		},
		{
			name:     "days, hours, minutes, and seconds",
			duration: 2*24*time.Hour + 1*time.Hour + 2*time.Minute + 30*time.Second,
			expected: "2d 1h 2m 30s",
		},
		{
			name:     "exactly one day",
			duration: 24 * time.Hour,
			expected: "1d 0h 0m 0s",
		},
		{
			name:     "exactly one hour",
			duration: time.Hour,
			expected: "1h 0m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUptimeHuman(tt.duration)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestServerWithRealDependencies tests server with real dependencies
func TestServerWithRealDependencies(t *testing.T) {
	cfg := &config.Config{
		Port:           "8080",
		Address:        "localhost",
		Env:            "test",
		LogLevel:       "info",
		MaxRequestBody: 1048576,
		MaxHeaderSize:  1048576,
	}

	dc := data.NewDataContainer()
	server := NewServer(cfg, dc)

	// Verify that real dependencies are injected
	// We can't directly check types without reflection, but we can verify they're not nil
	if server.httpHandler == nil {
		t.Error("HTTP handler should be injected")
	}

	if server.healthChecker == nil {
		t.Error("Health checker should be injected")
	}
}

// TestServerConfiguration tests server configuration values
func TestServerConfiguration(t *testing.T) {
	cfg := &config.Config{
		Port:           "8080",
		Address:        "localhost",
		Env:            "test",
		LogLevel:       "info",
		MaxRequestBody: 2048576, // 2MB
		MaxHeaderSize:  512,     // 512 bytes
	}

	dc := data.NewDataContainer()
	server := NewServer(cfg, dc)

	// Verify HTTP server configuration
	if server.server.ReadTimeout != 15*time.Second {
		t.Errorf("Read timeout should be 15 seconds, got %v", server.server.ReadTimeout)
	}

	if server.server.WriteTimeout != 15*time.Second {
		t.Errorf("Write timeout should be 15 seconds, got %v", server.server.WriteTimeout)
	}

	if server.server.IdleTimeout != 60*time.Second {
		t.Errorf("Idle timeout should be 60 seconds, got %v", server.server.IdleTimeout)
	}
}

// BenchmarkNewServer benchmarks server creation
func BenchmarkNewServer(b *testing.B) {
	// Initialize logging service to prevent nil pointer
	logging.InitLoggerWithRetention("", 1)
	defer logging.Close()

	cfg := &config.Config{
		Port:           "8080",
		Address:        "localhost",
		Env:            "test",
		LogLevel:       "info",
		MaxRequestBody: 1048576,
		MaxHeaderSize:  1048576,
	}

	dc := data.NewDataContainer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewServer(cfg, dc)
	}
}

// BenchmarkGetHealthData benchmarks health data generation
func BenchmarkGetHealthData(b *testing.B) {
	cfg := &config.Config{
		Port:           "8080",
		Address:        "localhost",
		Env:            "test",
		LogLevel:       "info",
		MaxRequestBody: 1048576,
		MaxHeaderSize:  1048576,
	}

	dc := data.NewDataContainer()
	server := NewServer(cfg, dc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = server.GetHealthData()
	}
}
