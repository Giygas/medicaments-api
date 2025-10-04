package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/logging"
	"github.com/juju/ratelimit"
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

						w.Header().Set("Content-Type", "application/json")
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

				w.Header().Set("Content-Type", "application/json")
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
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*ratelimit.Bucket),
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

// cleanup removes old clients periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			// Remove clients with full buckets
			for ip, bucket := range rl.clients {
				if bucket.Available() == bucket.Capacity() {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
}

var globalRateLimiter = NewRateLimiter()

func init() {
	globalRateLimiter.cleanup()
}

func getTokenCost(r *http.Request) int64 {
	path := r.URL.Path

	// Check for exact matches first
	switch path {
	case "/":
		return 0 // Free access to index page
	case "/docs":
		return 0 // Free access to docs page
	case "/docs/openapi.yaml":
		return 0 // Free access to OpenAPI spec
	case "/favicon.ico":
		return 0 // Free access to favicon
	case "/database":
		return 200 // Higher cost for full database
	case "/health":
		return 5 // Low cost for health check
	}

	// Check for path patterns
	if len(path) > 0 {
		switch {
		case path == "/database" || (len(path) > 10 && path[:10] == "/database/"):
			return 20 // Paged database access
		case path == "/medicament" || (len(path) > 12 && path[:12] == "/medicament/"):
			return 100 // Medicament search or lookup
		case path == "/generiques" || (len(path) > 11 && path[:11] == "/generiques/"):
			return 20 // Generique search or lookup
		}
	}

	return 20 // Default cost for other endpoints
}

// RateLimitHandler implements rate limiting using token bucket
func RateLimitHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr

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
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			logging.Error("Failed to encode JSON response", "error", err)
		}
	}
}
