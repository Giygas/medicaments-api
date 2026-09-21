package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/giygas/medicaments-api/ansmdocs"
	"github.com/giygas/medicaments-api/docstore"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/metrics"
	"github.com/go-chi/chi/v5"
)

// ============================================================================
// ANSM DOCUMENTS (RCP/NOTICE) V1 ENDPOINT TESTS
// ============================================================================

// docsTestStore is an in-memory interfaces.DocumentStore double. The mocks
// in interfaces_test.go are not importable across packages, so the handler
// tests use these local offline doubles instead.
type docsTestStore struct {
	payload   []byte
	meta      *docstore.DocumentMeta
	tombstone bool
	stats     docstore.Stats

	forceErr error // non-sentinel Get error (internal failure path)
	putErr   error

	getCalls       int
	putCalls       int
	lastPutKey     string
	lastPutPayload []byte
	lastPutDate    string
	tombstoneCalls int
}

func (s *docsTestStore) Get(cis, docType string) ([]byte, *docstore.DocumentMeta, error) {
	s.getCalls++
	switch {
	case s.forceErr != nil:
		return nil, nil, s.forceErr
	case s.tombstone:
		return nil, &docstore.DocumentMeta{Tombstone: true}, docstore.ErrTombstone
	case s.payload != nil:
		return s.payload, s.meta, nil
	default:
		return nil, nil, docstore.ErrNotFound
	}
}

func (s *docsTestStore) Put(cis, docType string, jsonBytes []byte, sourceDate string) error {
	s.putCalls++
	s.lastPutKey = cis + "|" + docType
	s.lastPutPayload = jsonBytes
	s.lastPutDate = sourceDate
	return s.putErr
}

func (s *docsTestStore) PutTombstone(cis, docType string) error {
	s.tombstoneCalls++
	return nil
}

func (s *docsTestStore) Stats() docstore.Stats {
	return s.stats
}

// docsTestFetcher is an interfaces.ANSMFetcher double that never touches
// the network.
type docsTestFetcher struct {
	payload    []byte
	sourceDate string
	err        error

	calls    int
	lastCIS  string
	lastType string
}

func (f *docsTestFetcher) Fetch(ctx context.Context, cis, docType string) ([]byte, string, error) {
	f.calls++
	f.lastCIS = cis
	f.lastType = docType
	return f.payload, f.sourceDate, f.err
}

// Compile-time checks ensuring the doubles satisfy the frozen contracts.
var (
	_ interfaces.DocumentStore = (*docsTestStore)(nil)
	_ interfaces.ANSMFetcher   = (*docsTestFetcher)(nil)
)

// testDocsDocument builds a realistic sectioned document payload and its
// JSON encoding, as ansmdocs.Fetcher would return it.
func testDocsDocument(t *testing.T, docType string) []byte {
	t.Helper()
	doc := ansmdocs.Document{
		CIS:       "60016308",
		Type:      docType,
		Titre:     "CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable",
		MiseAJour: "2025-11-07",
		Source:    ansmdocs.SourceAttribution,
		Sections: []ansmdocs.Section{
			{ID: "1", Titre: "DÉNOMINATION DU MÉDICAMENT", Contenu: "<p>CARVEDILOL VIATRIS 25 mg</p>"},
			{ID: "4.2", Titre: "Posologie et mode d'administration", Contenu: "<p>Un comprimé par jour.</p>"},
		},
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal test document: %v", err)
	}
	return payload
}

