# Tests Directory

This directory contains specialized tests for the medicaments-api, organized by purpose for better maintainability.

## 📁 Test Organization

### **Performance & Benchmark Tests**

| File | Purpose | Commands |
|------|---------|----------|
| `performance_benchmarks_test.go` | Core performance benchmarks with production logging | `go test ./tests -bench=. -benchmem`<br>`go test ./tests -bench=BenchmarkAlgorithmicPerformance -v`<br>`go test ./tests -bench=BenchmarkHTTPPerformance -v`<br>`go test ./tests -bench=BenchmarkRealWorldSearch -v`<br>`go test ./tests -bench=BenchmarkSustainedPerformance -v` |
| `documentation_claims_verification_test.go` | Validates all documentation claims against real data | `go test ./tests -run TestDocumentationClaimsVerification -v` |

### **Verification & Validation Tests**

| File | Purpose | Commands |
|------|---------|----------|
| `documentation_claims_verification_test.go` | Validates all documentation claims against real data | `go test ./tests -run TestDocumentationClaimsVerification -v` |

### **Integration Tests**

| File | Purpose | Commands |
|------|---------|----------|
| `integration_test.go` | Full pipeline integration testing | `go test ./tests -run TestIntegration -v` |
| `cross_file_consistency_integration_test.go` | Cross-file data consistency validation | `go test ./tests -run TestIntegrationCrossFileConsistency -v` |

### **API Endpoint Tests**

| File | Purpose | Commands |
|------|---------|----------|
| `endpoints_test.go` | API endpoint behavior validation | `go test ./tests -run TestEndpoints -v` |
| `etag_test.go` | HTTP caching mechanism testing | `go test ./tests -run TestETagFunctionality -v` |

### **Smoke Tests**

| File | Purpose | Commands |
|------|---------|----------|
| `smoke_test.go` | Quick validation and smoke tests | `go test ./tests -run TestApplicationStartupSmoke -v` |

## 🚀 Quick Start Commands

### **Run All Tests**
```bash
# All tests in tests directory
go test ./tests -v

# All tests in entire project
go test -v ./...
```

### **Performance Testing**
```bash
# All benchmarks with production environment (optimal performance)
go test ./tests -bench=. -benchmem

# Algorithmic benchmarks (handler-level)
go test ./tests -bench=BenchmarkAlgorithmicPerformance -benchmem -v

# HTTP benchmarks (network-level)
go test ./tests -bench=BenchmarkHTTPPerformance -benchmem -v

# Real-world search benchmarks
go test ./tests -bench=BenchmarkRealWorldSearch -benchmem -v

# Sustained performance benchmarks
go test ./tests -bench=BenchmarkSustainedPerformance -benchmem -v
```

### **Documentation Verification**
```bash
# Verify all documentation claims
go test ./tests -run TestDocumentationClaimsVerification -v
```

### **Integration Testing**
```bash
# Full pipeline tests
go test ./tests -run TestIntegrationFullDataParsingPipeline -v
go test ./tests -run TestIntegrationConcurrentUpdates -v
go test ./tests -run TestIntegrationErrorHandling -v

# Cross-file consistency
go test ./tests -run TestIntegrationCrossFileConsistency -v
```

### **Endpoint & Middleware Tests**
```bash
# All endpoint tests
go test ./tests -run TestEndpoints -v

# Middleware tests
go test ./tests -run TestBlockDirectAccessMiddleware -v
go test ./tests -run TestRateLimiter -v
go test ./tests -run TestRealIPMiddleware -v
go test ./tests -run TestRequestSizeMiddleware -v
go test ./tests -run TestCompressionOptimization -v

# ETag tests
go test ./tests -run TestETagFunctionality -v
```

### **Smoke Tests**
```bash
# Quick validation
go test ./tests -run TestApplicationStartupSmoke -v
```

## 📊 Performance Summary

Performance benchmarks use production environment (`config.EnvProduction`) for optimal performance. This means:

- **Console logging**: WARN level and above only (no INFO/DEBUG output to console)
- **File logging**: Full JSON output (all levels written to rotating log files)
- **Result**: Eliminates console I/O overhead during benchmarks for accurate measurements

### Example Verification Report Output

When running documentation claims verification:

```bash
go test ./tests -run TestDocumentationClaimsVerification -v
```

You'll see output like:

```
=== COMPREHENSIVE DOCUMENTATION CLAIMS VERIFICATION ===

--- ALGORITHMIC PERFORMANCE VERIFICATION ---
  /v1/medicaments/{cis}: 441695.7 req/sec (claimed: 400000.0 req/sec, diff: 10.4%)
  /v1/medicaments/{cis}: 2.0 µs (claimed: 3.0 µs, diff: -33.3%)
  /v1/generiques/{groupID}: 244390.5 req/sec (claimed: 200000.0 req/sec, diff: 22.2%)
  /v1/generiques/{groupID}: 4.0 µs (claimed: 5.0 µs, diff: -20.0%)
  /v1/medicaments?page={n}: 40152.5 req/sec (claimed: 40000.0 req/sec, diff: 0.4%)
  /v1/medicaments?page={n}: 24.0 µs (claimed: 30.0 µs, diff: -20.0%)
  /v1/medicaments?search={query}: 1634.3 req/sec (claimed: 1600.0 req/sec, diff: 2.1%)
  /v1/medicaments?search={query}: 611.0 µs (claimed: 750.0 µs, diff: -18.5%)
  /v1/generiques?libelle={nom}: 16742.9 req/sec (claimed: 18000.0 req/sec, diff: -7.0%)
  /v1/generiques?libelle={nom}: 59.0 µs (claimed: 60.0 µs, diff: -1.7%)
  /v1/presentations?cip={code}: 438566.6 req/sec (claimed: 430000.0 req/sec, diff: 2.0%)
  /v1/presentations?cip={code}: 2.0 µs (claimed: 2.0 µs, diff: 0.0%)
  /v1/medicaments?cip={code}: 394485.4 req/sec (claimed: 375000.0 req/sec, diff: 5.2%)
  /v1/medicaments?cip={code}: 2.0 µs (claimed: 5.0 µs, diff: -60.0%)
  /health: 416206.4 req/sec (claimed: 400000.0 req/sec, diff: 4.1%)
  /health: 2.0 µs (claimed: 3.0 µs, diff: -33.3%)

--- HTTP PERFORMANCE VERIFICATION ---
  /v1/medicaments/{cis}: 90015.7 req/sec (claimed: 78000.0 req/sec, diff: 15.4%)
  /v1/medicaments?page={n}: 49463.0 req/sec (claimed: 41000.0 req/sec, diff: 20.6%)
  /v1/medicaments?search={query}: 7412.0 req/sec (claimed: 6100.0 req/sec, diff: 21.5%)
  /v1/generiques?libelle={nom}: 46865.7 req/sec (claimed: 36000.0 req/sec, diff: 30.2%)
  /v1/presentations?cip={code}: 91614.3 req/sec (claimed: 77000.0 req/sec, diff: 19.0%)
  /v1/medicaments?cip={code}: 92352.7 req/sec (claimed: 75000.0 req/sec, diff: 23.1%)
  /health: 114412.3 req/sec (claimed: 92000.0 req/sec, diff: 24.4%)

--- MEMORY USAGE VERIFICATION ---
  Application memory: 75.3 MB alloc, 158.1 MB sys (claimed: 70.0-90.0 MB)

--- PARSING PERFORMANCE VERIFICATION ---
  Parsing time: 0.5 seconds (claimed: 0.7 seconds)

=== VERIFICATION REPORT ===
  ✅ PASS /v1/medicaments/{cis} algorithmic throughput: 441695.7 req/sec (claimed: 400000.0 req/sec, diff: 10.4%)
  ✅ PASS /v1/medicaments/{cis} algorithmic latency: 2.0 µs (claimed: 3.0 µs, diff: -33.3%)
  ✅ PASS /v1/generiques/{groupID} algorithmic throughput: 244390.5 req/sec (claimed: 200000.0 req/sec, diff: 22.2%)
  ✅ PASS /v1/generiques/{groupID} algorithmic latency: 4.0 µs (claimed: 5.0 µs, diff: -20.0%)
✅ PASS /v1/medicaments?page={n} algorithmic throughput: 40152.5 req/sec (claimed: 40000.0 req/sec, diff: 0.4%)
✅ PASS /v1/medicaments?page={n} algorithmic latency: 24.0 µs (claimed: 30.0 µs, diff: -20.0%)
✅ PASS /v1/medicaments?search={query} algorithmic throughput: 1634.3 req/sec (claimed: 1600.0 req/sec, diff: 2.1%)
✅ PASS /v1/medicaments?search={query} algorithmic latency: 611.0 µs (claimed: 750.0 µs, diff: -18.5%)
✅ PASS /v1/generiques?libelle={nom} algorithmic throughput: 16742.9 req/sec (claimed: 18000.0 req/sec, diff: -7.0%)
✅ PASS /v1/generiques?libelle={nom} algorithmic latency: 59.0 µs (claimed: 60.0 µs, diff: -1.7%)
✅ PASS /v1/presentations?cip={code} algorithmic throughput: 438566.6 req/sec (claimed: 430000.0 req/sec, diff: 2.0%)
✅ PASS /v1/presentations?cip={code} algorithmic latency: 2.0 µs (claimed: 2.0 µs, diff: 0.0%)
✅ PASS /v1/medicaments?cip={code} algorithmic throughput: 394485.4 req/sec (claimed: 375000.0 req/sec, diff: 5.2%)
✅ PASS /v1/medicaments?cip={code} algorithmic latency: 2.0 µs (claimed: 5.0 µs, diff: -60.0%)
✅ PASS /health algorithmic throughput: 416206.4 req/sec (claimed: 400000.0 req/sec, diff: 4.1%)
✅ PASS /health algorithmic latency: 2.0 µs (claimed: 3.0 µs, diff: -33.3%)
✅ PASS /v1/medicaments/{cis} HTTP throughput: 90015.7 req/sec (claimed: 78000.0 req/sec, diff: 15.4%)
✅ PASS /v1/medicaments?page={n} HTTP throughput: 49463.0 req/sec (claimed: 41000.0 req/sec, diff: 20.6%)
✅ PASS /v1/medicaments?search={query} HTTP throughput: 7412.0 req/sec (claimed: 6100.0 req/sec, diff: 21.5%)
✅ PASS /v1/generiques?libelle={nom} HTTP throughput: 46865.7 req/sec (claimed: 36000.0 req/sec, diff: 30.2%)
✅ PASS /v1/presentations?cip={code} HTTP throughput: 91614.3 req/sec (claimed: 77000.0 req/sec, diff: 19.0%)
✅ PASS /v1/medicaments?cip={code} HTTP throughput: 92352.7 req/sec (claimed: 75000.0 req/sec, diff: 23.1%)
✅ PASS /health HTTP throughput: 114412.3 req/sec (claimed: 92000.0 req/sec, diff: 24.4%)
✅ PASS Application memory usage: 75.3 MB (claimed: 80.0 MB, diff: -5.9%)
✅ PASS Concurrent TSV parsing: 0.5 seconds (claimed: 0.7 seconds, diff: -30.9%)

SUMMARY: 25/25 claims verified (100.0%)
```

