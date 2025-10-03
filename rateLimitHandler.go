package main

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/juju/ratelimit"
)

// Per-client rate limiting

type RateLimiter struct {
	clients map[string]*ratelimit.Bucket
	mu      sync.RWMutex
}

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

// Clean up old clients periodically
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

func rateLimitHandler(h http.Handler) http.Handler {
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
		h.ServeHTTP(w, r)
	})
}
