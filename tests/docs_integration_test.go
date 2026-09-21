package main

// Integration tests for the ANSM document endpoints (RCP/notice), exercising
// the full lazy-fetch pipeline offline: an httptest server stands in for
// base-donnees-publique.medicaments.gouv.fr, the disk cache lives in a temp
// directory, and every assertion runs against the real router, handler,
// ansmdocs.Fetcher and docstore.DocumentStore stack.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/ansmdocs"
	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/docstore"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/server"
)

// docsTestCIS is the CIS code the fake upstream serves documents for; it
// must exist in the test data container for the handler to accept it. It
// matches the code embedded in the ansmdocs/testdata fixtures.
const docsTestCIS = 60016308

// docsTestDocument mirrors the sectioned JSON wire contract of the document
// endpoints. It is declared locally (instead of reusing ansmdocs.Document)
// so these tests verify the served schema independently of the producing
// package.
type docsTestDocument struct {
	CIS       string `json:"cis"`
	Type      string `json:"type"`
	Titre     string `json:"titre"`
	MiseAJour string `json:"miseAJour"`
	Source    string `json:"source"`
	Sections  []struct {
		ID      string `json:"id"`
		Titre   string `json:"titre"`
		Contenu string `json:"contenu"`
	} `json:"sections"`
}

// docsTestErrorEnvelope mirrors the standard API error envelope.
type docsTestErrorEnvelope struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// fakeANSMUpstream is an httptest server standing in for the ANSM public
// database: it serves the configured page body for any
// /medicament/{cis}/extrait request and counts the requests it received.
type fakeANSMUpstream struct {
	server   *httptest.Server
	hits     atomic.Int64
	badPaths atomic.Int64
	fail     atomic.Bool
}

// newFakeANSMUpstream starts the fake upstream. When block is non-nil,
// every request waits until the channel is closed (used by the singleflight
// test to hold the unique upstream request open while concurrent callers
// pile up). The server is closed automatically at test teardown.
func newFakeANSMUpstream(t *testing.T, body []byte, block <-chan struct{}) *fakeANSMUpstream {
	t.Helper()
	f := &fakeANSMUpstream{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		if !strings.HasPrefix(r.URL.Path, "/medicament/") || !strings.HasSuffix(r.URL.Path, "/extrait") {
			f.badPaths.Add(1)
		}
		if block != nil {
			<-block
		}
		if f.fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(f.server.Close)
	return f
}

// baseURL returns the upstream root the fetcher BaseURL must point at
// (the fetcher itself appends /{cis}/extrait to it).
func (f *fakeANSMUpstream) baseURL() string {
	return f.server.URL + "/medicament"
}

// docsTestStack bundles the pieces under test: the fully routed API server
// with the docs endpoints wired, the temp cache directory and the fake
// upstream.
type docsTestStack struct {
	router   http.Handler
	cacheDir string
	upstream *fakeANSMUpstream
}

// loadDocsFixture reads an ANSM page fixture shipped with the ansmdocs
// package tests (tests run with tests/ as working directory).
func loadDocsFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "ansmdocs", "testdata", name))
	if err != nil {
		t.Fatalf("Failed to read fixture %s: %v", name, err)
	}
	return data
}

// newDocsTestDataContainer builds a data container holding the single
// medicament the fake upstream serves documents for.
func newDocsTestDataContainer() *data.DataContainer {
	medicament := entities.Medicament{
		Cis:          docsTestCIS,
		Denomination: "CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable",
	}
	container := data.NewDataContainer()
	container.UpdateData(
		[]entities.Medicament{medicament},
		[]entities.GeneriqueList{},
		map[int]entities.Medicament{medicament.Cis: medicament},
		map[int]entities.GeneriqueList{},
		map[int]entities.Presentation{},
		map[int]entities.Presentation{},
		&interfaces.DataQualityReport{},
	)
	return container
}

