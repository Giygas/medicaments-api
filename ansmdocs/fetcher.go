package ansmdocs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/juju/ratelimit"
	"golang.org/x/sync/singleflight"

	"github.com/giygas/medicaments-api/logging"
)

// Defaults applied by NewFetcher for zero-valued Config fields.
const (
	// DefaultBaseURL is the ANSM public database medicament page root; the
	// fetcher appends /{cis}/extrait to it.
	DefaultBaseURL = "https://base-donnees-publique.medicaments.gouv.fr/medicament"

	// DefaultUserAgent identifies the API towards ANSM. The version segment
	// stays "dev" until build-time version injection exists; override it
	// through Config.UserAgent when wiring the real version.
	DefaultUserAgent = "medicaments-api/dev (+https://medicaments-api.giygas.dev)"

	// DefaultTimeout bounds a single upstream request.
	DefaultTimeout = 15 * time.Second

	// DefaultRatePerSec is the global upstream politeness rate, deliberately
	// separate from (and independent of) the API's own rate limiter.
	DefaultRatePerSec = 2.0

	// DefaultMaxRetries is the number of retries after a failed attempt
	// (1 retry, i.e. up to 2 attempts in total).
	DefaultMaxRetries = 1

	// DefaultBackoff is the wait before a retry when upstream sends no
	// usable Retry-After header.
	DefaultBackoff = time.Second

	// DefaultMaxBodyBytes caps the page size read from upstream.
	DefaultMaxBodyBytes int64 = 10 << 20
)

// maxRetryAfterWait caps how long a Retry-After header may delay a retry:
// polite, but never held hostage by an absurd server value (the caller's
// context remains the ultimate bound).
const maxRetryAfterWait = 2 * time.Minute

// Config configures a Fetcher. Zero-valued fields fall back to the Default*
// constants. RatePerSec == 0 selects DefaultRatePerSec and a negative value
// disables the limiter entirely (tests only). Client is injectable so unit
// tests never touch the network.
type Config struct {
	BaseURL      string
	UserAgent    string
	Timeout      time.Duration
	RatePerSec   float64
	MaxRetries   int
	Backoff      time.Duration
	MaxBodyBytes int64
	Client       *http.Client
}

// validCISRe matches CIS codes (digits only, bounded length so they stay
// safe inside URLs), mirroring the docstore key validation.
var validCISRe = regexp.MustCompile(`^[0-9]{1,10}$`)

// Fetcher lazily retrieves ANSM document pages and turns them into final
// sectioned documents. Politeness controls: custom User-Agent, per-request
// timeout, one retry honoring Retry-After, a global upstream rate limiter
// separate from the API rate limiter, and singleflight deduplication keyed
// by (cis, docType) so concurrent requests for the same document trigger
// exactly one upstream call. A Fetcher is safe for concurrent use.
type Fetcher struct {
	cfg     Config
	client  *http.Client
	limiter *ratelimit.Bucket
	group   singleflight.Group
}

// NewFetcher returns a Fetcher with defaults applied for empty Config
// fields.
func NewFetcher(cfg Config) *Fetcher {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = DefaultMaxRetries
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = DefaultBackoff
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = DefaultMaxBodyBytes
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	var limiter *ratelimit.Bucket
	if cfg.RatePerSec >= 0 {
		rate := cfg.RatePerSec
		if rate == 0 {
			rate = DefaultRatePerSec
		}
		capacity := int64(math.Ceil(rate))
		if capacity < 1 {
			capacity = 1
		}
		limiter = ratelimit.NewBucketWithRate(rate, capacity)
	}
	return &Fetcher{cfg: cfg, client: client, limiter: limiter}
}

// fetchResult is the singleflight payload shared between concurrent callers
// of Fetch for the same (cis, docType) key.
type fetchResult struct {
	payload    []byte
	sourceDate string
}

// Fetch retrieves the ANSM page for cis, extracts the requested document and
// returns the final sectioned JSON bytes ready for docstore.Put, together
// with the source date (YYYY-MM-DD) used for the store metadata.
//
// Outcomes:
//   - success: (json, sourceDate, nil)
//   - document absent upstream: (nil, "", ErrNotAvailable) — record a
//     tombstone, never refetch
//   - invalid (cis, docType): (nil, "", validation error) — no request sent
//   - upstream failure after retries or parse failure: (nil, "", error)
func (f *Fetcher) Fetch(ctx context.Context, cis, docType string) ([]byte, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateKey(cis, docType); err != nil {
		return nil, "", err
	}

	key := cis + "|" + docType
	v, err, shared := f.group.Do(key, func() (any, error) {
		page, err := f.fetchPage(ctx, cis)
		if err != nil {
			return nil, err
		}
		doc, err := Extract(cis, docType, page)
		if err != nil {
			return nil, err
		}
		payload, err := json.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("ansmdocs: failed to encode document: %w", err)
		}
		logging.Debug("ANSM document fetched",
			"cis", cis, "docType", docType,
			"bytes", len(payload), "miseAJour", doc.MiseAJour)
		return fetchResult{payload: payload, sourceDate: doc.MiseAJour}, nil
	})
	if err != nil {
		return nil, "", err
	}
	if shared {
		logging.Debug("ANSM fetch shared between concurrent callers", "key", key)
	}
	res := v.(fetchResult)
	return res.payload, res.sourceDate, nil
}