// sha256HexOf mirrors docstore's digest so tests can predict the strong
// ETag of freshly fetched documents.
func sha256HexOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestServeDocumentsV1_StatusMatrix walks every branch of the document
// endpoint request flow (kill switch, validation, hit, tombstone, miss,
// upstream outcomes) for both document types.
func TestServeDocumentsV1_StatusMatrix(t *testing.T) {
	// Shared fixtures: one known medicament (CIS 60016308).
	medicaments := []entities.Medicament{
		{Cis: 60016308, Denomination: "CARVEDILOL VIATRIS 25 mg, comprimé"},
	}

	// Computed once for the fresh-fetch ETag expectations.
	rcpPayload := testDocsDocument(t, ansmdocs.TypeRCP)
	noticePayload := testDocsDocument(t, ansmdocs.TypeNotice)

	tests := []struct {
		name             string
		docType          string // URL segment: rcp | notice
		pathCIS          string
		store            *docsTestStore
		fetcher          *docsTestFetcher
		wireOnlyFetcher  bool // inject WithDocuments(nil, fetcher, ...) to test partial wiring
		ifNoneMatch      string
		expectedStatus   int
		expectedETag     string // "" = skip check
		expectCacheCtl   bool
		expectBody       string // "" = skip body content check
		expectEmptyBody  bool
		expectStoreGets  int
		expectFetches    int
		expectPuts       int
		expectTombstones int
		expectLastPutKey string
	}{
		{
			name:           "kill switch without docs dependencies answers 501",
			docType:        "rcp",
			pathCIS:        "60016308",
			store:          nil,
			fetcher:        nil,
			expectedStatus: http.StatusNotImplemented,
		},
		{
			name:            "kill switch when only the fetcher is wired answers 501",
			docType:         "rcp",
			pathCIS:         "60016308",
			fetcher:         &docsTestFetcher{},
			wireOnlyFetcher: true,
			expectedStatus:  http.StatusNotImplemented,
		},
		{
			name:           "invalid CIS answers 400",
			docType:        "rcp",
			pathCIS:        "1234567",
			store:          &docsTestStore{},
			fetcher:        &docsTestFetcher{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unknown CIS answers 404 without upstream or cache access",
			docType:        "notice",
			pathCIS:        "99999999",
			store:          &docsTestStore{},
			fetcher:        &docsTestFetcher{},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:    "cache hit serves stored payload with strong ETag from index",
			docType: "rcp",
			pathCIS: "60016308",
			store: &docsTestStore{
				payload: rcpPayload,
				meta:    &docstore.DocumentMeta{SHA256: "indexsha256value", SourceDate: "2025-11-07"},
			},
			fetcher:         &docsTestFetcher{},
			expectedStatus:  http.StatusOK,
			expectedETag:    `"indexsha256value"`,
			expectCacheCtl:  true,
			expectBody:      string(rcpPayload),
			expectStoreGets: 1,
		},
		{
			name:    "cache hit honors If-None-Match with 304",
			docType: "rcp",
			pathCIS: "60016308",
			store: &docsTestStore{
				payload: rcpPayload,
				meta:    &docstore.DocumentMeta{SHA256: "indexsha256value"},
			},
			fetcher:         &docsTestFetcher{},
			ifNoneMatch:     `"indexsha256value"`,
			expectedStatus:  http.StatusNotModified,
			expectedETag:    `"indexsha256value"`,
			expectCacheCtl:  true,
			expectEmptyBody: true,
			expectStoreGets: 1,
		},
		{
			name:    "cache hit honors weak-prefixed If-None-Match with 304",
			docType: "notice",
			pathCIS: "60016308",
			store: &docsTestStore{
				payload: noticePayload,
				meta:    &docstore.DocumentMeta{SHA256: "indexsha256value"},
			},
			fetcher:         &docsTestFetcher{},
			ifNoneMatch:     `W/"indexsha256value"`,
			expectedStatus:  http.StatusNotModified,
			expectedETag:    `"indexsha256value"`,
			expectEmptyBody: true,
			expectStoreGets: 1,
		},
		{
			name:    "mismatching If-None-Match serves full payload",
			docType: "rcp",
			pathCIS: "60016308",
			store: &docsTestStore{
				payload: rcpPayload,
				meta:    &docstore.DocumentMeta{SHA256: "indexsha256value"},
			},
			fetcher:         &docsTestFetcher{},
			ifNoneMatch:     `"differentvalue"`,
			expectedStatus:  http.StatusOK,
			expectedETag:    `"indexsha256value"`,
			expectBody:      string(rcpPayload),
			expectStoreGets: 1,
		},
		{
			name:            "tombstone answers 404 and never refetches",
			docType:         "rcp",
			pathCIS:         "60016308",
			store:           &docsTestStore{tombstone: true},
			fetcher:         &docsTestFetcher{},
			expectedStatus:  http.StatusNotFound,
			expectStoreGets: 1,
		},
		{
			name:            "non-sentinel store error answers 500",
			docType:         "rcp",
			pathCIS:         "60016308",
			store:           &docsTestStore{forceErr: context.Canceled},
			fetcher:         &docsTestFetcher{},
			expectedStatus:  http.StatusInternalServerError,
			expectStoreGets: 1,
		},
		{
			name:    "miss fetches upstream, caches then serves",
			docType: "rcp",
			pathCIS: "60016308",
			store:   &docsTestStore{},
			fetcher: &docsTestFetcher{
				payload:    rcpPayload,
				sourceDate: "2025-11-07",
			},
			expectedStatus:   http.StatusOK,
			expectedETag:     `"` + sha256HexOf(rcpPayload) + `"`,
			expectCacheCtl:   true,
			expectBody:       string(rcpPayload),
			expectStoreGets:  1,
			expectFetches:    1,
			expectPuts:       1,
			expectLastPutKey: "60016308|rcp",
		},
		{
			name:    "notice miss fetches upstream with the notice docType",
			docType: "notice",
			pathCIS: "60016308",
			store:   &docsTestStore{},
			fetcher: &docsTestFetcher{
				payload:    noticePayload,
				sourceDate: "2026-01-15",
			},
			expectedStatus:   http.StatusOK,
			expectedETag:     `"` + sha256HexOf(noticePayload) + `"`,
			expectBody:       string(noticePayload),
			expectStoreGets:  1,
			expectFetches:    1,
			expectPuts:       1,
			expectLastPutKey: "60016308|notice",
		},
		{
			name:    "fresh document is served even when caching fails",
			docType: "rcp",
			pathCIS: "60016308",
			store:   &docsTestStore{putErr: context.DeadlineExceeded},
			fetcher: &docsTestFetcher{
				payload:    rcpPayload,
				sourceDate: "2025-11-07",
			},
			expectedStatus:  http.StatusOK,
			expectedETag:    `"` + sha256HexOf(rcpPayload) + `"`,
			expectBody:      string(rcpPayload),
			expectStoreGets: 1,
			expectFetches:   1,
			expectPuts:      1,
		},
		{
			name:    "document absent upstream records tombstone and answers 404",
			docType: "rcp",
			pathCIS: "60016308",
			store:   &docsTestStore{},
			fetcher: &docsTestFetcher{
				err: ansmdocs.ErrNotAvailable,
			},
			expectedStatus:   http.StatusNotFound,
			expectStoreGets:  1,
			expectFetches:    1,
			expectTombstones: 1,
		},
		{
			name:    "upstream fetch failure answers 502",
			docType: "notice",
			pathCIS: "60016308",
			store:   &docsTestStore{},
			fetcher: &docsTestFetcher{
				err: context.DeadlineExceeded,
			},
			expectedStatus:  http.StatusBadGateway,
			expectStoreGets: 1,
			expectFetches:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var opts []HandlerOption
			if tt.wireOnlyFetcher {
				opts = append(opts, WithDocuments(nil, tt.fetcher, time.Second))
			} else if tt.store != nil || tt.fetcher != nil {
				opts = append(opts, WithDocuments(tt.store, tt.fetcher, time.Second))
			}
			handler := NewHandler(
				NewMockDataStoreBuilder().WithMedicaments(medicaments).Build(),
				NewMockDataValidatorBuilder().Build(),
				NewMockHealthCheckerBuilder().Build(),
				opts...,
			)

			router := chi.NewRouter()
			router.Get("/v1/medicaments/{cis}/rcp", handler.ServeRCPV1)
			router.Get("/v1/medicaments/{cis}/notice", handler.ServeNoticeV1)

			req := httptest.NewRequest("GET", "/v1/medicaments/"+tt.pathCIS+"/"+tt.docType, nil)
			if tt.ifNoneMatch != "" {
				req.Header.Set("If-None-Match", tt.ifNoneMatch)
			}
			rr := httptest.NewRecorder()

			// Act
			router.ServeHTTP(rr, req)

			// Assert: status
			if rr.Code != tt.expectedStatus {
				t.Fatalf("Expected status %d, got %d (body: %s)", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			// Assert: headers
			if tt.expectedETag != "" && rr.Header().Get("ETag") != tt.expectedETag {
				t.Errorf("Expected ETag %s, got %s", tt.expectedETag, rr.Header().Get("ETag"))
			}
			if tt.expectCacheCtl {
				if got := rr.Header().Get("Cache-Control"); got != docsCacheControl {
					t.Errorf("Expected Cache-Control %q, got %q", docsCacheControl, got)
				}
			}
			if tt.expectedStatus == http.StatusOK {
				if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
					t.Errorf("Expected JSON Content-Type, got %q", got)
				}
			}

			// Assert: body
			if tt.expectEmptyBody && rr.Body.Len() != 0 {
				t.Errorf("Expected empty body on 304, got %s", rr.Body.String())
			}
			if tt.expectBody != "" && rr.Body.String() != tt.expectBody {
				t.Errorf("Expected body %s, got %s", tt.expectBody, rr.Body.String())
			}

			// Assert: error envelope on non-2xx/304 responses
			if tt.expectedStatus >= 400 {
				var envelope map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
					t.Fatalf("Error response should be valid JSON: %v", err)
				}
				if code, ok := envelope["code"].(float64); !ok || int(code) != tt.expectedStatus {
					t.Errorf("Expected error code %d, got %v", tt.expectedStatus, envelope["code"])
				}
				if msg, ok := envelope["message"].(string); !ok || msg == "" {
					t.Errorf("Expected non-empty error message, got %v", envelope["message"])
				}
			}

			// Assert: dependency interactions
			if tt.store != nil && tt.store.getCalls != tt.expectStoreGets {
				t.Errorf("Expected %d Get calls, got %d", tt.expectStoreGets, tt.store.getCalls)
			}
			if tt.fetcher != nil && tt.fetcher.calls != tt.expectFetches {
				t.Errorf("Expected %d fetch calls, got %d", tt.expectFetches, tt.fetcher.calls)
			}
			if tt.fetcher != nil && tt.expectFetches > 0 && tt.fetcher.lastType != tt.docType {
				t.Errorf("Expected fetch docType %q, got %q", tt.docType, tt.fetcher.lastType)
			}
			if tt.store != nil && tt.store.putCalls != tt.expectPuts {
				t.Errorf("Expected %d Put calls, got %d", tt.expectPuts, tt.store.putCalls)
			}
			if tt.store != nil && tt.store.tombstoneCalls != tt.expectTombstones {
				t.Errorf("Expected %d PutTombstone calls, got %d", tt.expectTombstones, tt.store.tombstoneCalls)
			}
			if tt.expectLastPutKey != "" && tt.store.lastPutKey != tt.expectLastPutKey {
				t.Errorf("Expected Put key %q, got %q", tt.expectLastPutKey, tt.store.lastPutKey)
			}
			if tt.expectPuts > 0 && tt.store.lastPutDate == "" {
				t.Error("Expected Put to receive a non-empty source date")
			}
		})
	}
}

