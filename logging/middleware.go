// Package logging a simple slog logger middleware
package logging

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// LoggingMiddleware logs HTTP requests using slog with structured logging
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code and bytes written
			ww := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     200, // default status
			}

			// Call the next handler
			next.ServeHTTP(ww, r)

			// Calculate duration
			duration := time.Since(start)

			// Extract request ID from context
			requestID := r.Context().Value(middleware.RequestIDKey)
			if requestID == nil {
				requestID = "unknown"
			}

			// Log the request with structured data
			logger.InfoContext(r.Context(), "HTTP request",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"status_code", ww.statusCode,
				"bytes_written", ww.bytesWritten,
				"duration_ms", duration.Milliseconds(),
				"duration", duration.String(),
			)
		})
	}
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code and bytes written
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += n
	return n, err
}