// fetchPage downloads the medicament page, applying the politeness rules:
// rate-limited attempts, one retry on transient failures or transport
// errors, Retry-After honored when larger than the backoff. Definitive
// upstream absences (404/410) map to ErrNotAvailable.
func (f *Fetcher) fetchPage(ctx context.Context, cis string) ([]byte, error) {
	url := f.pageURL(cis)
	attempts := f.cfg.MaxRetries + 1
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		if err := f.waitTurn(ctx); err != nil {
			return nil, err
		}
		status, retryAfter, body, err := f.doRequest(ctx, url)
		switch {
		case err != nil:
			lastErr = err
			logging.Warn("ANSM request failed", "url", url, "attempt", attempt, "error", err)
		case status == http.StatusOK:
			return body, nil
		case status == http.StatusNotFound || status == http.StatusGone:
			// The page definitively does not exist upstream: same meaning
			// as an absent tab panel (tombstone), not a transient failure.
			return nil, fmt.Errorf("ansmdocs: page for cis %s returned HTTP %d: %w", cis, status, ErrNotAvailable)
		case retryableStatus(status):
			lastErr = fmt.Errorf("ansmdocs: upstream returned HTTP %d", status)
			logging.Warn("ANSM upstream transient failure", "url", url, "attempt", attempt, "status", status)
		default:
			return nil, fmt.Errorf("ansmdocs: unexpected HTTP status %d for cis %s", status, cis)
		}

		if attempt < attempts {
			wait := f.cfg.Backoff
			if retryAfter > wait {
				wait = retryAfter
			}
			if wait > maxRetryAfterWait {
				wait = maxRetryAfterWait
			}
			if err := sleepContext(ctx, wait); err != nil {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("ansmdocs: giving up after %d attempts for cis %s: %w", attempts, cis, lastErr)
}

// doRequest performs one GET, returning the response status, the parsed
// Retry-After delay and the size-capped body.
func (f *Fetcher) doRequest(ctx context.Context, url string) (status int, retryAfter time.Duration, body []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("ansmdocs: failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", f.cfg.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("ansmdocs: request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			logging.Warn("Failed to close ANSM response body", "error", cerr)
		}
	}()

	retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	body, err = io.ReadAll(io.LimitReader(resp.Body, f.cfg.MaxBodyBytes+1))
	if err != nil {
		return resp.StatusCode, 0, nil, fmt.Errorf("ansmdocs: failed to read response body: %w", err)
	}
	if int64(len(body)) > f.cfg.MaxBodyBytes {
		return resp.StatusCode, 0, nil, fmt.Errorf("ansmdocs: response body exceeds %d bytes", f.cfg.MaxBodyBytes)
	}
	return resp.StatusCode, retryAfter, body, nil
}

// pageURL builds the ANSM extrait page URL for a CIS code.
func (f *Fetcher) pageURL(cis string) string {
	return f.cfg.BaseURL + "/" + cis + "/extrait"
}

// waitTurn blocks until the global upstream rate limiter grants a slot,
// aborting early when ctx is cancelled.
func (f *Fetcher) waitTurn(ctx context.Context) error {
	if f.limiter == nil {
		return nil
	}
	d := f.limiter.Take(1)
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("ansmdocs: cancelled while waiting for the upstream rate limiter: %w", ctx.Err())
	}
}

// retryableStatus reports whether an HTTP status is worth a retry.
func retryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// parseRetryAfter parses a Retry-After header value, accepting both the
// delay-seconds and the HTTP-date forms. Unusable values yield 0.
func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if secs, err := strconv.Atoi(value); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}

// sleepContext sleeps for d, aborting early when ctx is cancelled.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("ansmdocs: cancelled during retry backoff: %w", ctx.Err())
	}
}

// validateKey rejects (cis, docType) pairs that must never reach the
// network.
func validateKey(cis, docType string) error {
	if !validCISRe.MatchString(cis) {
		return fmt.Errorf("ansmdocs: invalid cis %q: must be 1-10 digits", cis)
	}
	if docType != TypeRCP && docType != TypeNotice {
		return fmt.Errorf("ansmdocs: invalid docType %q (want %q or %q)", docType, TypeRCP, TypeNotice)
	}
	return nil
}
