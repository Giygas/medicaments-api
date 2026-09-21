package docstore

import (
	"sync"
	"time"
)

// DocumentMeta holds the in-memory index metadata for one (cis, docType)
// pair. It is rebuilt from disk at startup and updated on every write.
type DocumentMeta struct {
	// SHA256 is the hex-encoded sha256 digest of the uncompressed document
	// JSON. The HTTP layer uses it as a strong ETag validator.
	SHA256 string
	// SourceDate is the ANSM "Mis à jour le" date of the document
	// (YYYY-MM-DD). It is empty when the stored JSON does not carry one.
	SourceDate string
	// FetchedAt is the local timestamp at which the document (or the
	// tombstone decision) was cached.
	FetchedAt time.Time
	// Size is the compressed size of the document file on disk, in bytes.
	// It is always 0 for tombstones.
	Size int64
	// Tombstone reports that no document exists upstream (negative cache)
	// and the pair must never be fetched again.
	Tombstone bool
}

// Stats describes the current contents of the store. It feeds the metrics
// gauges and the /v1/diagnostics endpoint.
type Stats struct {
	Documents  int   // number of cached documents
	Tombstones int   // number of negative-cache markers
	TotalBytes int64 // sum of compressed document sizes (tombstones excluded)
}

// docKey uniquely identifies a cached document.
type docKey struct {
	cis     string
	docType string
}

// docIndex is the concurrency-safe in-memory index mapping (cis, docType)
// keys to their metadata. Entries are stored by value and returned as copies
// so readers never share mutable state with writers.
type docIndex struct {
	mu      sync.RWMutex
	entries map[docKey]DocumentMeta
}

// newDocIndex returns an empty index.
func newDocIndex() *docIndex {
	return &docIndex{entries: make(map[docKey]DocumentMeta)}
}

// get returns a copy of the metadata stored for the given key, and whether
// an entry exists at all.
func (i *docIndex) get(cis, docType string) (DocumentMeta, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	meta, ok := i.entries[docKey{cis, docType}]
	return meta, ok
}

// set stores the metadata for the given key, replacing any previous entry.
func (i *docIndex) set(cis, docType string, meta DocumentMeta) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.entries[docKey{cis, docType}] = meta
}

// remove deletes the entry for the given key, if present.
func (i *docIndex) remove(cis, docType string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.entries, docKey{cis, docType})
}

// stats computes the store counters from the current index contents.
func (i *docIndex) stats() Stats {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var st Stats
	for _, meta := range i.entries {
		if meta.Tombstone {
			st.Tombstones++
			continue
		}
		st.Documents++
		st.TotalBytes += meta.Size
	}
	return st
}