// TestServeDocumentsV1_AttributionContract verifies the Etalab 2.0 legal
// obligation: every 200 response carries the source attribution and the
// ANSM update date, and the ETag is a strong quoted sha256 hex digest.
func TestServeDocumentsV1_AttributionContract(t *testing.T) {
	// Arrange
	payload := testDocsDocument(t, ansmdocs.TypeRCP)
	store := &docsTestStore{
		payload: payload,
		meta:    &docstore.DocumentMeta{SHA256: sha256HexOf(payload)},
	}
	handler := NewHandler(
		NewMockDataStoreBuilder().WithMedicaments([]entities.Medicament{
			{Cis: 60016308, Denomination: "CARVEDILOL VIATRIS 25 mg, comprimé"},
		}).Build(),
		NewMockDataValidatorBuilder().Build(),
		NewMockHealthCheckerBuilder().Build(),
		WithDocuments(store, &docsTestFetcher{}, time.Second),
	)

	router := chi.NewRouter()
	router.Get("/v1/medicaments/{cis}/rcp", handler.ServeRCPV1)

	req := httptest.NewRequest("GET", "/v1/medicaments/60016308/rcp", nil)
	rr := httptest.NewRecorder()

	// Act
	router.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var doc ansmdocs.Document
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("Response should unmarshal as a sectioned document: %v", err)
	}
	if doc.Source != ansmdocs.SourceAttribution {
		t.Errorf("Expected source attribution %q, got %q", ansmdocs.SourceAttribution, doc.Source)
	}
	if doc.MiseAJour != "2025-11-07" {
		t.Errorf("Expected miseAJour %q, got %q", "2025-11-07", doc.MiseAJour)
	}
	if len(doc.Sections) == 0 {
		t.Error("Expected at least one section in the document")
	}

	etag := rr.Header().Get("ETag")
	if !hasQuotedETag(etag) {
		t.Errorf("ETag should be quoted, got: %s", etag)
	}
	if strings.HasPrefix(etag, "W/") {
		t.Errorf("Document ETag must be strong (no W/ prefix), got: %s", etag)
	}
}

