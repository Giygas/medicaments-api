package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/logging"
	"github.com/juju/ratelimit"
)

var (
	medicamentsParams = []string{"search", "page", "cip"}
	generiquesParams  = []string{"libelle"}
)

// RealIPMiddleware extracts the real IP from X-Forwarded-For header
func RealIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first IP from the comma-separated list
			if idx := strings.Index(xff, ","); idx != -1 {
				xff = xff[:idx]
			}
			r.RemoteAddr = strings.TrimSpace(xff)
		}
		next.ServeHTTP(w, r)
	})
}

// BlockDirectAccessMiddleware blocks direct access to the server
func BlockDirectAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request is coming from nginx (trusted proxy)
		if r.Header.Get("X-Real-IP") == "" && r.Header.Get("X-Forwarded-For") == "" {
			// No proxy headers, likely direct access - check if it's localhost for development
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// If we can't parse the host:port, try to use the whole RemoteAddr as host
				host = r.RemoteAddr
			}

			// Allow localhost access for development
			if host == "127.0.0.1" || host == "::1" || host == "localhost" {
				next.ServeHTTP(w, r)
				return
			}

			logging.Warn("Direct access blocked", "remote_addr", r.RemoteAddr, "user_agent", r.Header.Get("User-Agent"))
			http.Error(w, "Direct access not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestSizeMiddleware limits the size of request headers and body
func RequestSizeMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check Content-Length header if present
			if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
				if length, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
					if length > int64(cfg.MaxRequestBody) {
						logging.Warn("Request body too large",
							"content_length", length,
							"max_allowed", cfg.MaxRequestBody,
							"remote_addr", r.RemoteAddr,
							"user_agent", r.UserAgent())

						w.Header().Set("Content-Type", "application/json; charset=utf-8")
						w.WriteHeader(http.StatusRequestEntityTooLarge)

						errorResponse := map[string]string{
							"error": fmt.Sprintf("Request body too large. Maximum allowed size is %d bytes", cfg.MaxRequestBody),
						}

						respondWithJSON(w, http.StatusRequestEntityTooLarge, errorResponse)
						return
					}
				}
			}

			// Check header size (rough estimate)
			headerSize := int64(0)
			for key, values := range r.Header {
				headerSize += int64(len(key))
				for _, value := range values {
					headerSize += int64(len(value))
				}
			}

			if headerSize > int64(cfg.MaxHeaderSize) {
				logging.Warn("Request headers too large",
					"header_size", headerSize,
					"max_allowed", cfg.MaxHeaderSize,
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent())

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusRequestHeaderFieldsTooLarge)

				errorResponse := map[string]string{
					"error": fmt.Sprintf("Request headers too large. Maximum allowed size is %d bytes", cfg.MaxHeaderSize),
				}

				respondWithJSON(w, http.StatusRequestHeaderFieldsTooLarge, errorResponse)
				return
			}

			// If all checks pass, proceed with the request
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter manages per-client rate limiting
type RateLimiter struct {
	clients map[string]*ratelimit.Bucket
	mu      sync.RWMutex
	wg      sync.WaitGroup

	stopChan chan struct{}
	stopped  bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		clients:  make(map[string]*ratelimit.Bucket),
		stopChan: make(chan struct{}),
		stopped:  false,
	}
}

func (rl *RateLimiter) getBucket(clientIP string) *ratelimit.Bucket {
	rl.mu.RLock()
	bucket, exists := rl.clients[clientIP]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		if bucket, exists = rl.clients[clientIP]; !exists {
			// Create bucket: 3 tokens per second, max 1000 tokens
			bucket = ratelimit.NewBucketWithRate(3, 1000)
			rl.clients[clientIP] = bucket
		}
		rl.mu.Unlock()
	}

	return bucket
}

