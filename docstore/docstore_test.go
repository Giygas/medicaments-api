package docstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/logging"
)

// init initializes the logger for all tests in this package.
// Uses InitLoggerWithEnvironment to initialize with test environment settings.
// This ensures console logs are ERROR-only and no log files are created.
func init() {
	logging.InitLoggerWithEnvironment("", config.EnvTest, "", 4, 100*1024*1024)
}

// newTestStoreAt returns a DocumentStore backed by the given directory.
func newTestStoreAt(t *testing.T, dir string) *DocumentStore {
	t.Helper()
	s, err := NewDocumentStore(dir)
	if err != nil {
		t.Fatalf("NewDocumentStore(%q) error = %v", dir, err)
	}
	return s
}

// newTestStore returns a DocumentStore backed by a fresh temp directory.
func newTestStore(t *testing.T) *DocumentStore {
	t.Helper()
	return newTestStoreAt(t, t.TempDir())
}

// testDocument builds a realistic final processed document payload with the
// canonical schema served by the API (including the "miseAJour" field the
// startup scan relies on to recover source dates).
func testDocument(t *testing.T, cis, docType, miseAJour string) []byte {
	t.Helper()
	doc := map[string]any{
		"cis":       cis,
		"type":      docType,
		"titre":     "CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable",
		"miseAJour": miseAJour,
		"source":    "ANSM - Base de données publique des médicaments",
		"sections": []map[string]string{
			{"id": "1", "titre": "DÉNOMINATION DU MÉDICAMENT", "contenu": "<p>CARVEDILOL VIATRIS 25 mg, comprimé.</p>"},
			{"id": "4.2", "titre": "Posologie et mode d'administration", "contenu": "<p>Voie orale.</p>"},
		},
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal test document: %v", err)
	}
	return data
}

// countFilesWithSuffix counts regular files in dir whose name ends with suffix.
func countFilesWithSuffix(t *testing.T, dir, suffix string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read cache dir %s: %v", dir, err)
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), suffix) {
			count++
		}
	}
	return count
}

func TestPutGetRoundtrip(t *testing.T) {
	tests := []struct {
		name       string
		cis        string
		docType    string
		sourceDate string
	}{
		{"rcp document", "60016308", DocTypeRCP, "2025-11-07"},
		{"notice document", "60016308", DocTypeNotice, "2024-02-01"},
		{"another cis", "12345678", DocTypeRCP, "2026-01-15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := newTestStore(t)
			want := testDocument(t, tt.cis, tt.docType, tt.sourceDate)

			// Act
			if err := store.Put(tt.cis, tt.docType, want, tt.sourceDate); err != nil {
				t.Fatalf("Put() error = %v", err)
			}
			got, meta, err := store.Get(tt.cis, tt.docType)

			// Assert
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Get() payload mismatch: got %d bytes, want %d bytes", len(got), len(want))
			}
			if meta == nil {
				t.Fatal("Get() meta = nil, want metadata")
			}
			if wantSum := sha256Hex(want); meta.SHA256 != wantSum {
				t.Errorf("meta.SHA256 = %q, want %q", meta.SHA256, wantSum)
			}
			if meta.SourceDate != tt.sourceDate {
				t.Errorf("meta.SourceDate = %q, want %q", meta.SourceDate, tt.sourceDate)
			}
			if meta.Tombstone {
				t.Error("meta.Tombstone = true, want false")
			}
			if meta.FetchedAt.IsZero() {
				t.Error("meta.FetchedAt is zero, want a timestamp")
			}

			// On-disk file is gzipped at the exact expected path.
			raw, err := os.ReadFile(store.docPath(tt.cis, tt.docType))
			if err != nil {
				t.Fatalf("cached file missing: %v", err)
			}
			if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
				t.Error("cached file does not start with gzip magic bytes")
			}
			if meta.Size != int64(len(raw)) {
				t.Errorf("meta.Size = %d, want %d (compressed size on disk)", meta.Size, len(raw))
			}
			decompressed, err := decompress(raw)
			if err != nil {
				t.Fatalf("cached file is not valid gzip: %v", err)
			}
			if !bytes.Equal(decompressed, want) {
				t.Error("decompressed cached file does not match original payload")
			}
		})
	}
}

