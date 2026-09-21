package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/giygas/medicaments-api/ansmdocs"
	"github.com/giygas/medicaments-api/docstore"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/metrics"
)

// HTTP semantics for the ANSM document endpoints (spec decisions 9 and 11).
const (
	// docsCacheControl is the cache policy served with document responses:
	// one hour of client-side freshness, 24 hours of shared cache (nginx).
	docsCacheControl = "public, max-age=3600, s-maxage=86400"

	// docsFetchBudgetMargin is added on top of twice the per-request fetch
	// timeout when bounding the upstream fetch context: the fetcher may
	// perform one retry (second full request) plus rate limiter wait and
	// backoff, so the margin absorbs those without cancelling a
	// still-viable fetch.
	docsFetchBudgetMargin = 10 * time.Second

	// errDocsDisabled is the 501 message shown when DOCS_ENABLED=false
	// (or the dependencies were never wired).
	errDocsDisabled = "ANSM documents feature is disabled on this server"

	// errDocsNotAvailable is the 404 message for tombstoned or upstream
	// absent documents.
	errDocsNotAvailable = "Document not available for this medicament"

	// errDocsBadGateway is the 502 message for upstream fetch/parse
	// failures.
	errDocsBadGateway = "Failed to retrieve document from ANSM"

	// errDocsStoreFailure is the 500 message for unexpected cache errors.
	errDocsStoreFailure = "Document cache error"
)

// ServeRCPV1 handles GET /v1/medicaments/{cis}/rcp: it serves the ANSM RCP
// (Résumé des Caractéristiques du Produit) of a medicament as sectioned
// JSON, lazily fetched from base-donnees-publique.medicaments.gouv.fr on
// first request and permanently cached afterwards.
func (h *Handler) ServeRCPV1(w http.ResponseWriter, r *http.Request) {
	h.serveDocument(w, r, ansmdocs.TypeRCP)
}

// ServeNoticeV1 handles GET /v1/medicaments/{cis}/notice: it serves the
// ANSM patient notice of a medicament as sectioned JSON, lazily fetched
// from base-donnees-publique.medicaments.gouv.fr on first request and
// permanently cached afterwards.
func (h *Handler) ServeNoticeV1(w http.ResponseWriter, r *http.Request) {
	h.serveDocument(w, r, ansmdocs.TypeNotice)
}