// TestServeDocumentsV1_RouteCoexistence verifies that the document routes
// coexist with GET /v1/medicaments/{cis} in the chi router: the plain CIS
// lookup keeps working while /rcp and /notice route to the document
// handlers.
func TestServeDocumentsV1_RouteCoexistence(t *testing.T) {
	// Arrange
	medicaments := []entities.Medicament{
		{Cis: 60016308, Denomination: "CARVEDILOL VIATRIS 25 mg, comprimé"},
	}
	store := &docsTestStore{
		payload: testDocsDocument(t, ansmdocs.TypeRCP),
		meta:    &docstore.DocumentMeta{SHA256: "coexistencesha256"},
	}
	fetcher := &docsTestFetcher{
		payload:    testDocsDocument(t, ansmdocs.TypeNotice),
		sourceDate: "2026-01-15",
	}
	handler := NewHandler(
		NewMockDataStoreBuilder().WithMedicaments(medicaments).Build(),
		NewMockDataValidatorBuilder().Build(),
		NewMockHealthCheckerBuilder().Build(),
		WithDocuments(store, fetcher, time.Second),
	)

	// Mirror the exact registration performed by server.setupRoutes.
	router := chi.NewRouter()
	router.Get("/v1/medicaments/{cis}", handler.FindMedicamentByCIS)
	router.Get("/v1/medicaments/{cis}/rcp", handler.ServeRCPV1)
	router.Get("/v1/medicaments/{cis}/notice", handler.ServeNoticeV1)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"plain CIS lookup still served", "/v1/medicaments/60016308", http.StatusOK},
		{"rcp document served from cache", "/v1/medicaments/60016308/rcp", http.StatusOK},
		{"notice document served after fetch", "/v1/medicaments/60016308/notice", http.StatusOK},
		{"unknown document type falls through to 404", "/v1/medicaments/60016308/smotreb", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			// Act
			router.ServeHTTP(rr, req)

			// Assert
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d for %s, got %d (body: %s)",
					tt.expectedStatus, tt.path, rr.Code, rr.Body.String())
			}
		})
	}
}

