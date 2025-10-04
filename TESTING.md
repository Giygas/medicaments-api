# Testing Strategy

This document outlines the comprehensive testing strategy for the Medicaments API.

## Test Suite Overview

We have a multi-tiered testing approach that runs in GitHub Actions:

### 1. Unit Tests (`unit-tests` job)
- **Purpose**: Fast feedback on code changes
- **What it tests**:
  - Parser tests (`./medicamentsparser`)
  - Main application unit tests (short mode)
  - Race condition detection
  - Code formatting (`gofmt`)
  - Static analysis (`go vet`)
  - Dependency verification
- **Coverage**: Generates coverage report for unit tests
- **Duration**: ~30 seconds

### 2. Integration Tests (`integration-tests` job)
- **Purpose**: Test complete data parsing pipeline with real data
- **What it tests**:
  - Full data parsing from external sources
  - API endpoints with real parsed data
  - Data integrity and cross-references
  - Performance and memory usage
  - Race condition detection on integration tests
- **Data**: Processes ~15,803 medicaments and 1,618 generique groups
- **Duration**: ~3-5 minutes
- **Coverage**: Generates separate integration coverage report

### 3. Full Test Suite (`full-test-suite` job)
- **Purpose**: Complete test coverage with quality gates
- **What it tests**:
  - All tests combined (unit + integration)
  - Coverage threshold enforcement (minimum 70%)
  - Comprehensive race detection
- **Coverage Threshold**: 70% minimum required
- **Duration**: ~5-8 minutes

### 4. Build Test (`build-test` job)
- **Purpose**: Ensure application compiles correctly
- **What it tests**:
  - Cross-platform build (Linux AMD64)
  - Binary generation
  - Artifact upload for deployment

## Running Tests Locally

### Quick Unit Tests
```bash
go test -short -v
```

### Integration Tests
```bash
go test -run TestIntegrationFullDataParsingPipeline -v
```

### Full Test Suite with Coverage
```bash
go test -race -coverprofile=coverage.out -v
go tool cover -html=coverage.out -o coverage.html
```

### Parser Tests Only
```bash
go test ./medicamentsparser -v
```

### Race Detection
```bash
go test -race -v
```

## Test Categories

### Unit Tests
- **Endpoint Tests**: All HTTP endpoints with mock data
- **Middleware Tests**: Rate limiting, IP validation, request size limits
- **Compression Tests**: JSON compression optimization
- **Parser Tests**: Individual parser components

### Integration Tests
- **Data Pipeline**: Complete download → parse → in-memory flow
- **API Integration**: Real data serving through all endpoints
- **Performance**: Memory usage and timing benchmarks
- **Data Integrity**: Cross-reference validation between entities

## Coverage Reports

Coverage reports are generated and uploaded as artifacts:
- `coverage-reports-unit`: Unit test coverage
- `coverage-reports-integration`: Integration test coverage  
- `coverage-reports-full`: Combined coverage with threshold validation

## CI/CD Pipeline

1. **Pull Request**: Runs unit tests only (fast feedback)
2. **Push to main**: Runs full comprehensive test suite
3. **Deployment**: Only proceeds if all tests pass

## Performance Benchmarks

Integration tests track:
- **Parsing Time**: 3-5 seconds for full pipeline
- **Memory Usage**: Monitored during parsing
- **API Response Times**: Validated for all endpoints
- **Data Processing**: 15,803 medicaments + 1,618 generiques

## Quality Gates

- **Code Formatting**: Must pass `gofmt` check
- **Static Analysis**: Must pass `go vet`
- **Coverage**: Minimum 70% required for deployment
- **Race Conditions**: Must pass race detection
- **Build**: Must compile successfully

## Test Data

Integration tests use real data from external sources:
- **Conditions**: 27,602 records
- **Generiques**: 10,549 records  
- **Specialites**: 15,803 records
- **Compositions**: 32,558 records
- **Presentations**: 20,905 records

## Troubleshooting

### Integration Test Timeouts
- Default timeout: 10 minutes
- Race detection timeout: 15 minutes
- Can be adjusted with `-timeout` flag

### Memory Issues
- Integration tests monitor memory usage
- Expected usage: ~50MB during parsing
- Failures indicate memory leaks or inefficient parsing

### Network Issues
- Integration tests require internet access
- Tests will fail if external sources are unavailable
- Can be skipped with `-short` flag during development