// cleanup starts a background goroutine that manages rate limiter memory.
// Executes every 30 minutes to remove inactive clients (those with full buckets).
// Continues until shutdown signal is received via stopChan.
// Called once at application startup via init().
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	rl.wg.Add(1)
	go func() {
		defer rl.wg.Done()
		for {
			select {
			case <-ticker.C:
				rl.mu.Lock()
				// Remove clients with full buckets
				for ip, bucket := range rl.clients {
					if bucket.Available() == bucket.Capacity() {
						delete(rl.clients, ip)
					}
				}
				rl.mu.Unlock()
			case <-rl.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

var globalRateLimiter = NewRateLimiter()

// StopRateLimiter stops the rate limiter cleanup goroutine.
// Must be called during application shutdown to prevent goroutine leaks.
// Stops the 30-minute cleanup ticker and ensures all goroutines exit cleanly.
// Safe to call multiple times - first call stops the goroutine, subsequent calls are no-ops.
func StopRateLimiter() {
	rl := globalRateLimiter
	rl.mu.Lock()
	if rl.stopped {
		rl.mu.Unlock()
		return
	}
	rl.stopped = true
	close(rl.stopChan)
	rl.mu.Unlock()

	rl.wg.Wait()
}

func init() {
	globalRateLimiter.cleanup()
}

// getTokenCost returns the rate limit token cost for an HTTP request.
// Cost reflects the computational expense of the operation:
// - Exports and full DB operations: 50-200 tokens
// - Search operations: 20-80 tokens
// - ID lookups and simple queries: 5-10 tokens
// - Unknown/invalid requests: 5 tokens (default)
//
// V1 routes are checked first for performance.
// Legacy routes (deprecated, will be removed) are handled last.
func getTokenCost(r *http.Request) int64 {
	requestPath := r.URL.Path

	q := r.URL.Query()

	// V1 routes - check first for performance
	if strings.HasPrefix(requestPath, "/v1/") {
		const (
			v1MedicamentsPrefix   = "/v1/medicaments/"
			v1PresentationsPrefix = "/v1/presentations/"
			v1GeneriquesPrefix    = "/v1/generiques/"
		)

		// Match /v1/presentations/{id}
		if len(requestPath) > len(v1PresentationsPrefix) &&
			requestPath[:len(v1PresentationsPrefix)] == v1PresentationsPrefix {
			return 5
		}

		// Match /v1/medicaments/{cis} (excludes /v1/medicaments and /v1/medicaments/export/*)
		if len(requestPath) > len(v1MedicamentsPrefix) &&
			requestPath[:len(v1MedicamentsPrefix)] == v1MedicamentsPrefix &&
			!strings.HasPrefix(requestPath[len(v1MedicamentsPrefix):], "export") {
			return 10
		}

		// Matches /v1/generiques/{groupID}
		if len(requestPath) > len(v1GeneriquesPrefix) &&
			requestPath[:len(v1GeneriquesPrefix)] == v1GeneriquesPrefix {
			return 5
		}

		switch requestPath {
		case "/v1/medicaments/export":
			// Full medicament export - expensive operation
			return 200

		case "/v1/medicaments":
			// Ensure only one parameter is present
			if !HasSingleParam(q, medicamentsParams) {
				return 5 // Default for invalid multi-param requests
			}

			if q.Get("search") != "" {
				return 50
			}
			if q.Get("page") != "" {
				return 20
			}
			if q.Get("cip") != "" {
				return 10
			}

			return 5 // Default for /v1/medicaments without recognized params

		case "/v1/generiques":
			// Ensure only one parameter is present
			if !HasSingleParam(q, generiquesParams) {
				return 5 // Default for invalid multi-param requests
			}

			if q.Get("libelle") != "" {
				return 30
			}

			return 5 // Default for /v1/generiques without recognized params
		case "/v1/health", "/health":
			// Health endpoint has no parameters
			return 5
		case "/v1/diagnostics":
			// Diagnostics endpoint - moderate cost (caching prevents recomputation)
			return 30
		}
	}

	// Legacy routes - existing logic preserved
	endpoint, element := path.Split(r.URL.Path)

	switch endpoint {
	case "/":
		switch element {
		case "medicament": // This case is when the user forgot to add something to search
			return 20
		case "database":
			return 200
		case "openapi.yaml":
			return 0
		default:
			return 5
		}
	case "/medicament/id/":
		return 10
	case "/medicament/cip/":
		return 10
	case "/medicament/":
		return 80
	case "/generiques/":
		return 20
	case "/database/":
		return 20
	}

	return 5 // Default cost for other endpoints
}

// HasSingleParam ensures exactly one of the specified parameters is present in the query.
// Returns false if zero or multiple parameters are present.
func HasSingleParam(q url.Values, allowedParams []string) bool {
	count := 0
	for _, param := range allowedParams {
		if q.Get(param) != "" {
			count++
		}
	}
	return count == 1
}

// RateLimitHandler implements rate limiting using token bucket
func RateLimitHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract IP without port from RemoteAddr
		clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// If we can't parse the host:port, use the whole RemoteAddr as IP
			clientIP = r.RemoteAddr
		}

		bucket := globalRateLimiter.getBucket(clientIP)

		// Calculate the token cost for the request
		tokenCost := getTokenCost(r)

		// Add rate limit headers before consuming tokens
		w.Header().Set("X-RateLimit-Limit", "1000")
		w.Header().Set("X-RateLimit-Rate", "3")

		// Check if the client has enough tokens
		if bucket.TakeAvailable(tokenCost) < tokenCost {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(bucket.Available(), 10))

		// Serve the request
		next.ServeHTTP(w, r)
	})
}

// respondWithJSON writes a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload any) {

	if payload == nil {
		payload = map[string]any{}
	}

	// Marshal first (before headers)
	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		logging.Error("Failed to encode JSON response", "error", err)
		return
	}

	// Only send headers after successful encoding
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		logging.Warn("Failed to write response", "error", err)
	}
}
