package metrics

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// docsMetricNames lists every metric this feature must expose, with the
// exact names required by the RCP/notice feature spec (decision 10).
var docsMetricNames = []string{
	"docs_cache_hits_total",
	"docs_cache_misses_total",
	"ansm_fetch_requests_total",
	"ansm_fetch_duration_seconds",
	"docs_store_documents",
	"docs_store_tombstones",
}

// counterValue reads the current value of a Prometheus counter.
func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("Failed to read counter: %v", err)
	}
	return m.GetCounter().GetValue()
}

// gaugeValue reads the current value of a Prometheus gauge.
func gaugeValue(t *testing.T, g prometheus.Gauge) float64 {
	t.Helper()
	var m dto.Metric
	if err := g.Write(&m); err != nil {
		t.Fatalf("Failed to read gauge: %v", err)
	}
	return m.GetGauge().GetValue()
}

// TestDocsMetricsRegisteredOnDefaultRegistry verifies that package
// initialization registered every docs metric on the default registry
// (MustRegister panics on failure, so simply reaching here proves a clean
// first registration).
func TestDocsMetricsRegisteredOnDefaultRegistry(t *testing.T) {
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather default registry: %v", err)
	}

	registered := make(map[string]bool)
	for _, mf := range families {
		registered[mf.GetName()] = true
	}

	for _, name := range docsMetricNames {
		if !registered[name] {
			t.Errorf("Metric %q missing from the default registry", name)
		}
	}
}

// TestDocsMetricsReregistrationIsSafe verifies that re-registering the
// docs collectors does not panic: prometheus.Register must report
// AlreadyRegisteredError instead, which is what protects the package init
// (and any future re-wiring) from double-registration panics.
func TestDocsMetricsReregistrationIsSafe(t *testing.T) {
	collectors := []prometheus.Collector{
		DocsCacheHits,
		DocsCacheMisses,
		ANSMFetchRequests,
		ANSMFetchDuration,
		DocsStoreDocuments,
		DocsStoreTombstones,
	}

	for _, c := range collectors {
		err := prometheus.Register(c)
		var are prometheus.AlreadyRegisteredError
		if !errors.As(err, &are) {
			t.Errorf("Re-registering %T: expected AlreadyRegisteredError, got %v", c, err)
		}
	}
}

// TestDocsMetricsBehavior walks the instrumentation primitives used by the
// handlers (counter increments, per-status labeled counters, gauge sets,
// histogram observations) and verifies their read-back values.
func TestDocsMetricsBehavior(t *testing.T) {
	t.Run("cache hit and miss counters increment independently", func(t *testing.T) {
		hitsBefore := counterValue(t, DocsCacheHits)
		missesBefore := counterValue(t, DocsCacheMisses)

		DocsCacheHits.Inc()
		DocsCacheHits.Inc()
		DocsCacheMisses.Inc()

		if got := counterValue(t, DocsCacheHits) - hitsBefore; got != 2 {
			t.Errorf("docs_cache_hits_total delta = %v, want 2", got)
		}
		if got := counterValue(t, DocsCacheMisses) - missesBefore; got != 1 {
			t.Errorf("docs_cache_misses_total delta = %v, want 1", got)
		}
	})

	t.Run("fetch requests counter tracks each status separately", func(t *testing.T) {
		statuses := []string{FetchStatusSuccess, FetchStatusNotFound, FetchStatusError}

		baseline := make(map[string]float64)
		for _, status := range statuses {
			baseline[status] = counterValue(t, ANSMFetchRequests.WithLabelValues(status))
		}

		ANSMFetchRequests.WithLabelValues(FetchStatusSuccess).Inc()
		ANSMFetchRequests.WithLabelValues(FetchStatusNotFound).Inc()
		ANSMFetchRequests.WithLabelValues(FetchStatusNotFound).Inc()

		want := map[string]float64{
			FetchStatusSuccess:  1,
			FetchStatusNotFound: 2,
			FetchStatusError:    0,
		}
		for _, status := range statuses {
			got := counterValue(t, ANSMFetchRequests.WithLabelValues(status)) - baseline[status]
			if got != want[status] {
				t.Errorf("ansm_fetch_requests_total{status=%q} delta = %v, want %v", status, got, want[status])
			}
		}
	})

	t.Run("fetch duration histogram records observations with the HTTP bucket scheme", func(t *testing.T) {
		var before dto.Metric
		if err := ANSMFetchDuration.Write(&before); err != nil {
			t.Fatalf("Failed to read histogram: %v", err)
		}

		ANSMFetchDuration.Observe(0.042)

		var after dto.Metric
		if err := ANSMFetchDuration.Write(&after); err != nil {
			t.Fatalf("Failed to read histogram: %v", err)
		}

		if got := after.GetHistogram().GetSampleCount() - before.GetHistogram().GetSampleCount(); got != 1 {
			t.Errorf("ansm_fetch_duration_seconds sample delta = %d, want 1", got)
		}
		// The bucket scheme must mirror the HTTP duration histogram
		// (1ms-5s range, 11 buckets).
		if got := len(after.GetHistogram().GetBucket()); got != 11 {
			t.Errorf("ansm_fetch_duration_seconds bucket count = %d, want 11", got)
		}
	})

	t.Run("store gauges reflect the last set values", func(t *testing.T) {
		DocsStoreDocuments.Set(15811)
		DocsStoreTombstones.Set(4210)

		if got := gaugeValue(t, DocsStoreDocuments); got != 15811 {
			t.Errorf("docs_store_documents = %v, want 15811", got)
		}
		if got := gaugeValue(t, DocsStoreTombstones); got != 4210 {
			t.Errorf("docs_store_tombstones = %v, want 4210", got)
		}
	})
}
