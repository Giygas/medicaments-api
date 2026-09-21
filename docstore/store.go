// Package docstore implements the permanent on-disk cache for ANSM RCP and
// patient notice documents (the final processed sectioned JSON, never raw
// HTML). Documents are stored gzipped as {cis}_{docType}.json.gz with atomic
// writes (temp file + rename in the same directory). Documents known to be
// absent upstream are recorded as tombstone marker files forming a negative
// cache that is never refetched. An in-memory index mapping (cis, docType)
// to metadata is rebuilt by scanning the cache directory at startup.
package docstore

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/giygas/medicaments-api/logging"
)

// Supported document types.
const (
	// DocTypeRCP identifies "Résumé des Caractéristiques du Produit" documents.
	DocTypeRCP = "rcp"
	// DocTypeNotice identifies patient notice documents.
	DocTypeNotice = "notice"
)

const (
	docsExt      = ".json.gz"
	tombstoneExt = ".missing.json"
	tempPattern  = ".tmp-docstore-*"

	dirPerm  os.FileMode = 0o750
	filePerm os.FileMode = 0o640
)

// Sentinel errors returned by DocumentStore operations. Distinguish them
// with errors.Is.
var (
	// ErrNotFound is returned by Get when neither a document nor a tombstone
	// is cached for the requested (cis, docType) pair.
	ErrNotFound = errors.New("docstore: document not found")

	// ErrTombstone is returned by Get when the document is known to be
	// absent upstream (negative cache entry exists).
	ErrTombstone = errors.New("docstore: document marked as missing (tombstone)")

	// ErrInvalidKey is returned when a cis or docType is not safe to embed
	// in a cache filename (path traversal or unsupported type).
	ErrInvalidKey = errors.New("docstore: invalid (cis, docType) key")

	// ErrDocumentExists is returned by PutTombstone when a real document is
	// already cached for the key, refusing to discard valid data.
	ErrDocumentExists = errors.New("docstore: document already cached")
)

// CIS codes are numeric; the length bound keeps filenames well behaved.
// validDocTypes lists the allowed docType values, kept in sync with the
// DocType* constants above.
var (
	validCISRe    = regexp.MustCompile(`^[0-9]{1,10}$`)
	validDocTypes = map[string]struct{}{DocTypeRCP: {}, DocTypeNotice: {}}
)

// DocumentStore is the concrete on-disk document cache. It is safe for
// concurrent use by multiple goroutines.
type DocumentStore struct {
	dir   string
	index *docIndex
}

// NewDocumentStore creates the cache directory if needed and rebuilds the
// in-memory index by scanning it. Files that are corrupt or do not follow
// the naming convention are skipped (and logged), so a damaged cache
// self-heals by treating those documents as misses.
func NewDocumentStore(dir string) (*DocumentStore, error) {
	if dir == "" {
		return nil, errors.New("docstore: cache directory must not be empty")
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("docstore: failed to create cache directory %s: %w", dir, err)
	}

	s := &DocumentStore{dir: dir, index: newDocIndex()}
	if err := s.rebuildIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

// Get returns the decompressed document JSON and its metadata for the given
// (cis, docType) pair. Outcomes are distinguished as follows:
//
//   - hit: returns (payload, meta, nil)
//   - tombstone: returns (nil, meta, ErrTombstone) — never refetch upstream
//   - miss: returns (nil, nil, ErrNotFound)
//   - invalid key or unreadable file: returns (nil, nil, wrapped error)
//
// A document whose file disappeared or became corrupt after the index was
// built is reported as a miss and its index entry dropped, so the caller can
// fetch and re-cache it (self-healing).
func (s *DocumentStore) Get(cis, docType string) ([]byte, *DocumentMeta, error) {
	if err := validateKey(cis, docType); err != nil {
		return nil, nil, err
	}

	meta, ok := s.index.get(cis, docType)
	if !ok {
		return nil, nil, ErrNotFound
	}
	if meta.Tombstone {
		return nil, &meta, ErrTombstone
	}

	raw, err := os.ReadFile(s.docPath(cis, docType))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.index.remove(cis, docType)
			logging.Warn("Indexed document vanished from disk, dropping index entry",
				"cis", cis, "docType", docType)
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("docstore: failed to read cached document %s: %w", s.docPath(cis, docType), err)
	}

	payload, err := decompress(raw)
	if err != nil {
		s.index.remove(cis, docType)
		logging.Error("Cached document is corrupt, dropping index entry",
			"cis", cis, "docType", docType, "error", err)
		return nil, nil, ErrNotFound
	}
	return payload, &meta, nil
}