### Interpreting the Report

**Status Indicators:**
- ✅ PASS = Meets or exceeds claim (within tolerance)
- ❌ FAIL = Below minimum threshold (more than tolerance below claim)

**Performance Sections:**
- **Algorithmic**: Handler-level benchmarks with subset dataset (~500 items)
- **HTTP**: Network-level benchmarks with full dataset (~15K+ items)
- **Memory**: Application memory usage under load
- **Parsing**: Concurrent TSV file processing time

**Metrics:**
- **Throughput**: Requests per second (higher is better)
- **Latency**: Microseconds per operation (lower is better)
- **Diff**: Percentage difference from claimed value
  - Positive = Measured higher than claimed (good!)
  - Negative = Measured lower than claimed (within tolerance is OK)

**Tolerance Settings:**
- Algorithmic claims: 20% tolerance (30% for search endpoints)
- HTTP throughput claims: 25% tolerance (for network variance)
- Memory claim: Range of 70-90 MB (80 MB average)
- Parsing time: 100% tolerance (for CI variability)

**Environment Impact:**
Using `config.EnvProduction` ensures benchmarks run with production-like logging:
- Console: WARN and above only (minimal I/O overhead)
- File: Full JSON output (all levels captured)
- Result: More accurate performance measurements

### Current Performance Claims

