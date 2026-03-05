package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		HTTPRequestInFlight.Inc()
		defer HTTPRequestInFlight.Dec()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()

		path := chi.RouteContext(r.Context()).RoutePattern()

		HTTPRequestTotals.WithLabelValues(
			r.Method,
			path,
			fmt.Sprintf("%d", wrapped.statusCode),
		).Inc()

		HTTPRequestDuration.WithLabelValues(
			r.Method,
			path,
		).Observe(duration)
	},
	)
}