// newDocsTestStack builds a full API server (real router and middleware
// stack, following the setupEndpointsTestServer pattern) with the document
// endpoints wired to a temp docstore and the fake upstream. The fetcher
// disables the upstream rate limiter and retries for deterministic, fast
// upstream hit counting.
func newDocsTestStack(t *testing.T, body []byte, block <-chan struct{}) *docsTestStack {
	t.Helper()

	upstream := newFakeANSMUpstream(t, body, block)

	cacheDir := t.TempDir()
	store, err := docstore.NewDocumentStore(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create document store: %v", err)
	}

	fetcher := ansmdocs.NewFetcher(ansmdocs.Config{
		BaseURL:    upstream.baseURL(),
		RatePerSec: -1, // tests only: no upstream rate limiting
		Timeout:    5 * time.Second,
		MaxRetries: 0, // exactly one upstream attempt per fetch
		Backoff:    10 * time.Millisecond,
	})

	cfg := &config.Config{
		Port:               "0",
		Address:            "localhost",
		Env:                config.EnvTest,
		LogLevel:           "error",
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		DisableRateLimiter: true,
	}
	srv := server.NewServer(cfg, newDocsTestDataContainer(), handlers.WithDocuments(store, fetcher, 5*time.Second))

	return &docsTestStack{router: srv.Router(), cacheDir: cacheDir, upstream: upstream}
}

// docsDoGet performs a GET against the routed API, mimicking a localhost
// client so the direct-access middleware lets it through. It deliberately
// takes no *testing.T so it is safe to call from test goroutines.
func docsDoGet(router http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// docsDoGetIfNoneMatch performs a conditional GET carrying an If-None-Match
// header, mimicking a localhost client.
func docsDoGetIfNoneMatch(router http.Handler, path, etag string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("If-None-Match", etag)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// docsDecodeDocument decodes a 200 response body into the wire-contract
// document.
func docsDecodeDocument(t *testing.T, rr *httptest.ResponseRecorder) docsTestDocument {
	t.Helper()
	var doc docsTestDocument
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("Failed to decode document response: %v\nbody: %s", err, rr.Body.String())
	}
	return doc
}

// docsDecodeError decodes an error response body into the error envelope.
func docsDecodeError(t *testing.T, rr *httptest.ResponseRecorder) docsTestErrorEnvelope {
	t.Helper()
	var env docsTestErrorEnvelope
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("Failed to decode error response: %v\nbody: %s", err, rr.Body.String())
	}
	return env
}

// docsHasSectionID reports whether the document carries a section with the
// given ID.
func docsHasSectionID(doc docsTestDocument, id string) bool {
	for _, s := range doc.Sections {
		if s.ID == id {
			return true
		}
	}
	return false
}