// ============================================================================
// ANSM DOCUMENTS METRICS + DIAGNOSTICS TESTS
// ============================================================================

// promGaugeValue reads the current value of a Prometheus gauge.
func promGaugeValue(g prometheus.Gauge) float64 {
	var m dto.Metric
	if err := g.Write(&m); err != nil {
		return 0
	}
	return m.GetGauge().GetValue()
}

// docsFetchCounterValue reads the current value of one labeled child of
// ansm_fetch_requests_total.
func docsFetchCounterValue(status string) float64 {
	return promCounterValue(metrics.ANSMFetchRequests.WithLabelValues(status))
}

// docsFetchSampleCount reads the number of observations recorded in
// ansm_fetch_duration_seconds.
func docsFetchSampleCount() uint64 {
	var m dto.Metric
	if err := metrics.ANSMFetchDuration.Write(&m); err != nil {
		return 0
	}
	return m.GetHistogram().GetSampleCount()
}

// docsMetricsSnapshot captures every docs metric value so tests can assert
// exact deltas on the global Prometheus collectors.
type docsMetricsSnapshot struct {
	hits, misses                            float64
	fetchSuccess, fetchNotFound, fetchError float64
	fetchSamples                            uint64
	gaugeDocuments, gaugeTombstones         float64
}