// Put caches the final processed document JSON for the given (cis, docType)
// pair: it computes the content sha256, gzips the payload, writes it
// atomically (temp file + rename in the same directory) and updates the
// in-memory index. A previously recorded tombstone for the key is superseded
// because a real document always takes precedence.
func (s *DocumentStore) Put(cis, docType string, jsonBytes []byte, sourceDate string) error {
	if err := validateKey(cis, docType); err != nil {
		return err
	}
	if len(jsonBytes) == 0 {
		return errors.New("docstore: refusing to cache empty document")
	}

	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(jsonBytes); err != nil {
		return fmt.Errorf("docstore: failed to compress document: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("docstore: failed to finalize compressed document: %w", err)
	}

	meta := DocumentMeta{
		SHA256:     sha256Hex(jsonBytes),
		SourceDate: sourceDate,
		FetchedAt:  time.Now().UTC(),
		Size:       int64(compressed.Len()),
	}
	if err := writeFileAtomic(s.dir, s.docPath(cis, docType), compressed.Bytes()); err != nil {
		return err
	}

	// The document supersedes any tombstone marker recorded for this key;
	// remove the marker so a restart does not see conflicting entries.
	if existing, ok := s.index.get(cis, docType); ok && existing.Tombstone {
		if err := os.Remove(s.tombstonePath(cis, docType)); err != nil && !errors.Is(err, os.ErrNotExist) {
			logging.Warn("Failed to remove superseded tombstone marker",
				"cis", cis, "docType", docType, "error", err)
		}
	}

	s.index.set(cis, docType, meta)
	logging.Debug("Document cached",
		"cis", cis, "docType", docType, "bytes", meta.Size, "sha256", meta.SHA256)
	return nil
}

// PutTombstone records that no document exists upstream for the given
// (cis, docType) pair (negative cache) so it is never fetched again. Writing
// a tombstone over an already cached document is refused with
// ErrDocumentExists (that would discard valid data); writing a tombstone
// that already exists is an idempotent no-op.
func (s *DocumentStore) PutTombstone(cis, docType string) error {
	if err := validateKey(cis, docType); err != nil {
		return err
	}
	if existing, ok := s.index.get(cis, docType); ok {
		if !existing.Tombstone {
			return fmt.Errorf("%w: cis %s docType %s", ErrDocumentExists, cis, docType)
		}
		return nil
	}

	now := time.Now().UTC()
	marker, err := json.Marshal(tombstoneMarker{CIS: cis, DocType: docType, RecordedAt: now})
	if err != nil {
		return fmt.Errorf("docstore: failed to encode tombstone marker: %w", err)
	}
	if err := writeFileAtomic(s.dir, s.tombstonePath(cis, docType), marker); err != nil {
		return err
	}

	s.index.set(cis, docType, DocumentMeta{FetchedAt: now, Tombstone: true})
	logging.Debug("Tombstone recorded", "cis", cis, "docType", docType)
	return nil
}

// Stats returns the current store counters (documents, tombstones, total
// compressed bytes). It is used by the metrics gauges and /v1/diagnostics.
func (s *DocumentStore) Stats() Stats {
	return s.index.stats()
}

// rebuildIndex scans the cache directory and rebuilds the in-memory index.
// A live document always wins over a tombstone marker for the same key.
func (s *DocumentStore) rebuildIndex() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("docstore: failed to scan cache directory %s: %w", s.dir, err)
	}

	var skipped int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(s.dir, name)

		switch {
		case strings.HasSuffix(name, docsExt):
			cis, docType, err := splitStem(strings.TrimSuffix(name, docsExt))
			if err != nil {
				skipped++
				logging.Warn("Ignoring file with unrecognized name in docs cache", "file", name, "reason", err)
				continue
			}
			meta, err := scanDocument(path)
			if err != nil {
				skipped++
				logging.Error("Ignoring corrupt cached document", "file", name, "error", err)
				continue
			}
			s.index.set(cis, docType, meta)

		case strings.HasSuffix(name, tombstoneExt):
			cis, docType, err := splitStem(strings.TrimSuffix(name, tombstoneExt))
			if err != nil {
				skipped++
				logging.Warn("Ignoring marker with unrecognized name in docs cache", "file", name, "reason", err)
				continue
			}
			// Never let a marker shadow an already indexed live document.
			if existing, ok := s.index.get(cis, docType); ok && !existing.Tombstone {
				logging.Warn("Ignoring tombstone marker because a document is already cached",
					"cis", cis, "docType", docType, "file", name)
				continue
			}
			meta, err := scanTombstone(path)
			if err != nil {
				skipped++
				logging.Error("Ignoring unreadable tombstone marker", "file", name, "error", err)
				continue
			}
			s.index.set(cis, docType, meta)

		default:
			logging.Debug("Ignoring unrecognized file in docs cache", "file", name)
		}
	}

	st := s.index.stats()
	logging.Info("Docs cache index rebuilt",
		"directory", s.dir,
		"documents", st.Documents,
		"tombstones", st.Tombstones,
		"totalBytes", st.TotalBytes,
		"skipped", skipped)
	return nil
}

// docPath returns the cache path of a document file for a validated key.
func (s *DocumentStore) docPath(cis, docType string) string {
	return filepath.Join(s.dir, cis+"_"+docType+docsExt)
}

// tombstonePath returns the cache path of a tombstone marker file for a
// validated key.
func (s *DocumentStore) tombstonePath(cis, docType string) string {
	return filepath.Join(s.dir, cis+"_"+docType+tombstoneExt)
}

