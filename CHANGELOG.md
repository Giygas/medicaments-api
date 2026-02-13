# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-02-13

### Added
- **RESTful v1 API** with 9 new endpoints using path-based routing
  - Medicament lookup by CIS, CIP, or multi-word search
  - Presentation and generique group endpoints
  - `/v1/diagnostics` for system metrics and data quality reports
- **Prometheus metrics** on port 9090 (request counter, duration histogram, in-flight gauge)
- **Multi-word search** with AND logic (up to 6 words)
- **ETag caching** with SHA256-based validation
- **Sequential log numbering** to prevent unbounded growth (`_01`, `_02`, etc.)
- **Orphaned presentations tracking** in `/v1/diagnostics`
  - Shows CIP codes for presentations with non-existent CIS

### Changed
- **Pre-computed normalized names**: 5x faster search, 170x fewer allocations
  - Medicaments search: 3,500ns → 750ns (4.7x faster)
  - Generiques search: 3,500ns → 75ns (46.7x faster)
- **Input validation optimization**: 5-10x faster via string.Contains() and regex pre-compilation
- **Response writer pooling** reduces allocations by 2.2MB/sec at high throughput
- **Fast-path logging** for /health and /metrics endpoints
- **LOG_LEVEL now functional** with environment-based fallback (console only)
- **Endpoint costs updated** for rate limiting (5-200 tokens per request)
- **Go version requirement**: 1.21 → 1.24+ (latest stable)

### Deprecated
- **Legacy API endpoints** (Sunset date: 2026-07-31)
  - `/database` → Use `/v1/medicaments/export`
  - `/database/{page}` → Use `/v1/medicaments?page={n}`
  - `/medicament/{nom}` → Use `/v1/medicaments?search={nom}`
  - `/medicament/id/{cis}` → Use `/v1/medicaments/{cis}`
  - `/medicament/cip/{cip}` → Use `/v1/medicaments?cip={cip}`
  - `/generiques/{libelle}` → Use `/v1/generiques?libelle={libelle}`
  - `/generiques/group/{id}` → Use `/v1/generiques/{id}`

**Deprecation headers returned**:
```
Deprecation: true
Sunset: 2026-07-31T23:59:59Z
Link: </v1/...>; rel="successor-version"
Warning: 299 - "Deprecated endpoint..."
```

### Fixed
- **Race conditions in rotating logger** (resource leaks + concurrency issues)
- **/v1/medicaments returns 404** when not found (instead of empty array)
- **Validation genériques**: groupID range 1-9999 with clear error messages
- **ASCII-only input validation** with helpful rejection messages for accented characters
- **Server shutdown logging** fixed
- **TSV edge case handling** with skip statistics for malformed lines
- **Validation off-by-one bug** corrected
- **Charset encoding**: Automatic UTF-8/ISO8859-1 detection in downloader
- **HTTP timeout and scanner error handling** (5-minute download timeout for BDPM files,
  1MB scanner buffer for robust parsing, error checking after each file)
- **Graceful shutdown for metrics/profiling servers** (context cancellation,
  prevents goroutine leaks, cleaner shutdowns with 5-second timeout)

### Performance
**HTTP throughput improvements across all endpoints**:

| Endpoint                  | Before    | After      | Improvement |
|---------------------------|-----------|------------|-------------|
| `/v1/presentations/{cip}`  | 35K req/s | 77K req/s  | +120%       |
| `/v1/medicaments/{cis}`    | 30K req/s | 78K req/s  | +160%       |
| `/v1/medicaments?cip={c}`  | 35K req/s | 75K req/s  | +114%       |
| `/v1/medicaments?page={n}` | 20K req/s | 41K req/s  | +105%       |
| `/v1/generiques?libelle={nom}` | 20K req/s | 36K req/s | +80%    |
| `/v1/medicaments?search={q}` | 5K req/s | 6.1K req/s | +22%     |
| `/health`                  | 30K req/s | 92K req/s  | +207%       |

**Memory footprint**: 55-80MB stable (67.5MB median)

**Algorithmic improvements**:
- Medicaments search: 250 → 1,250 req/s (5x)
- Generiques search: 1,500 → 15,000 req/s (10x)
- Allocations per search: 16,000 → 94 (170x reduction)

### Security
- **Input validation pattern**: `^[a-zA-Z0-9\s\-\.\+']+$` (ASCII-only)
  - Rejects accented characters with helpful error message
  - Supports alphanumeric + spaces + hyphen/apostrophe/period/plus sign
- **Multi-word search limit**: Maximum 6 words (DoS prevention)
- **Variable rate limiting**: 5-200 tokens per endpoint (1,000 tokens, 3/sec recharge)
- **Dangerous pattern detection**: SQL injection, XSS, command injection, path traversal (5-10x faster than regex)
- **Direct CIP/CIS validation** via strconv.Atoi() without regex

### Breaking Changes

**1. Generique group response structure** (MAJOR)

Before (`GET /generiques/group/{id}`):
```json
{
  "cis": 12345678,
  "group": 100,
  "libelle": "Paracétamol",
  "type": "princeps"
}
```

After (`GET /v1/generiques/{groupID}`):
```json
{
  "groupID": 100,
  "libelle": "Paracétamol",
  "medicaments": [
    {
      "cis": 12345678,
      "elementPharmaceutique": "PARACETAMOL 500 mg, comprimé",
      "formePharmaceutique": "Comprimé",
      "type": "princeps",
      "composition": [...]
    }
  ],
  "orphanCIS": [87654321, 98765432]
}
```

**Field mappings**:
- `group` → `groupID` (renamed)
- `cis` → removed (now in medicaments array)
- `type` → removed (now in each medicament in array)
- NEW: `medicaments` array with full composition data
- NEW: `orphanCIS` array for data quality tracking

**Impact**: Clients expecting old structure will break. Must migrate to new structure.

**2. Health endpoint simplified**

System metrics moved from `/health` to `/v1/diagnostics`
- `/health`: Returns basic status only (fast endpoint)
- `/v1/diagnostics`: Detailed system metrics and data quality reports

**3. Go version requirement**

Minimum Go version: 1.21 → 1.24+ (latest stable)

### Migration Guide

**Complete migration guide available in README.md** under "Guide de Migration vers v1"

**Quick reference**:
```javascript
// Legacy
fetch('https://medicaments-api.giygas.dev/medicament/paracetamol')

// V1
fetch('https://medicaments-api.giygas.dev/v1/medicaments?search=paracetamol')
```

### Testing & Quality

- **Overall coverage**: 78.5%
- **Handlers**: 85.6%
- **Medicaments Parser**: 84.2%
- **New test files**: Smoke tests, ETag validation, v1 endpoints, cross-file consistency
- **CI benchmarks**: Non-blocking with 25% variance tolerance

[Unreleased]: https://github.com/giygas/medicaments-api/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/giygas/medicaments-api/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/giygas/medicaments-api/releases/tag/v1.0.0
