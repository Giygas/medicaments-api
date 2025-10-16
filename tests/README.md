# Tests Directory

This directory contains specialized tests for the medicaments-api, organized by purpose for better maintainability.

## 📁 Test Organization

### **Performance & Benchmark Tests**

| File | Purpose | Commands |
|------|---------|----------|
| `benchmark_test.go` | Core performance benchmarks with clean output | `go test ./tests/ -bench=. -benchmem`<br>`go test ./tests/ -bench=BenchmarkSummary -run=^$ -v` |
| `algorithmic_performance_test.go` | Deep algorithmic performance analysis | `go test ./tests/ -run TestAlgorithmicPerformance -v` |
| `realworld_performance_test.go` | Real-world concurrent load testing | `go test ./tests/ -run TestRealWorld -v` |
| `memory_analysis_test.go` | Memory usage and efficiency analysis | `go test ./tests/ -run TestMemoryAnalysis -v` |

### **Verification & Validation Tests**

| File | Purpose | Commands |
|------|---------|----------|
| `documentation_claims_verification_test.go` | Validates all documentation claims against real data | `go test ./tests/ -run TestDocumentationClaimsVerification -v` |
| `parsing_time_test.go` | Quick parsing performance validation | `go test ./tests/ -run TestParsingTime -v` |

### **Integration Tests**

| File | Purpose | Commands |
|------|---------|----------|
| `integration_test.go` | Full pipeline integration testing | `go test ./tests/ -run TestIntegration -v` |
| `endpoints_test.go` | API endpoint behavior validation | `go test ./tests/ -run TestEndpoints -v` |
| `etag_test.go` | HTTP caching mechanism testing | `go test ./tests/ -run TestETag -v` |

## 🚀 Quick Start Commands

### **Run All Tests**
```bash
# All tests in this directory
go test ./tests/ -v

# All tests in entire project
go test -v ./...
```

### **Performance Testing**
```bash
# Clean benchmark output
go test ./tests/ -bench=. -benchmem

# Beautiful performance summary
go test ./tests/ -bench=BenchmarkSummary -run=^$ -v

# Individual benchmark categories
go test ./tests/ -bench=BenchmarkDatabase -run=^$
go test ./tests/ -bench=BenchmarkMedicament -run=^$
```

### **Documentation Verification**
```bash
# Verify all documentation claims
go test ./tests/ -run TestDocumentationClaimsVerification -v

# Quick parsing time check
go test ./tests/ -run TestParsingTime -v
```

### **Integration Testing**
```bash
# Full pipeline tests
go test ./tests/ -run TestIntegrationFullDataParsingPipeline -v

# Concurrent updates test
go test ./tests/ -run TestIntegrationConcurrentUpdates -v

# Memory usage test
go test ./tests/ -run TestIntegrationMemoryUsage -v
```

## 📊 Performance Summary

The `BenchmarkSummary` function provides a comprehensive, well-formatted performance report including:

- **System Information**: Platform, memory, goroutines
- **Algorithmic Performance**: HTTP handler-level benchmarks
- **Parsing Performance**: Full dataset parsing time
- **Documentation Verification**: Status of all accuracy claims

Example output:
```
============================================================
📊 MEDICAMENTS API PERFORMANCE SUMMARY
============================================================
🖥️  System: darwin arm64
🧵 Goroutines: 7
💾 Memory: 39.9 MB alloc, 63.0 MB sys
📦 Data: 15811 medicaments, 1628 generiques

⚡ ALGORITHMIC PERFORMANCE (HTTP Handler Level)
--------------------------------------------------
BenchmarkSummary/MedicamentByID-8         	  438453	      2779 ns/op
BenchmarkSummary/GeneriquesByID-8         	  476863	      2305 ns/op
BenchmarkSummary/DatabasePage-8           	   53756	     21079 ns/op
BenchmarkSummary/Health-8                 	   40160	     31170 ns/op

🔄 PARSING PERFORMANCE
------------------------------
⏱️  Full parsing: 434.77175ms (15811 medicaments)

📋 DOCUMENTATION VERIFICATION
-----------------------------------
✅ Parsing time: ~0.5s (verified)
✅ Memory usage: 30-50MB stable (verified)
✅ Algorithmic performance: 350K-400K req/sec (verified)
✅ Test coverage: 75.5% (exceeds claim)
============================================================
```

## 🎯 Test Coverage

- **Unit Tests**: In individual package directories (`*_test.go`)
- **Integration Tests**: `integration_test.go`
- **Performance Tests**: `benchmark_test.go`, `algorithmic_performance_test.go`
- **Documentation Verification**: `documentation_claims_verification_test.go`

## 📝 Notes

- All tests use `package main` to access the main application code
- Tests are organized by purpose, not by file size
- Performance tests cache data to avoid repeated parsing
- Integration tests use real BDPM data for authentic testing
- Documentation verification ensures accuracy of public claims

## 🔧 Development

When adding new tests:

1. **Performance tests** → Add to `benchmark_test.go` or create specialized file
2. **Integration tests** → Add to `integration_test.go`
3. **New verification** → Add to `documentation_claims_verification_test.go`
4. **Unit tests** → Keep in respective package directories

This organization keeps the root directory clean while maintaining comprehensive test coverage.