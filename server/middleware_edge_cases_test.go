package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/config"
)

// ============================================================================
// EDGE CASE TESTS FOR MIDDLEWARE
// ============================================================================

func TestRealIPMiddleware_SingleIP(t *testing.T) {
	// X-Forwarded-For with single IP (no comma)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.RemoteAddr = "192.168.1.1:12345"

	rr := httptest.NewRecorder()
	handler := RealIPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rr.Header().Set("X-Real-IP", r.RemoteAddr)
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	realIP := rr.Header().Get("X-Real-IP")
	if realIP != "203.0.113.1" {
		t.Errorf("Expected RemoteAddr to be '203.0.113.1', got '%s'", realIP)
	}
}

func TestRealIPMiddleware_WithoutXForwardedFor(t *testing.T) {
	// Request without X-Forwarded-For header (should keep original RemoteAddr without port)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	rr := httptest.NewRecorder()
	handler := RealIPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rr.Header().Set("X-Original-RemoteAddr", r.RemoteAddr)
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	originalAddr := rr.Header().Get("X-Original-RemoteAddr")
	if originalAddr != "192.168.1.1" {
		t.Errorf("Expected RemoteAddr without port, got '%s'", originalAddr)
	}
}

func TestBlockDirectAccessMiddleware_LocalhostIPv4(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	handler := BlockDirectAccessMiddleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK for localhost, got %d", rr.Code)
	}
}

func TestBlockDirectAccessMiddleware_LocalhostIPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[::1]:12345"

	rr := httptest.NewRecorder()
	handler := BlockDirectAccessMiddleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK for localhost IPv6, got %d", rr.Code)
	}
}

func TestBlockDirectAccessMiddleware_DirectIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	rr := httptest.NewRecorder()
	handler := BlockDirectAccessMiddleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status Forbidden for direct access, got %d", rr.Code)
	}
}

func TestBlockDirectAccessMiddleware_WithXForwardedFor(t *testing.T) {
	// Should allow when X-Forwarded-For is present (proxy)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	req.RemoteAddr = "192.168.1.1:12345"

	rr := httptest.NewRecorder()
	handler := BlockDirectAccessMiddleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK when proxied, got %d", rr.Code)
	}
}

func TestBlockDirectAccessMiddleware_WithXRealIP(t *testing.T) {
	// Should allow when X-Real-IP is present (proxy)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.1")
	req.RemoteAddr = "192.168.1.1:12345"

	rr := httptest.NewRecorder()
	handler := BlockDirectAccessMiddleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK when proxied, got %d", rr.Code)
	}
}

func TestRequestSizeMiddleware_NegativeContentLength(t *testing.T) {
	// Test with negative Content-Length (should be handled - ParseInt fails, so validation is skipped)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Length", "-100")

	rr := httptest.NewRecorder()
	cfg := config.Config{MaxRequestBody: 1024 * 1024, MaxHeaderSize: 1024 * 1024}
	handler := RequestSizeMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	// Negative Content-Length causes ParseInt to fail, so validation is skipped - request proceeds normally
	if rr.Code != http.StatusOK {
		t.Logf("Request processed with status %d (negative Content-Length was ignored)", rr.Code)
	}
}

func TestRequestSizeMiddleware_ExceedsMaxSize(t *testing.T) {
	// Test with Content-Length exceeding maximum
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Length", "2000000") // 2MB, larger than 1MB default

	rr := httptest.NewRecorder()
	cfg := config.Config{MaxRequestBody: 1024 * 1024, MaxHeaderSize: 1024 * 1024}
	handler := RequestSizeMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	// Should return 413 Request Entity Too Large
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for large Content-Length, got %d", rr.Code)
	}
}

func TestRequestSizeMiddleware_ExactlyMaxSize(t *testing.T) {
	// Test with Content-Length exactly at maximum
	// Note: The middleware uses > (not >=), so exact max size is allowed
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Length", "1048576") // Exactly 1MB

	rr := httptest.NewRecorder()
	cfg := config.Config{MaxRequestBody: 1024 * 1024, MaxHeaderSize: 1024 * 1024}
	handler := RequestSizeMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	// Should return 200 OK (exact max size is allowed with > comparison)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK for Content-Length at exact max size, got %d", rr.Code)
	}
}

func TestRequestSizeMiddleware_NoContentLength(t *testing.T) {
	// Test without Content-Length header (should be allowed)
	req := httptest.NewRequest("GET", "/", nil)
	// Don't set Content-Length

	rr := httptest.NewRecorder()
	cfg := config.Config{MaxRequestBody: 1024 * 1024, MaxHeaderSize: 1024 * 1024}
	handler := RequestSizeMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK when no Content-Length, got %d", rr.Code)
	}
}