// TestIntegrationDocsLazyFetchLifecycle proves the fetch-once / serve-forever
// contract end to end: the first request lazily fetches upstream exactly once
// and caches the document on disk, subsequent requests are served from the
// store without new upstream hits, conditional requests revalidate via the
// strong ETag, and the notice endpoint fetches its own copy under its own
// cache key.
func TestIntegrationDocsLazyFetchLifecycle(t *testing.T) {
	// Arrange: fake upstream serving a page with both tab panels, fresh disk
	// cache, fully wired API server.
	stack := newDocsTestStack(t, loadDocsFixture(t, "both_tabs.html"), nil)

	// Act: first RCP request lazily fetches the document.
	rr := docsDoGet(stack.router, "/v1/medicaments/60016308/rcp")

	// Assert: 200, exactly one upstream hit, document cached on disk.
	if rr.Code != http.StatusOK {
		t.Fatalf("First RCP request returned status %d, expected %d\nbody: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if hits := stack.upstream.hits.Load(); hits != 1 {
		t.Errorf("Upstream hits after first request = %d, expected exactly 1", hits)
	}
	if bad := stack.upstream.badPaths.Load(); bad != 0 {
		t.Errorf("Upstream received %d requests with an unexpected path shape", bad)
	}
	if _, err := os.Stat(filepath.Join(stack.cacheDir, "60016308_rcp.json.gz")); err != nil {
		t.Errorf("Expected 60016308_rcp.json.gz in the cache directory after first request: %v", err)
	}

	// Assert: sectioned JSON wire contract (envelope, Etalab attribution,
	// canonical RCP section numbering).
	doc := docsDecodeDocument(t, rr)
	if doc.CIS != "60016308" {
		t.Errorf("Document cis = %q, expected %q", doc.CIS, "60016308")
	}
	if doc.Type != "rcp" {
		t.Errorf("Document type = %q, expected %q", doc.Type, "rcp")
	}
	if !strings.Contains(doc.Titre, "CARVEDILOL VIATRIS 25 mg") {
		t.Errorf("Document titre = %q, expected it to contain the medicament name", doc.Titre)
	}
	if doc.MiseAJour != "2025-11-07" {
		t.Errorf("Document miseAJour = %q, expected %q (from the DateNotif 07/11/2025)", doc.MiseAJour, "2025-11-07")
	}
	if doc.Source != "ANSM - Base de données publique des médicaments" {
		t.Errorf("Document source = %q, expected the ANSM attribution required by the Etalab licence", doc.Source)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("Document has no sections")
	}
	if !docsHasSectionID(doc, "1") {
		t.Errorf("RCP sections miss canonical rubrique %q (got IDs from %d sections)", "1", len(doc.Sections))
	}
	if !docsHasSectionID(doc, "4.2") {
		t.Errorf("RCP sections miss canonical rubrique %q (got IDs from %d sections)", "4.2", len(doc.Sections))
	}
	for i, s := range doc.Sections {
		if s.Contenu == "" {
			t.Errorf("Section %q (%d) has empty contenu", s.ID, i)
		}
	}

	// Assert: caching headers on the fetched response.
	etag := rr.Header().Get("ETag")
	if len(etag) != 66 || !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
		t.Errorf("ETag = %q, expected a quoted 64-char sha256 strong ETag", etag)
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "public, max-age=3600, s-maxage=86400" {
		t.Errorf("Cache-Control = %q, expected %q", cc, "public, max-age=3600, s-maxage=86400")
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, expected it to contain application/json", ct)
	}

	// Act: second request must be served from the store.
	rr2 := docsDoGet(stack.router, "/v1/medicaments/60016308/rcp")

	// Assert: same document, same ETag, still exactly one upstream hit.
	if rr2.Code != http.StatusOK {
		t.Fatalf("Second RCP request returned status %d, expected %d", rr2.Code, http.StatusOK)
	}
	if rr2.Body.String() != rr.Body.String() {
		t.Error("Second RCP response body differs from the first")
	}
	if rr2.Header().Get("ETag") != etag {
		t.Errorf("Second RCP ETag = %q, expected the stable %q", rr2.Header().Get("ETag"), etag)
	}
	if hits := stack.upstream.hits.Load(); hits != 1 {
		t.Errorf("Upstream hits after second request = %d, expected still exactly 1 (serve-forever)", hits)
	}

	// Act: conditional request revalidates with the strong ETag.
	rr304 := docsDoGetIfNoneMatch(stack.router, "/v1/medicaments/60016308/rcp", etag)

	// Assert: 304 Not Modified with an empty body.
	if rr304.Code != http.StatusNotModified {
		t.Errorf("Conditional RCP request returned status %d, expected %d", rr304.Code, http.StatusNotModified)
	}
	if rr304.Body.Len() != 0 {
		t.Errorf("304 response body = %q, expected empty", rr304.Body.String())
	}

	// Act: the notice endpoint fetches the same upstream page under its own
	// cache key.
	rrNotice := docsDoGet(stack.router, "/v1/medicaments/60016308/notice")

	// Assert: 200 with the notice contract, slugified section IDs, own cache
	// file, exactly one additional upstream hit.
	if rrNotice.Code != http.StatusOK {
		t.Fatalf("Notice request returned status %d, expected %d\nbody: %s", rrNotice.Code, http.StatusOK, rrNotice.Body.String())
	}
	notice := docsDecodeDocument(t, rrNotice)
	if notice.Type != "notice" {
		t.Errorf("Notice type = %q, expected %q", notice.Type, "notice")
	}
	if notice.MiseAJour != "2026-01-15" {
		t.Errorf("Notice miseAJour = %q, expected %q (from the notice panel DateNotif 15/01/2026)", notice.MiseAJour, "2026-01-15")
	}
	if !docsHasSectionID(notice, "denomination-du-medicament") {
		t.Errorf("Notice sections miss slugified ID %q (notices derive IDs from their headings)", "denomination-du-medicament")
	}
	if _, err := os.Stat(filepath.Join(stack.cacheDir, "60016308_notice.json.gz")); err != nil {
		t.Errorf("Expected 60016308_notice.json.gz in the cache directory after notice request: %v", err)
	}
	if hits := stack.upstream.hits.Load(); hits != 2 {
		t.Errorf("Upstream hits after notice request = %d, expected exactly 2 (one per docType)", hits)
	}
}

// TestIntegrationDocsSingleflightConcurrentFirstRequests proves that N
// concurrent first requests for the same document trigger exactly ONE
// upstream call: the upstream blocks until released so every caller piles up
// on the singleflight barrier, then all N responses carry the same document.
func TestIntegrationDocsSingleflightConcurrentFirstRequests(t *testing.T) {
	// Arrange: blocking upstream so the unique first fetch stays in flight
	// while the concurrent requests arrive. The release is once-guarded so
	// the deferred cleanup cannot double-close the channel on failure
	// paths (the httptest server Close would otherwise deadlock on the
	// still-blocked handler).
	const concurrency = 8
	block := make(chan struct{})
	stack := newDocsTestStack(t, loadDocsFixture(t, "both_tabs.html"), block)
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(block) }) }
	defer release()

	bodies := make([]string, concurrency)
	codes := make([]int, concurrency)
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rr := docsDoGet(stack.router, "/v1/medicaments/60016308/rcp")
			codes[i] = rr.Code
			bodies[i] = rr.Body.String()
		}()
	}

	// Act: fire all requests, let them reach the singleflight barrier, then
	// release the single upstream request.
	close(start)
	time.Sleep(300 * time.Millisecond)
	release()
	wg.Wait()

	// Assert: every concurrent first request succeeded with the same body,
	// and the upstream was hit exactly once.
	for i := range concurrency {
		if codes[i] != http.StatusOK {
			t.Errorf("Concurrent request %d returned status %d, expected %d", i, codes[i], http.StatusOK)
		}
		if bodies[i] != bodies[0] {
			t.Errorf("Concurrent request %d returned a different body than request 0", i)
		}
	}
	if hits := stack.upstream.hits.Load(); hits != 1 {
		t.Errorf("Upstream hits after %d concurrent first requests = %d, expected exactly 1 (singleflight)", concurrency, hits)
	}
}