// tombstoneMarker is the JSON body written into {cis}_{docType}.missing.json
// files so the negative-cache decision survives restarts.
type tombstoneMarker struct {
	CIS        string    `json:"cis"`
	DocType    string    `json:"docType"`
	RecordedAt time.Time `json:"recordedAt"`
}

// documentEnvelope extracts the fields the store needs from the stored
// document JSON. The cached payload is the final processed document whose
// canonical schema includes "miseAJour" (ANSM update date), which is how the
// startup scan recovers the source date of previously cached documents.
type documentEnvelope struct {
	MiseAJour string `json:"miseAJour"`
}

// scanDocument reads a cached gzip document, verifies its integrity, and
// derives its index metadata: sha256 of the uncompressed JSON, source date
// extracted from the document itself, compressed size and file modification
// time as the fetch timestamp.
func scanDocument(path string) (DocumentMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return DocumentMeta{}, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			logging.Warn("Failed to close cached document file", "file", path, "error", cerr)
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return DocumentMeta{}, fmt.Errorf("failed to stat file: %w", err)
	}

	payload, err := decompressFile(f)
	if err != nil {
		return DocumentMeta{}, err
	}
	var env documentEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return DocumentMeta{}, fmt.Errorf("invalid document JSON: %w", err)
	}

	return DocumentMeta{
		SHA256:     sha256Hex(payload),
		SourceDate: env.MiseAJour,
		FetchedAt:  info.ModTime(),
		Size:       info.Size(),
	}, nil
}

// scanTombstone reads a tombstone marker file and derives its index
// metadata. The recorded timestamp is read from the marker body, falling
// back to the file modification time for manually created markers.
func scanTombstone(path string) (DocumentMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DocumentMeta{}, err
	}
	var marker tombstoneMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return DocumentMeta{}, fmt.Errorf("invalid tombstone marker JSON: %w", err)
	}

	fetchedAt := marker.RecordedAt
	if fetchedAt.IsZero() {
		if info, err := os.Stat(path); err == nil {
			fetchedAt = info.ModTime()
		}
	}
	return DocumentMeta{FetchedAt: fetchedAt, Tombstone: true}, nil
}

// decompress gunzips an in-memory compressed payload.
func decompress(compressed []byte) ([]byte, error) {
	return decompressFile(bytes.NewReader(compressed))
}

// decompressFile gunzips a gzip stream read from an open reader.
func decompressFile(r io.Reader) ([]byte, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("corrupt gzip stream: %w", err)
	}
	data, err := io.ReadAll(zr)
	if rerr := zr.Close(); rerr != nil && err == nil {
		err = rerr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read gzip stream: %w", err)
	}
	return data, nil
}

// writeFileAtomic durably writes data to the target path via a temp file in
// the SAME directory followed by a rename. Readers therefore never observe
// partial or corrupt files: the target either has its previous content or
// the complete new content. On failure the temp file is removed.
func writeFileAtomic(dir, path string, data []byte) error {
	tmp, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return fmt.Errorf("docstore: failed to create temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()

	// cleanup is the failure path: close (a second close after a successful
	// one is a harmless error) and remove the temp file so no partial data
	// is ever left behind.
	cleanup := func() {
		if cerr := tmp.Close(); cerr != nil {
			logging.Warn("Failed to close temp file after error", "file", tmpName, "error", cerr)
		}
		if rerr := os.Remove(tmpName); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			logging.Warn("Failed to remove temp file after error", "file", tmpName, "error", rerr)
		}
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("docstore: failed to write temp file %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("docstore: failed to sync temp file %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("docstore: failed to close temp file %s: %w", tmpName, err)
	}
	if err := os.Chmod(tmpName, filePerm); err != nil {
		cleanup()
		return fmt.Errorf("docstore: failed to chmod temp file %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("docstore: failed to rename %s to %s: %w", tmpName, path, err)
	}
	return nil
}

// splitStem splits a cache filename stem ("{cis}_{docType}") into its
// components and validates them. docType is the segment after the last
// underscore; CIS codes never contain underscores.
func splitStem(stem string) (cis, docType string, err error) {
	i := strings.LastIndex(stem, "_")
	if i < 0 {
		return "", "", fmt.Errorf("missing docType segment in %q", stem)
	}
	cis, docType = stem[:i], stem[i+1:]
	if err := validateKey(cis, docType); err != nil {
		return "", "", err
	}
	return cis, docType, nil
}

// validateKey ensures a (cis, docType) pair is safe to embed in a filename,
// rejecting path traversal and unsupported document types.
func validateKey(cis, docType string) error {
	if !validCISRe.MatchString(cis) {
		return fmt.Errorf("%w: cis must be 1-10 digits, got %q", ErrInvalidKey, cis)
	}
	if _, ok := validDocTypes[docType]; !ok {
		return fmt.Errorf("%w: docType must be %q or %q, got %q", ErrInvalidKey, DocTypeRCP, DocTypeNotice, docType)
	}
	return nil
}

// sha256Hex returns the hex-encoded sha256 digest of data.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
