package logging

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

// TestLoggingMiddlewareSkipsHealthCheck verifies that /health and /metrics endpoints are not logged
func TestLoggingMiddlewareSkipsHealthCheck(t *testing.T) {
	t.Helper()

	// Create a logger that captures log output
	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create the middleware
	mw := LoggingMiddleware(logger)

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Wrap the handler
	handler := mw(nextHandler)

	// Test 1: /health should NOT be logged
	t.Run("/health is not logged", func(t *testing.T) {
		logOutput.Reset()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-123"))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Verify response
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}

		// Verify NO log output for /health
		logs := logOutput.String()
		if logs != "" {
			t.Errorf("expected no logs for /health, got: %s", logs)
		}
	})

	// Test 2: /metrics should NOT be logged
	t.Run("/metrics is not logged", func(t *testing.T) {
		logOutput.Reset()
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-456"))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Verify response
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}

		// Verify NO log output for /metrics
		logs := logOutput.String()
		if logs != "" {
			t.Errorf("expected no logs for /metrics, got: %s", logs)
		}
	})

	// Test 3: Regular paths ARE logged
	t.Run("regular paths are logged", func(t *testing.T) {
		logOutput.Reset()
		req := httptest.NewRequest(http.MethodGet, "/v1/medicaments", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-789"))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Verify response
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}

		// Verify log output for regular path
		logs := logOutput.String()
		if logs == "" {
			t.Errorf("expected logs for regular path, got empty output")
		}

		// Verify log contains expected fields
		if !strings.Contains(logs, "HTTP request") {
			t.Errorf("log should contain 'HTTP request', got: %s", logs)
		}
		if !strings.Contains(logs, "/v1/medicaments") {
			t.Errorf("log should contain path, got: %s", logs)
		}
	})

	// Test 4: Type-safe request ID
	t.Run("type-safe request ID", func(t *testing.T) {
		logOutput.Reset()
		// Test with non-string request ID (edge case)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// Pass integer instead of string
		ctx := context.WithValue(req.Context(), middleware.RequestIDKey, 12345)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Verify it falls back to "unknown"
		logs := logOutput.String()
		if logs == "" {
			t.Errorf("expected logs for /test, got empty output")
		}
		if !strings.Contains(logs, "request_id=unknown") {
			t.Errorf("log should contain request_id=unknown for non-string ID, got: %s", logs)
		}
	})

	// Test 5: Query params only added when present
	t.Run("query params conditional logging", func(t *testing.T) {
		// Test with no query params
		t.Run("no query params", func(t *testing.T) {
			logOutput.Reset()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-1"))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			logs := logOutput.String()
			// Should NOT contain query field when empty
			if strings.Contains(logs, "query=") {
				t.Errorf("log should not contain 'query=' field when empty, got: %s", logs)
			}
		})

		// Test with query params
		t.Run("with query params", func(t *testing.T) {
			logOutput.Reset()
			req := httptest.NewRequest(http.MethodGet, "/test?foo=bar&baz=qux", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-2"))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			logs := logOutput.String()
			// SHOULD contain query field when present
			if !strings.Contains(logs, "query=") {
				t.Errorf("log should contain 'query=' field when present, got: %s", logs)
			}
			if !strings.Contains(logs, "foo=bar") {
				t.Errorf("log should contain query value, got: %s", logs)
			}
		})
	})
}