// TestIntegrationDocsTombstoneNeverRefetch proves the negative-cache
// contract: a document absent upstream (missing tab panel) is recorded as a
// tombstone marker file, answers 404 forever, and is never refetched — while
// the other docType from the same page still fetches normally.
func TestIntegrationDocsTombstoneNeverRefetch(t *testing.T) {
	// Arrange: page carrying an RCP panel but no notice panel.
	stack := newDocsTestStack(t, loadDocsFixture(t, "missing_notice.html"), nil)

	// Act: request the absent notice.
	rr := docsDoGet(stack.router, "/v1/medicaments/60016308/notice")

	// Assert: 404 with the not-available message, tombstone marker on disk,
	// exactly one upstream hit to learn the absence.
	if rr.Code != http.StatusNotFound {
		t.Fatalf("Absent notice request returned status %d, expected %d", rr.Code, http.StatusNotFound)
	}
	env := docsDecodeError(t, rr)
	if !strings.Contains(strings.ToLower(env.Message), "not available") {
		t.Errorf("Absent notice error message = %q, expected it to state the document is not available", env.Message)
	}
	if _, err := os.Stat(filepath.Join(stack.cacheDir, "60016308_notice.missing.json")); err != nil {
		t.Errorf("Expected 60016308_notice.missing.json tombstone marker in the cache directory: %v", err)
	}
	if hits := stack.upstream.hits.Load(); hits != 1 {
		t.Errorf("Upstream hits after absent notice request = %d, expected exactly 1", hits)
	}

	// Act: repeat the request several times.
	for i := range 3 {
		rrRetry := docsDoGet(stack.router, "/v1/medicaments/60016308/notice")
		// Assert: 404 again, zero additional upstream hits (never refetched).
		if rrRetry.Code != http.StatusNotFound {
			t.Errorf("Tombstoned notice request %d returned status %d, expected %d", i+1, rrRetry.Code, http.StatusNotFound)
		}
		if hits := stack.upstream.hits.Load(); hits != 1 {
			t.Fatalf("Upstream hits after tombstoned request %d = %d, expected still 1 (never refetch)", i+1, hits)
		}
	}

	// Act: request the RCP from the same page.
	rrRCP := docsDoGet(stack.router, "/v1/medicaments/60016308/rcp")

	// Assert: the notice tombstone does not block the RCP docType.
	if rrRCP.Code != http.StatusOK {
		t.Errorf("RCP request after notice tombstone returned status %d, expected %d", rrRCP.Code, http.StatusOK)
	}
	if hits := stack.upstream.hits.Load(); hits != 2 {
		t.Errorf("Upstream hits after RCP request = %d, expected exactly 2", hits)
	}
}