func TestGetMiss(t *testing.T) {
	tests := []struct {
		name       string
		putCis     string
		getCis     string
		putDocType string
		getDocType string
	}{
		{"empty store", "", "60016308", DocTypeRCP, DocTypeRCP},
		{"other cis", "60016308", "99999999", DocTypeRCP, DocTypeRCP},
		{"other docType", "60016308", "60016308", DocTypeRCP, DocTypeNotice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := newTestStore(t)
			if tt.putCis != "" {
				if err := store.Put(tt.putCis, tt.putDocType, testDocument(t, tt.putCis, tt.putDocType, "2025-11-07"), "2025-11-07"); err != nil {
					t.Fatalf("Put() error = %v", err)
				}
			}

			// Act
			got, meta, err := store.Get(tt.getCis, tt.getDocType)

			// Assert
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("Get() error = %v, want ErrNotFound", err)
			}
			if got != nil {
				t.Errorf("Get() payload = %d bytes, want nil", len(got))
			}
			if meta != nil {
				t.Errorf("Get() meta = %+v, want nil", meta)
			}
			if errors.Is(err, ErrTombstone) {
				t.Error("miss must not be reported as tombstone")
			}
		})
	}
}

func TestPutTombstoneLookup(t *testing.T) {
	// Arrange
	store := newTestStore(t)
	const cis, docType = "60016308", DocTypeNotice

	// Act
	if err := store.PutTombstone(cis, docType); err != nil {
		t.Fatalf("PutTombstone() error = %v", err)
	}

	// Assert: marker file exists at the exact expected path.
	if _, err := os.Stat(store.tombstonePath(cis, docType)); err != nil {
		t.Fatalf("tombstone marker file missing: %v", err)
	}

	// Get reports a tombstone, NOT a miss.
	got, meta, err := store.Get(cis, docType)
	if !errors.Is(err, ErrTombstone) {
		t.Errorf("Get() error = %v, want ErrTombstone", err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("tombstone must not be reported as miss")
	}
	if got != nil {
		t.Errorf("Get() payload = %d bytes, want nil", len(got))
	}
	if meta == nil {
		t.Fatal("Get() meta = nil, want tombstone metadata")
	}
	if !meta.Tombstone {
		t.Error("meta.Tombstone = false, want true")
	}
	if meta.FetchedAt.IsZero() {
		t.Error("meta.FetchedAt is zero, want a timestamp")
	}

	// Writing the same tombstone again is idempotent.
	if err := store.PutTombstone(cis, docType); err != nil {
		t.Errorf("second PutTombstone() error = %v, want nil (idempotent)", err)
	}
	st := store.Stats()
	if st.Tombstones != 1 {
		t.Errorf("Stats().Tombstones = %d, want 1", st.Tombstones)
	}
	if st.Documents != 0 {
		t.Errorf("Stats().Documents = %d, want 0", st.Documents)
	}
}

func TestPutTombstoneOverDocumentRefused(t *testing.T) {
	// Arrange: a real document is already cached.
	store := newTestStore(t)
	const cis, docType = "60016308", DocTypeRCP
	if err := store.Put(cis, docType, testDocument(t, cis, docType, "2025-11-07"), "2025-11-07"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Act
	err := store.PutTombstone(cis, docType)

	// Assert
	if !errors.Is(err, ErrDocumentExists) {
		t.Errorf("PutTombstone() error = %v, want ErrDocumentExists", err)
	}
	if _, _, gerr := store.Get(cis, docType); gerr != nil {
		t.Errorf("Get() error = %v, want document still readable", gerr)
	}
	if _, serr := os.Stat(store.tombstonePath(cis, docType)); !os.IsNotExist(serr) {
		t.Error("tombstone marker must not be created when a document exists")
	}
	st := store.Stats()
	if st.Documents != 1 || st.Tombstones != 0 {
		t.Errorf("Stats() = %+v, want 1 document and 0 tombstones", st)
	}
}

func TestPutReplacesTombstone(t *testing.T) {
	// Arrange
	store := newTestStore(t)
	const cis, docType = "60016308", DocTypeRCP
	if err := store.PutTombstone(cis, docType); err != nil {
		t.Fatalf("PutTombstone() error = %v", err)
	}

	// Act: a real document arrives for the tombstoned key.
	want := testDocument(t, cis, docType, "2025-11-07")
	if err := store.Put(cis, docType, want, "2025-11-07"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Assert: the document is served and the tombstone is fully cleared.
	got, meta, err := store.Get(cis, docType)
	if err != nil {
		t.Fatalf("Get() error = %v, want hit", err)
	}
	if !bytes.Equal(got, want) {
		t.Error("Get() payload does not match stored document")
	}
	if meta.Tombstone {
		t.Error("meta.Tombstone = true, want false")
	}
	if _, serr := os.Stat(store.tombstonePath(cis, docType)); !os.IsNotExist(serr) {
		t.Error("tombstone marker file must be removed when superseded by a document")
	}
	st := store.Stats()
	if st.Documents != 1 || st.Tombstones != 0 {
		t.Errorf("Stats() = %+v, want 1 document and 0 tombstones", st)
	}
}

func TestPutOverwritesDocument(t *testing.T) {
	// Arrange
	store := newTestStore(t)
	const cis, docType = "60016308", DocTypeRCP
	v1 := testDocument(t, cis, docType, "2024-01-01")
	if err := store.Put(cis, docType, v1, "2024-01-01"); err != nil {
		t.Fatalf("Put(v1) error = %v", err)
	}

	// Act: re-put with newer content and source date.
	v2 := testDocument(t, cis, docType, "2025-11-07")
	if err := store.Put(cis, docType, v2, "2025-11-07"); err != nil {
		t.Fatalf("Put(v2) error = %v", err)
	}

	// Assert: latest content wins, exactly one file on disk.
	got, meta, err := store.Get(cis, docType)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !bytes.Equal(got, v2) {
		t.Error("Get() payload does not match latest version")
	}
	if meta.SourceDate != "2025-11-07" {
		t.Errorf("meta.SourceDate = %q, want %q", meta.SourceDate, "2025-11-07")
	}
	if wantSum := sha256Hex(v2); meta.SHA256 != wantSum {
		t.Errorf("meta.SHA256 = %q, want %q", meta.SHA256, wantSum)
	}
	if n := countFilesWithSuffix(t, store.dir, docsExt); n != 1 {
		t.Errorf("found %d document files in cache dir, want exactly 1 (no temp leftovers)", n)
	}
	st := store.Stats()
	if st.Documents != 1 {
		t.Errorf("Stats().Documents = %d, want 1", st.Documents)
	}
}

func TestStartupScanRebuildsIndex(t *testing.T) {
	// Arrange: pre-populate a cache directory via a first store instance,
	// then add junk files directly on disk.
	dir := t.TempDir()
	s1 := newTestStoreAt(t, dir)

	docA := testDocument(t, "60016308", DocTypeRCP, "2025-11-07")
	if err := s1.Put("60016308", DocTypeRCP, docA, "2025-11-07"); err != nil {
		t.Fatalf("Put(A) error = %v", err)
	}
	docB := testDocument(t, "12345678", DocTypeNotice, "2024-02-01")
	if err := s1.Put("12345678", DocTypeNotice, docB, "2024-02-01"); err != nil {
		t.Fatalf("Put(B) error = %v", err)
	}
	if err := s1.PutTombstone("99999999", DocTypeRCP); err != nil {
		t.Fatalf("PutTombstone(C) error = %v", err)
	}
	// Corrupt document and unrelated file: both must be skipped by the scan.
	if err := os.WriteFile(filepath.Join(dir, "77777777_rcp"+docsExt), []byte("not a gzip stream"), filePerm); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("junk"), filePerm); err != nil {
		t.Fatalf("failed to write junk file: %v", err)
	}

	// Act: "restart" — build a second store over the same directory.
	s2 := newTestStoreAt(t, dir)

	// Assert: counters come from the rebuilt index.
	st := s2.Stats()
	if st.Documents != 2 {
		t.Errorf("Stats().Documents = %d, want 2", st.Documents)
	}
	if st.Tombstones != 1 {
		t.Errorf("Stats().Tombstones = %d, want 1", st.Tombstones)
	}

	// Document A: full metadata rebuilt (sha, source date recovered from the
	// JSON body, size from disk).
	got, meta, err := s2.Get("60016308", DocTypeRCP)
	if err != nil {
		t.Fatalf("Get(A) error = %v, want hit", err)
	}
	if !bytes.Equal(got, docA) {
		t.Error("Get(A) payload mismatch after restart")
	}
	if wantSum := sha256Hex(docA); meta.SHA256 != wantSum {
		t.Errorf("meta.SHA256 = %q, want %q", meta.SHA256, wantSum)
	}
	if meta.SourceDate != "2025-11-07" {
		t.Errorf("meta.SourceDate = %q, want %q recovered from document JSON", meta.SourceDate, "2025-11-07")
	}
	if meta.FetchedAt.IsZero() {
		t.Error("meta.FetchedAt is zero, want file modification time")
	}
	if wantSize := int64(len(readFileT(t, s2.docPath("60016308", DocTypeRCP)))); meta.Size != wantSize {
		t.Errorf("meta.Size = %d, want %d", meta.Size, wantSize)
	}

	// Document B is readable after restart.
	if _, _, err := s2.Get("12345678", DocTypeNotice); err != nil {
		t.Errorf("Get(B) error = %v, want hit", err)
	}

	// Tombstone C survives the restart and is reported as such.
	_, tmeta, err := s2.Get("99999999", DocTypeRCP)
	if !errors.Is(err, ErrTombstone) {
		t.Errorf("Get(C) error = %v, want ErrTombstone", err)
	}
	if tmeta == nil || !tmeta.Tombstone {
		t.Error("tombstone metadata missing after restart")
	}
	if tmeta != nil && tmeta.FetchedAt.IsZero() {
		t.Error("tombstone meta.FetchedAt is zero, want recorded timestamp")
	}

	// The corrupt document is skipped: it reads as a miss so it can be
	// re-fetched and healed.
	if _, _, err := s2.Get("77777777", DocTypeRCP); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(corrupt) error = %v, want ErrNotFound", err)
	}

	// TotalBytes is the sum of the two valid document file sizes.
	wantBytes := int64(len(readFileT(t, s2.docPath("60016308", DocTypeRCP)))) +
		int64(len(readFileT(t, s2.docPath("12345678", DocTypeNotice))))
	if st.TotalBytes != wantBytes {
		t.Errorf("Stats().TotalBytes = %d, want %d", st.TotalBytes, wantBytes)
	}
}

func TestStartupScanDocumentWinsOverMarker(t *testing.T) {
	// Arrange: a valid document plus a stray marker file for the same key.
	dir := t.TempDir()
	s1 := newTestStoreAt(t, dir)
	const cis, docType = "11111111", DocTypeRCP
	if err := s1.Put(cis, docType, testDocument(t, cis, docType, "2025-11-07"), "2025-11-07"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	marker, err := json.Marshal(tombstoneMarker{CIS: cis, DocType: docType, RecordedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("failed to marshal marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, cis+"_"+docType+tombstoneExt), marker, filePerm); err != nil {
		t.Fatalf("failed to write marker: %v", err)
	}

	// Act
	s2 := newTestStoreAt(t, dir)

	// Assert: the live document takes precedence over the marker.
	if _, _, err := s2.Get(cis, docType); err != nil {
		t.Errorf("Get() error = %v, want document to win over marker", err)
	}
	st := s2.Stats()
	if st.Documents != 1 || st.Tombstones != 0 {
		t.Errorf("Stats() = %+v, want 1 document and 0 tombstones", st)
	}
}

func TestGetSelfHealsBrokenFiles(t *testing.T) {
	tests := []struct {
		name      string
		breakFile func(t *testing.T, path string)
	}{
		{
			name: "file deleted behind the store",
			breakFile: func(t *testing.T, path string) {
				if err := os.Remove(path); err != nil {
					t.Fatalf("failed to remove file: %v", err)
				}
			},
		},
		{
			name: "file corrupted behind the store",
			breakFile: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte("garbage bytes"), filePerm); err != nil {
					t.Fatalf("failed to corrupt file: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := newTestStore(t)
			const cis, docType = "60016308", DocTypeRCP
			if err := store.Put(cis, docType, testDocument(t, cis, docType, "2025-11-07"), "2025-11-07"); err != nil {
				t.Fatalf("Put() error = %v", err)
			}

			// Act: break the file behind the store's back, then read.
			tt.breakFile(t, store.docPath(cis, docType))
			got, _, err := store.Get(cis, docType)

			// Assert: reported as a miss, index purged, and re-caching works.
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("Get() error = %v, want ErrNotFound", err)
			}
			if got != nil {
				t.Errorf("Get() payload = %d bytes, want nil", len(got))
			}
			if st := store.Stats(); st.Documents != 0 {
				t.Errorf("Stats().Documents = %d, want 0 after self-heal", st.Documents)
			}
			want := testDocument(t, cis, docType, "2026-01-15")
			if err := store.Put(cis, docType, want, "2026-01-15"); err != nil {
				t.Fatalf("re-cache Put() error = %v", err)
			}
			if _, _, err := store.Get(cis, docType); err != nil {
				t.Errorf("Get() after re-cache error = %v, want hit", err)
			}
		})
	}
}

func TestWriteFileAtomic(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	target := filepath.Join(dir, "60016308_rcp"+docsExt)

	// Act
	if err := writeFileAtomic(dir, target, []byte("v1 payload")); err != nil {
		t.Fatalf("writeFileAtomic() error = %v", err)
	}

	// Assert: exact content, exactly one file in the directory (no temp leftovers).
	if got := readFileT(t, target); string(got) != "v1 payload" {
		t.Errorf("file content = %q, want %q", got, "v1 payload")
	}
	if n := len(readDirT(t, dir)); n != 1 {
		t.Errorf("directory contains %d entries, want 1 (no temp files left)", n)
	}

	// Act: overwrite via a second atomic write.
	if err := writeFileAtomic(dir, target, []byte("v2 payload")); err != nil {
		t.Fatalf("writeFileAtomic() overwrite error = %v", err)
	}

	// Assert
	if got := readFileT(t, target); string(got) != "v2 payload" {
		t.Errorf("file content = %q, want %q", got, "v2 payload")
	}
	if n := len(readDirT(t, dir)); n != 1 {
		t.Errorf("directory contains %d entries, want 1 (no temp files left)", n)
	}
}

func TestPutFailsLeavesNoPartialFile(t *testing.T) {
	// Root bypasses file permissions, which would invalidate the simulation.
	if os.Geteuid() == 0 {
		t.Skip("test requires a non-root user (root bypasses file permissions)")
	}

	// Arrange
	dir := t.TempDir()
	store := newTestStoreAt(t, dir)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("failed to make cache dir read-only: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Errorf("failed to restore cache dir permissions: %v", err)
		}
	})

	// Act: the write must fail because temp files cannot be created.
	err := store.Put("60016308", DocTypeRCP, testDocument(t, "60016308", DocTypeRCP, "2025-11-07"), "2025-11-07")
	if err == nil {
		t.Fatal("Put() expected an error with a read-only cache directory")
	}

	// Assert: no partial file, no temp leftovers, index untouched.
	if n := len(readDirT(t, dir)); n != 0 {
		t.Errorf("directory contains %d entries, want 0 (no partial file left)", n)
	}
	st := store.Stats()
	if st.Documents != 0 || st.Tombstones != 0 || st.TotalBytes != 0 {
		t.Errorf("Stats() = %+v, want all zeros after failed Put", st)
	}
	if _, _, gerr := store.Get("60016308", DocTypeRCP); !errors.Is(gerr, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound after failed Put", gerr)
	}
}

func TestStats(t *testing.T) {
	t.Run("empty store", func(t *testing.T) {
		// Arrange
		store := newTestStore(t)

		// Act
		st := store.Stats()

		// Assert
		if st.Documents != 0 || st.Tombstones != 0 || st.TotalBytes != 0 {
			t.Errorf("Stats() = %+v, want all zeros on an empty store", st)
		}
	})

	t.Run("mixed documents and tombstones", func(t *testing.T) {
		// Arrange
		store := newTestStore(t)
		docA := testDocument(t, "60016308", DocTypeRCP, "2025-11-07")
		docB := testDocument(t, "60016308", DocTypeNotice, "2024-02-01")
		if err := store.Put("60016308", DocTypeRCP, docA, "2025-11-07"); err != nil {
			t.Fatalf("Put(A) error = %v", err)
		}
		if err := store.Put("60016308", DocTypeNotice, docB, "2024-02-01"); err != nil {
			t.Fatalf("Put(B) error = %v", err)
		}
		if err := store.PutTombstone("99999999", DocTypeRCP); err != nil {
			t.Fatalf("PutTombstone() error = %v", err)
		}

		// Act
		st := store.Stats()

		// Assert
		if st.Documents != 2 {
			t.Errorf("Stats().Documents = %d, want 2", st.Documents)
		}
		if st.Tombstones != 1 {
			t.Errorf("Stats().Tombstones = %d, want 1", st.Tombstones)
		}
		wantBytes := int64(len(readFileT(t, store.docPath("60016308", DocTypeRCP)))) +
			int64(len(readFileT(t, store.docPath("60016308", DocTypeNotice))))
		if st.TotalBytes != wantBytes {
			t.Errorf("Stats().TotalBytes = %d, want %d (documents only)", st.TotalBytes, wantBytes)
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	// Arrange: precompute payloads on the test goroutine so workers never
	// touch the testing.T instance.
	store := newTestStore(t)
	const workers = 8
	const keys = 16
	docs := make([][]byte, keys)
	for k := range docs {
		cis := fmt.Sprintf("%08d", k)
		docs[k] = testDocument(t, cis, DocTypeRCP, "2025-11-07")
	}

	// Act: all workers hammer the same rcp keys with identical payloads and
	// tombstone the same notice keys, while Stats is polled concurrently.
	var wg sync.WaitGroup
	// 3 slots per iteration guards against a full channel deadlocking workers
	// if several operations were to fail at once.
	errCh := make(chan error, workers*keys*3)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < keys; k++ {
				cis := fmt.Sprintf("%08d", k)
				if err := store.Put(cis, DocTypeRCP, docs[k], "2025-11-07"); err != nil {
					errCh <- fmt.Errorf("Put(%s) error = %w", cis, err)
					continue
				}
				if _, _, err := store.Get(cis, DocTypeRCP); err != nil {
					errCh <- fmt.Errorf("Get(%s) error = %w", cis, err)
				}
				if err := store.PutTombstone(cis, DocTypeNotice); err != nil {
					errCh <- fmt.Errorf("PutTombstone(%s) error = %w", cis, err)
				}
				_ = store.Stats()
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	// Assert: final state is deterministic.
	st := store.Stats()
	if st.Documents != keys {
		t.Errorf("Stats().Documents = %d, want %d", st.Documents, keys)
	}
	if st.Tombstones != keys {
		t.Errorf("Stats().Tombstones = %d, want %d", st.Tombstones, keys)
	}
	for k := 0; k < keys; k++ {
		cis := fmt.Sprintf("%08d", k)
		if _, _, err := store.Get(cis, DocTypeRCP); err != nil {
			t.Errorf("Get(%s, rcp) error = %v, want hit", cis, err)
		}
		if _, _, err := store.Get(cis, DocTypeNotice); !errors.Is(err, ErrTombstone) {
			t.Errorf("Get(%s, notice) error = %v, want ErrTombstone", cis, err)
		}
	}
}

func TestKeyValidation(t *testing.T) {
	tests := []struct {
		name    string
		cis     string
		docType string
	}{
		{"empty cis", "", DocTypeRCP},
		{"non-numeric cis", "abcdef", DocTypeRCP},
		{"path traversal in cis", "../../etc", DocTypeRCP},
		{"cis too long", "12345678901", DocTypeRCP},
		{"underscore in cis", "600_163", DocTypeRCP},
		{"empty docType", "60016308", ""},
		{"unknown docType", "60016308", "pdf"},
		{"path traversal in docType", "60016308", "../evil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := newTestStore(t)

			// Act & Assert: every entrypoint rejects the invalid key with
			// ErrInvalidKey and never touches the filesystem.
			if _, _, err := store.Get(tt.cis, tt.docType); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Get() error = %v, want ErrInvalidKey", err)
			}
			if err := store.Put(tt.cis, tt.docType, []byte("{}"), "2025-11-07"); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Put() error = %v, want ErrInvalidKey", err)
			}
			if err := store.PutTombstone(tt.cis, tt.docType); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("PutTombstone() error = %v, want ErrInvalidKey", err)
			}
			if n := len(readDirT(t, store.dir)); n != 0 {
				t.Errorf("cache dir contains %d entries, want 0 (no file created)", n)
			}
		})
	}
}

func TestPutRejectsEmptyPayload(t *testing.T) {
	// Arrange
	store := newTestStore(t)

	// Act
	err := store.Put("60016308", DocTypeRCP, nil, "2025-11-07")

	// Assert
	if err == nil {
		t.Fatal("Put(nil) expected an error, want refusal to cache empty document")
	}
	if st := store.Stats(); st.Documents != 0 {
		t.Errorf("Stats().Documents = %d, want 0", st.Documents)
	}
}

func TestNewDocumentStoreValidation(t *testing.T) {
	t.Run("empty directory path", func(t *testing.T) {
		// Arrange & Act
		_, err := NewDocumentStore("")

		// Assert
		if err == nil {
			t.Fatal("NewDocumentStore(\"\") expected an error")
		}
	})

	t.Run("directory path is a file", func(t *testing.T) {
		// Arrange
		path := filepath.Join(t.TempDir(), "notadir")
		if err := os.WriteFile(path, []byte("x"), filePerm); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Act
		_, err := NewDocumentStore(path)

		// Assert
		if err == nil {
			t.Fatal("NewDocumentStore(file) expected an error")
		}
	})

	t.Run("nested directory is created", func(t *testing.T) {
		// Arrange
		nested := filepath.Join(t.TempDir(), "files", "docs")

		// Act
		s, err := NewDocumentStore(nested)

		// Assert
		if err != nil {
			t.Fatalf("NewDocumentStore(nested) error = %v", err)
		}
		if info, serr := os.Stat(nested); serr != nil || !info.IsDir() {
			t.Errorf("expected directory %s to be created", nested)
		}
		if st := s.Stats(); st.Documents != 0 || st.Tombstones != 0 {
			t.Errorf("Stats() = %+v, want empty store", st)
		}
	})
}

// readFileT reads a file, failing the test on error.
func readFileT(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return data
}

// readDirT lists a directory, failing the test on error.
func readDirT(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir %s: %v", dir, err)
	}
	return entries
}
