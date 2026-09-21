// Package metrics provides Prometheus metrics collection for HTTP server monitoring.
// It exports three metrics for tracking HTTP request performance:
//   - http_request_total: Counter with method, path, and status labels
//   - http_request_duration_seconds: Histogram with method and path labels
//   - http_request_in_flight: Gauge for concurrent requests
//
// It also exports observability for the ANSM documents (RCP/notice) feature:
//   - docs_cache_hits_total / docs_cache_misses_total: document cache Counters
//   - ansm_fetch_requests_total: Counter with a status label (see the
//     FetchStatus* constants)
//   - ansm_fetch_duration_seconds: Histogram of upstream fetch latency
//   - docs_store_documents / docs_store_tombstones: document store Gauges
//
// All metrics are automatically registered with the Prometheus default registry
// during package initialization.
package metrics

import "github.com/prometheus/client_golang/prometheus"

// Label values for the ansm_fetch_requests_total status label. They classify
// the outcome of one end-to-end upstream interaction triggered by a cache
// miss: a successfully fetched document, a document definitively absent on
// the ANSM page (tombstone candidate), or any other failure.
const (
	FetchStatusSuccess  = "success"
	FetchStatusNotFound = "notfound"
	FetchStatusError    = "error"
)

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

	RateLimiterBucketsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rate_limiter_buckets_total",
			Help: "Total number of rate limiter buckets (IPs seen in last ~5 minutes)",
		},
	)

	// DocsCacheHits counts ANSM documents served straight from the on-disk
	// cache (no upstream fetch needed).
	DocsCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "docs_cache_hits_total",
			Help: "Total ANSM documents (RCP/notice) served from the on-disk cache",
		},
	)

	// DocsCacheMisses counts document lookups that were absent from the
	// cache and triggered an upstream fetch. Tombstone lookups (negative
	// cache hits) are counted in neither counter: no document is served
	// and no fetch happens; their signal lives in the tombstones gauge and
	// in ansm_fetch_requests_total{status="notfound"} recorded when the
	// tombstone was created.
	DocsCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "docs_cache_misses_total",
			Help: "Total ANSM document lookups missed in the on-disk cache and fetched upstream (tombstone lookups excluded)",
		},
	)

	// ANSMFetchRequests counts end-to-end upstream fetch interactions with
	// a status label using the FetchStatus* constants.
	ANSMFetchRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ansm_fetch_requests_total",
			Help: "Total ANSM upstream fetches by outcome (success, notfound, error)",
		},
		[]string{"status"},
	)

	// ANSMFetchDuration measures the latency of end-to-end upstream fetch
	// interactions (including internal retries), observed for every
	// outcome. Buckets mirror the HTTP duration histogram (1ms-5s).
	ANSMFetchDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ansm_fetch_duration_seconds",
			Help:    "ANSM upstream fetch latency (including retries, all outcomes)",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
	)

	// DocsStoreDocuments gauges the number of cached documents; it is
	// refreshed from docstore Stats() on diagnostics reads and after
	// successful writes.
	DocsStoreDocuments = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "docs_store_documents",
			Help: "Number of ANSM documents currently held in the on-disk cache",
		},
	)

	// DocsStoreTombstones gauges the number of negative-cache markers.
	DocsStoreTombstones = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "docs_store_tombstones",
			Help: "Number of ANSM document tombstones (negative cache entries) in the on-disk cache",
		},
	)
)

func init() {
	prometheus.MustRegister(HTTPRequestTotals)
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(HTTPRequestInFlight)
	prometheus.MustRegister(RateLimiterBucketsTotal)
	prometheus.MustRegister(DocsCacheHits)
	prometheus.MustRegister(DocsCacheMisses)
	prometheus.MustRegister(ANSMFetchRequests)
	prometheus.MustRegister(ANSMFetchDuration)
	prometheus.MustRegister(DocsStoreDocuments)
	prometheus.MustRegister(DocsStoreTombstones)

	// Pre-initialize the CounterVec children so the family and all three
	// status series are exported from startup. Without this, a CounterVec
	// emits no metric family until its first WithLabelValues call, leaving
	// dashboards empty and absent()-style alerts firing until the first
	// fetch of each kind occurs.
	for _, status := range []string{FetchStatusSuccess, FetchStatusNotFound, FetchStatusError} {
		ANSMFetchRequests.WithLabelValues(status)
	}
}