// TestIntegrationDocsUnknownCISNoUpstreamCall proves that an unknown CIS is
// rejected with 404 before any cache lookup or upstream call happens.
func TestIntegrationDocsUnknownCISNoUpstreamCall(t *testing.T) {
	// Arrange.
	stack := newDocsTestStack(t, loadDocsFixture(t, "both_tabs.html"), nil)

	// Act: request a document for a CIS absent from the data container.
	rr := docsDoGet(stack.router, "/v1/medicaments/99999999/rcp")

	// Assert: 404 with the medicament-not-found message, zero upstream hits,
	// nothing written to the cache directory.
	if rr.Code != http.StatusNotFound {
		t.Fatalf("Unknown CIS request returned status %d, expected %d", rr.Code, http.StatusNotFound)
	}
	env := docsDecodeError(t, rr)
	if !strings.Contains(strings.ToLower(env.Message), "medicament not found") {
		t.Errorf("Unknown CIS error message = %q, expected the medicament-not-found message", env.Message)
	}
	if hits := stack.upstream.hits.Load(); hits != 0 {
		t.Errorf("Upstream hits after unknown CIS request = %d, expected 0 (no upstream call)", hits)
	}
	entries, err := os.ReadDir(stack.cacheDir)
	if err != nil {
		t.Fatalf("Failed to read cache directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Cache directory contains %d entries after an unknown CIS request, expected 0", len(entries))
	}
}