// serveDocument implements the shared request flow of both document
// endpoints, in order:
//
//  1. kill switch: nil store or fetcher (DOCS_ENABLED=false) -> 501
//  2. CIS validation and existence check in the data container -> 404
//     without any upstream call for unknown medicaments
//  3. cache hit -> serve with strong ETag and If-None-Match support
//  4. tombstone (negative cache) -> 404, never refetched
//  5. miss -> bounded upstream fetch; ErrNotAvailable records a tombstone
//     and answers 404, other fetch errors answer 502, success is cached
//     then served like a hit
func (h *Handler) serveDocument(w http.ResponseWriter, r *http.Request, docType string) {
	// Kill switch: dependencies are injected only when DOCS_ENABLED=true.
	if h.docStore == nil || h.docFetcher == nil {
		logging.Warn("ANSM documents endpoint disabled (kill switch active)",
			"path", r.URL.Path)
		h.RespondWithError(w, http.StatusNotImplemented, errDocsDisabled)
		return
	}

	// Validate the CIS code itself.
	cisStr := r.PathValue("cis")
	cis, err := h.validator.ValidateCIS(cisStr)
	if err != nil {
		h.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Unknown CIS: 404 before touching the cache or the upstream.
	if _, exists := h.dataStore.GetMedicamentsMap()[cis]; !exists {
		h.RespondWithError(w, http.StatusNotFound, "Medicament not found")
		return
	}

	// Canonical cache key (leading zeros of the path parameter removed).
	key := strconv.Itoa(cis)

	payload, meta, err := h.docStore.Get(key, docType)
	switch {
	case err == nil:
		metrics.DocsCacheHits.Inc()
		logging.Debug("ANSM document served from cache",
			"cis", key, "docType", docType)
		h.serveDocumentBytes(w, r, payload, meta)

	case errors.Is(err, docstore.ErrTombstone):
		// Known absent upstream: negative cache, never refetch. Neither
		// cache counter is incremented (see DocsCacheMisses Help): the
		// tombstone signal lives in the store gauges and in the
		// notfound fetch counter recorded when it was created.
		h.RespondWithError(w, http.StatusNotFound, errDocsNotAvailable)

	case errors.Is(err, docstore.ErrNotFound):
		metrics.DocsCacheMisses.Inc()
		h.fetchStoreAndServeDocument(w, r, key, docType)

	default:
		logging.Error("ANSM documents cache lookup failed",
			"cis", key, "docType", docType, "error", err)
		h.RespondWithError(w, http.StatusInternalServerError, errDocsStoreFailure)
	}
}

// fetchStoreAndServeDocument lazily fetches a missing document upstream,
// caches it and serves it. A document definitively absent upstream
// (ansmdocs.ErrNotAvailable) is recorded as a tombstone so it is never
// fetched again.
func (h *Handler) fetchStoreAndServeDocument(w http.ResponseWriter, r *http.Request, cis, docType string) {
	// Bound the whole upstream operation: the fetcher may issue up to two
	// requests (initial + retry) plus rate limiter wait and backoff, so
	// grant twice the per-request timeout plus a fixed margin.
	ctx, cancel := context.WithTimeout(r.Context(), 2*h.docFetchTimeout+docsFetchBudgetMargin)
	defer cancel()

	fetchStart := time.Now()
	payload, sourceDate, err := h.docFetcher.Fetch(ctx, cis, docType)
	metrics.ANSMFetchDuration.Observe(time.Since(fetchStart).Seconds())

	switch {
	case errors.Is(err, ansmdocs.ErrNotAvailable):
		metrics.ANSMFetchRequests.WithLabelValues(metrics.FetchStatusNotFound).Inc()
		// Negative cache: the document does not exist upstream, never
		// refetch this (cis, docType) pair.
		if terr := h.docStore.PutTombstone(cis, docType); terr != nil {
			logging.Error("Failed to record document tombstone",
				"cis", cis, "docType", docType, "error", terr)
		} else {
			refreshDocsStoreGauges(h.docStore.Stats())
			logging.Info("ANSM document absent upstream, tombstone recorded",
				"cis", cis, "docType", docType)
		}
		h.RespondWithError(w, http.StatusNotFound, errDocsNotAvailable)
		return

	case err != nil:
		metrics.ANSMFetchRequests.WithLabelValues(metrics.FetchStatusError).Inc()
		logging.Warn("ANSM document fetch failed",
			"cis", cis, "docType", docType, "error", err)
		h.RespondWithError(w, http.StatusBadGateway, errDocsBadGateway)
		return
	}
	metrics.ANSMFetchRequests.WithLabelValues(metrics.FetchStatusSuccess).Inc()

	// Cache the freshly fetched document. A caching failure degrades
	// nothing for the client: the document is still served (and will be
	// retried on the next request).
	if err := h.docStore.Put(cis, docType, payload, sourceDate); err != nil {
		logging.Error("Failed to cache fetched ANSM document",
			"cis", cis, "docType", docType, "error", err)
	} else {
		refreshDocsStoreGauges(h.docStore.Stats())
		logging.Info("ANSM document fetched and cached",
			"cis", cis, "docType", docType, "miseAJour", sourceDate, "bytes", len(payload))
	}

	// meta is nil: the strong ETag is derived from the payload itself,
	// which matches the sha256 the store just indexed.
	h.serveDocumentBytes(w, r, payload, nil)
}

// serveDocumentBytes writes a document payload with strong ETag
// validation and the docs cache policy, answering 304 Not Modified when
// the client's If-None-Match matches.
func (h *Handler) serveDocumentBytes(w http.ResponseWriter, r *http.Request, payload []byte, meta *docstore.DocumentMeta) {
	etag := strongDocsETag(payload, meta)

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", docsCacheControl)
	if meta != nil && !meta.FetchedAt.IsZero() {
		w.Header().Set("Last-Modified", meta.FetchedAt.UTC().Format(http.TimeFormat))
	}

	if CheckETag(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(payload); err != nil {
		logging.Error("Failed to write document response", "error", err)
	}
}

// strongDocsETag derives the strong ETag (quoted sha256 hex digest, no W/
// prefix), preferring the store index metadata and falling back to
// hashing the payload — both produce the same digest for a given document.
func strongDocsETag(payload []byte, meta *docstore.DocumentMeta) string {
	if meta != nil && meta.SHA256 != "" {
		return `"` + meta.SHA256 + `"`
	}
	sum := sha256.Sum256(payload)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

// refreshDocsStoreGauges pushes the given store counters into the
// docs_store_documents and docs_store_tombstones Prometheus gauges. It is
// called after every successful store write and on every diagnostics read,
// so the gauges track the live index without a periodic refresher.
func refreshDocsStoreGauges(st docstore.Stats) {
	metrics.DocsStoreDocuments.Set(float64(st.Documents))
	metrics.DocsStoreTombstones.Set(float64(st.Tombstones))
}

// promCounterValue reads the current value of a Prometheus counter. It is
// the diagnostics-side read of counters that this package increments.
func promCounterValue(c prometheus.Counter) float64 {
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		logging.Error("Failed to read Prometheus counter", "error", err)
		return 0
	}
	return m.GetCounter().GetValue()
}

// DocsDiagnosticsImpl defines the doc store stats section of the
// /v1/diagnostics response. It is omitted entirely (JSON "docs" key
// absent) when the documents feature is disabled, so the diagnostics shape
// of servers with DOCS_ENABLED=false is unchanged.
type DocsDiagnosticsImpl struct {
	Enabled          bool    `json:"enabled"`
	Documents        int     `json:"documents"`
	Tombstones       int     `json:"tombstones"`
	TotalBytes       int64   `json:"total_bytes"`
	CacheHitsTotal   float64 `json:"cache_hits_total"`
	CacheMissesTotal float64 `json:"cache_misses_total"`
}

// buildDocsDiagnostics assembles the docs section of /v1/diagnostics: the
// live store counters (also refreshed into the Prometheus gauges), the
// cumulative cache hit/miss counters, and the feature enabled flag. It
// returns nil when the documents dependencies are not wired (kill switch).
func (h *Handler) buildDocsDiagnostics() *DocsDiagnosticsImpl {
	if h.docStore == nil || h.docFetcher == nil {
		return nil
	}

	st := h.docStore.Stats()
	refreshDocsStoreGauges(st)

	return &DocsDiagnosticsImpl{
		Enabled:          true,
		Documents:        st.Documents,
		Tombstones:       st.Tombstones,
		TotalBytes:       st.TotalBytes,
		CacheHitsTotal:   promCounterValue(metrics.DocsCacheHits),
		CacheMissesTotal: promCounterValue(metrics.DocsCacheMisses),
	}
}
