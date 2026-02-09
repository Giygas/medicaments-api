// Package metrics provides Prometheus metrics collection for HTTP server monitoring.
// It exports three metrics for tracking HTTP request performance:
//   - http_request_total: Counter with method, path, and status labels
//   - http_request_duration_seconds: Histogram with method and path labels
//   - http_request_in_flight: Gauge for concurrent requests
//
// All metrics are automatically registered with the Prometheus default registry
// during package initialization.
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestTotals = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"method", "path"},
	)

	HTTPRequestInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_request_in_flight",
			Help: "Current in-flight requests",
		},
	)
)

func init() {
	prometheus.MustRegister(HTTPRequestTotals)
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(HTTPRequestInFlight)
}