Recent optimizations have significantly improved performance:

**1. Pre-computed Normalized Names** (previous commit):
- Added `DenominationNormalized` field to Medicament entity
- Added `LibelleNormalized` field to GeneriqueList entity
- Normalization happens once during parsing instead of on every request
- **Result**: 10x search performance improvement

**2. Environment-aware Logging** (current commit):
- Production/test environments use reduced console logging
- Console: WARN/ERROR only (vs INFO in dev)
- File: Full JSON output always
- **Result**: Eliminates console I/O overhead during benchmarks

**Combined Effect**: 2-3x HTTP throughput improvement on most endpoints

### Current Performance Claims (Documentation)

**Algorithmic Benchmarks** (handler-level with subset dataset ~500 items):
- `/v1/medicaments/{cis}`: 400,000 req/sec, 3.0µs latency
- `/v1/generiques/{groupID}`: 200,000 req/sec, 5.0µs latency
- `/v1/medicaments?page={n}`: 40,000 req/sec, 30.0µs latency
- `/v1/medicaments?search={query}`: 1,600 req/sec, 750.0µs latency
- `/v1/generiques?libelle={nom}`: 18,000 req/sec, 60.0µs latency
- `/v1/presentations?cip={code}`: 430,000 req/sec, 2.0µs latency
- `/v1/medicaments?cip={code}`: 375,000 req/sec, 5.0µs latency
- `/health`: 400,000 req/sec, 3.0µs latency

**HTTP Benchmarks** (network-level with full dataset ~15K+ items):
- `/v1/medicaments/{cis}`: 78,000 req/sec
- `/v1/medicaments?page={n}`: 41,000 req/sec
- `/v1/medicaments?search={query}`: 6,100 req/sec
- `/v1/generiques?libelle={nom}`: 36,000 req/sec
- `/v1/presentations?cip={code}`: 77,000 req/sec
- `/v1/medicaments?cip={code}`: 75,000 req/sec
- `/health`: 92,000 req/sec

**Memory Usage**: 70-90 MB (80 MB midpoint)
**Concurrent Parsing**: ~0.5-0.7 seconds for full dataset

## 📋 Test Coverage

### Test Types

- **Unit Tests**: In individual package directories (`*_test.go`)
- **Integration Tests**: `integration_test.go`, `cross_file_consistency_integration_test.go`
- **Performance Tests**: `performance_benchmarks_test.go`
- **Documentation Verification**: `documentation_claims_verification_test.go`
- **Endpoint Tests**: `endpoints_test.go`
- **Middleware Tests**: In `server/middleware_test.go`
- **ETag Tests**: `etag_test.go`
- **Smoke Tests**: `smoke_test.go`

## 📝 Notes

- All tests use `package main` to access main application code
- Tests are organized by purpose, not by file size
- Performance benchmarks use production environment for optimal measurements
- Integration tests use real BDPM data for authentic testing
- Documentation verification ensures accuracy of public claims

## 🔧 Development

### Running Benchmarks

When running performance benchmarks, they automatically use production environment:

```go
// All benchmark functions initialize with production logging
logging.InitLoggerWithEnvironment("", config.EnvProduction, 4, 100*1024*1024)
```

This ensures:
- No console I/O overhead (WARN/ERROR to console only)
- File logging captures all output for analysis
- Accurate performance measurements (production-like environment)

### Adding New Tests

1. **Performance benchmarks** → Add to `performance_benchmarks_test.go`
2. **Integration tests** → Add to `integration_test.go` or create new file
3. **New verification** → Add to `documentation_claims_verification_test.go`
4. **Unit tests** → Keep in respective package directories

### Test Organization Guidelines

- **Keep tests organized by purpose**: Performance, Integration, Verification, Endpoint, Smoke
- **Use descriptive test names**: Make it clear what each test validates
- **Test edge cases**: Include both happy path and error scenarios
- **Use test helpers**: Common setup/teardown in helper functions
- **Avoid test pollution**: Clean up resources in test cleanup (defer, t.Cleanup())
- **Use production environment for benchmarks**: Ensures realistic performance measurements

This organization keeps the root directory clean while maintaining comprehensive test coverage.
