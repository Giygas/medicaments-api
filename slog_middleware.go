package main

import (
	"log/slog"
	"net/http"
	"time"
)

// slogMiddleware logs HTTP requests using slog with structured logging
func slogMiddleware(next http.Handler) http.Handler {
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

		// Log the request with structured data
		slog.InfoContext(r.Context(), "HTTP request",
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