// TestIntegrationDocsKillSwitchReturns501 proves the DOCS_ENABLED=false kill
// switch: a server built without the document dependencies answers 501 on
// both document endpoints and never touches a cache or an upstream.
func TestIntegrationDocsKillSwitchReturns501(t *testing.T) {
	// Arrange: server WITHOUT handlers.WithDocuments, mirroring
	// DOCS_ENABLED=false wiring (nil store/fetcher is the kill switch).
	cfg := &config.Config{
		Port:               "0",
		Address:            "localhost",
		Env:                config.EnvTest,
		LogLevel:           "error",
		MaxRequestBody:     1048576,
		MaxHeaderSize:      1048576,
		DisableRateLimiter: true,
	}
	srv := server.NewServer(cfg, newDocsTestDataContainer())
	router := srv.Router()

	for _, endpoint := range []string{
		"/v1/medicaments/60016308/rcp",
		"/v1/medicaments/60016308/notice",
	} {
		// Act.
		rr := docsDoGet(router, endpoint)
		// Assert: 501 with the disabled-feature message.
		if rr.Code != http.StatusNotImplemented {
			t.Errorf("Kill-switched %s returned status %d, expected %d", endpoint, rr.Code, http.StatusNotImplemented)
		}
		env := docsDecodeError(t, rr)
		if !strings.Contains(strings.ToLower(env.Message), "disabled") {
			t.Errorf("Kill-switched %s error message = %q, expected it to state the feature is disabled", endpoint, env.Message)
		}
	}
}

// TestIntegrationDocsUpstreamFailureReturns502AndRetries proves the
// transient-failure contract: an upstream 5xx answers 502 WITHOUT recording
// a tombstone, and the next request retries the fetch and succeeds once the
// upstream recovers.
func TestIntegrationDocsUpstreamFailureReturns502AndRetries(t *testing.T) {
	// Arrange: upstream failing with 500.
	stack := newDocsTestStack(t, loadDocsFixture(t, "both_tabs.html"), nil)
	stack.upstream.fail.Store(true)

	// Act: first request hits the failing upstream.
	rr := docsDoGet(stack.router, "/v1/medicaments/60016308/rcp")

	// Assert: 502, exactly one upstream attempt, and NO tombstone recorded
	// (transient failures must stay retryable).
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("Upstream 500 request returned status %d, expected %d\nbody: %s", rr.Code, http.StatusBadGateway, rr.Body.String())
	}
	env := docsDecodeError(t, rr)
	if !strings.Contains(env.Message, "ANSM") {
		t.Errorf("502 error message = %q, expected it to mention the ANSM upstream", env.Message)
	}
	if hits := stack.upstream.hits.Load(); hits != 1 {
		t.Errorf("Upstream hits after failed request = %d, expected exactly 1", hits)
	}
	if _, err := os.Stat(filepath.Join(stack.cacheDir, "60016308_rcp.missing.json")); !os.IsNotExist(err) {
		t.Errorf("A tombstone was recorded after a transient upstream failure (stat err: %v), expected none", err)
	}

	// Act: upstream recovers, next request retries the fetch.
	stack.upstream.fail.Store(false)
	rrRetry := docsDoGet(stack.router, "/v1/medicaments/60016308/rcp")

	// Assert: 200, exactly one additional upstream hit (the retry), document
	// now cached.
	if rrRetry.Code != http.StatusOK {
		t.Fatalf("Retry after upstream recovery returned status %d, expected %d", rrRetry.Code, http.StatusOK)
	}
	if hits := stack.upstream.hits.Load(); hits != 2 {
		t.Errorf("Upstream hits after retry = %d, expected exactly 2 (initial failure + retry)", hits)
	}
	if _, err := os.Stat(filepath.Join(stack.cacheDir, "60016308_rcp.json.gz")); err != nil {
		t.Errorf("Expected 60016308_rcp.json.gz in the cache directory after successful retry: %v", err)
	}

	// Act: a third request is served from the cache.
	rrCached := docsDoGet(stack.router, "/v1/medicaments/60016308/rcp")

	// Assert: no further upstream hit.
	if rrCached.Code != http.StatusOK {
		t.Errorf("Post-retry cached request returned status %d, expected %d", rrCached.Code, http.StatusOK)
	}
	if hits := stack.upstream.hits.Load(); hits != 2 {
		t.Errorf("Upstream hits after cached request = %d, expected still 2", hits)
	}
}