// readDocsMetricsSnapshot captures the current docs metric values.
func readDocsMetricsSnapshot() docsMetricsSnapshot {
	return docsMetricsSnapshot{
		hits:            promCounterValue(metrics.DocsCacheHits),
		misses:          promCounterValue(metrics.DocsCacheMisses),
		fetchSuccess:    docsFetchCounterValue(metrics.FetchStatusSuccess),
		fetchNotFound:   docsFetchCounterValue(metrics.FetchStatusNotFound),
		fetchError:      docsFetchCounterValue(metrics.FetchStatusError),
		fetchSamples:    docsFetchSampleCount(),
		gaugeDocuments:  promGaugeValue(metrics.DocsStoreDocuments),
		gaugeTombstones: promGaugeValue(metrics.DocsStoreTombstones),
	}
}

// docsMetricsDeltas is the expected difference between two snapshots.
type docsMetricsDeltas struct {
	hits, misses                            float64
	fetchSuccess, fetchNotFound, fetchError float64
	fetchSamples                            uint64
}

// TestServeDocumentsV1_MetricsInstrumentation walks every instrumented
// branch of the document flow (hit, miss, tombstone, fetch outcomes) and
// asserts the exact counter, histogram and gauge deltas it produces.
func TestServeDocumentsV1_MetricsInstrumentation(t *testing.T) {
	// Shared fixtures: one known medicament (CIS 60016308).
	medicaments := []entities.Medicament{
		{Cis: 60016308, Denomination: "CARVEDILOL VIATRIS 25 mg, comprimé"},
	}
	rcpPayload := testDocsDocument(t, ansmdocs.TypeRCP)

	// gaugeSentinel is a value store Stats() can never produce; it detects
	// whether a branch refreshed the gauges or left them untouched.
	const gaugeSentinel = -1.0

	tests := []struct {
		name             string
		store            *docsTestStore
		fetcher          *docsTestFetcher
		expectedStatus   int
		want             docsMetricsDeltas
		wantGaugeRefresh bool // gauges end up at store.stats values instead of the sentinel
	}{
		{
			name: "cache hit increments only docs_cache_hits_total",
			store: &docsTestStore{
				payload: rcpPayload,
				meta:    &docstore.DocumentMeta{SHA256: "metricsHitSha256"},
			},
			fetcher:        &docsTestFetcher{},
			expectedStatus: http.StatusOK,
			want:           docsMetricsDeltas{hits: 1},
		},
		{
			name:           "tombstone lookup increments neither counter",
			store:          &docsTestStore{tombstone: true},
			fetcher:        &docsTestFetcher{},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:             "miss with successful fetch counts miss, success and duration, refreshes gauges",
			store:            &docsTestStore{stats: docstore.Stats{Documents: 5, Tombstones: 1, TotalBytes: 4096}},
			fetcher:          &docsTestFetcher{payload: rcpPayload, sourceDate: "2025-11-07"},
			expectedStatus:   http.StatusOK,
			want:             docsMetricsDeltas{misses: 1, fetchSuccess: 1, fetchSamples: 1},
			wantGaugeRefresh: true,
		},
		{
			name:             "miss with absent upstream document counts miss and notfound fetch, refreshes gauges",
			store:            &docsTestStore{stats: docstore.Stats{Documents: 2, Tombstones: 7, TotalBytes: 512}},
			fetcher:          &docsTestFetcher{err: ansmdocs.ErrNotAvailable},
			expectedStatus:   http.StatusNotFound,
			want:             docsMetricsDeltas{misses: 1, fetchNotFound: 1, fetchSamples: 1},
			wantGaugeRefresh: true,
		},
		{
			name:           "miss with failing fetch counts miss and error fetch, leaves gauges untouched",
			store:          &docsTestStore{},
			fetcher:        &docsTestFetcher{err: context.DeadlineExceeded},
			expectedStatus: http.StatusBadGateway,
			want:           docsMetricsDeltas{misses: 1, fetchError: 1, fetchSamples: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler := NewHandler(
				NewMockDataStoreBuilder().WithMedicaments(medicaments).Build(),
				NewMockDataValidatorBuilder().Build(),
				NewMockHealthCheckerBuilder().Build(),
				WithDocuments(tt.store, tt.fetcher, time.Second),
			)
			router := chi.NewRouter()
			router.Get("/v1/medicaments/{cis}/rcp", handler.ServeRCPV1)

			metrics.DocsStoreDocuments.Set(gaugeSentinel)
			metrics.DocsStoreTombstones.Set(gaugeSentinel)
			before := readDocsMetricsSnapshot()

			req := httptest.NewRequest("GET", "/v1/medicaments/60016308/rcp", nil)
			rr := httptest.NewRecorder()

			// Act
			router.ServeHTTP(rr, req)

			// Assert: the branch was reached at all.
			if rr.Code != tt.expectedStatus {
				t.Fatalf("Expected status %d, got %d (body: %s)", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			after := readDocsMetricsSnapshot()
			got := docsMetricsDeltas{
				hits:          after.hits - before.hits,
				misses:        after.misses - before.misses,
				fetchSuccess:  after.fetchSuccess - before.fetchSuccess,
				fetchNotFound: after.fetchNotFound - before.fetchNotFound,
				fetchError:    after.fetchError - before.fetchError,
				fetchSamples:  after.fetchSamples - before.fetchSamples,
			}
			if got != tt.want {
				t.Errorf("Metric deltas: got %+v, want %+v", got, tt.want)
			}

			if tt.wantGaugeRefresh {
				if after.gaugeDocuments != float64(tt.store.stats.Documents) {
					t.Errorf("Expected docs_store_documents gauge %d, got %v",
						tt.store.stats.Documents, after.gaugeDocuments)
				}
				if after.gaugeTombstones != float64(tt.store.stats.Tombstones) {
					t.Errorf("Expected docs_store_tombstones gauge %d, got %v",
						tt.store.stats.Tombstones, after.gaugeTombstones)
				}
			} else {
				if after.gaugeDocuments != gaugeSentinel || after.gaugeTombstones != gaugeSentinel {
					t.Errorf("Expected gauges untouched at sentinel %v, got documents=%v tombstones=%v",
						gaugeSentinel, after.gaugeDocuments, after.gaugeTombstones)
				}
			}
		})
	}
}

// TestServeDiagnosticsV1_DocsSection verifies the doc store stats section
// of /v1/diagnostics: present with live values when the feature is wired,
// entirely absent (shape unchanged) when the kill switch is active.
func TestServeDiagnosticsV1_DocsSection(t *testing.T) {
	tests := []struct {
		name            string
		store           *docsTestStore
		fetcher         *docsTestFetcher
		wantDocsPresent bool
		wantDocuments   float64
		wantTombstones  float64
		wantTotalBytes  float64
	}{
		{
			name:            "wired dependencies expose live store stats",
			store:           &docsTestStore{stats: docstore.Stats{Documents: 3, Tombstones: 2, TotalBytes: 1234}},
			fetcher:         &docsTestFetcher{},
			wantDocsPresent: true,
			wantDocuments:   3,
			wantTombstones:  2,
			wantTotalBytes:  1234,
		},
		{
			name:            "kill switch omits the docs section entirely",
			store:           nil,
			fetcher:         nil,
			wantDocsPresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var opts []HandlerOption
			if tt.store != nil || tt.fetcher != nil {
				opts = append(opts, WithDocuments(tt.store, tt.fetcher, time.Second))
			}
			handler := NewHandler(
				NewMockDataStoreBuilder().WithMedicaments([]entities.Medicament{
					{Cis: 60016308, Denomination: "CARVEDILOL VIATRIS 25 mg, comprimé"},
				}).Build(),
				NewMockDataValidatorBuilder().Build(),
				NewMockHealthCheckerBuilder().Build(),
				opts...,
			)

			// The hit/miss counters are read at request time: capture the
			// values the section must mirror.
			wantHits := promCounterValue(metrics.DocsCacheHits)
			wantMisses := promCounterValue(metrics.DocsCacheMisses)

			req := httptest.NewRequest("GET", "/v1/diagnostics", nil)
			rr := httptest.NewRecorder()

			// Act
			handler.ServeDiagnosticsV1(rr, req)

			// Assert
			if rr.Code != http.StatusOK {
				t.Fatalf("Expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
			}

			var envelope map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("Diagnostics response should be valid JSON: %v", err)
			}

			// The existing diagnostics shape must survive the extension.
			for _, key := range []string{"timestamp", "uptime_seconds", "next_update", "data_age_hours", "system", "data_integrity"} {
				if _, ok := envelope[key]; !ok {
					t.Errorf("Existing diagnostics key %q missing from response", key)
				}
			}

			docsRaw, present := envelope["docs"]
			if present != tt.wantDocsPresent {
				t.Fatalf("docs section presence: got %v, want %v", present, tt.wantDocsPresent)
			}
			if !tt.wantDocsPresent {
				return
			}

			docs, ok := docsRaw.(map[string]any)
			if !ok {
				t.Fatalf("docs section should be a JSON object, got %T", docsRaw)
			}
			if enabled, ok := docs["enabled"].(bool); !ok || !enabled {
				t.Errorf("Expected docs.enabled true, got %v", docs["enabled"])
			}
			if got := docs["documents"].(float64); got != tt.wantDocuments {
				t.Errorf("Expected docs.documents %v, got %v", tt.wantDocuments, got)
			}
			if got := docs["tombstones"].(float64); got != tt.wantTombstones {
				t.Errorf("Expected docs.tombstones %v, got %v", tt.wantTombstones, got)
			}
			if got := docs["total_bytes"].(float64); got != tt.wantTotalBytes {
				t.Errorf("Expected docs.total_bytes %v, got %v", tt.wantTotalBytes, got)
			}
			if got := docs["cache_hits_total"].(float64); got != wantHits {
				t.Errorf("Expected docs.cache_hits_total %v, got %v", wantHits, got)
			}
			if got := docs["cache_misses_total"].(float64); got != wantMisses {
				t.Errorf("Expected docs.cache_misses_total %v, got %v", wantMisses, got)
			}

			// Reading Stats() for diagnostics must also refresh the gauges.
			if got := promGaugeValue(metrics.DocsStoreDocuments); got != tt.wantDocuments {
				t.Errorf("Expected docs_store_documents gauge %v after diagnostics, got %v", tt.wantDocuments, got)
			}
			if got := promGaugeValue(metrics.DocsStoreTombstones); got != tt.wantTombstones {
				t.Errorf("Expected docs_store_tombstones gauge %v after diagnostics, got %v", tt.wantTombstones, got)
			}
		})
	}
}
