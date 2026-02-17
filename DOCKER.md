# Docker Staging Setup Guide

**Complete guide for running medicaments-api in a Docker staging environment**

---

## Table of Contents

- [Quick Start](#quick-start)
- [Essential Commands](#essential-commands)
- [Project Overview](#project-overview)
  - [What Was Created](#what-was-created)
  - [Project Structure](#project-structure)
  - [Configuration](#configuration)
- [Observability Stack](#observability-stack)
  - [Architecture](#observability-architecture)
  - [Services](#observability-services)
  - [Access Points](#observability-access-points)
  - [Log Format](#observability-log-format)
  - [Metrics Collected](#observability-metrics-collected)
  - [Default Credentials](#observability-default-credentials)
  - [Resource Usage](#observability-resource-usage)
  - [Troubleshooting](#observability-troubleshooting)
  - [Configuration Files](#observability-configuration-files)
- [Docker Compose Commands](#docker-compose-commands)
  - [Build and Run](#build-and-run)
  - [View Logs](#view-logs)
  - [Container Management](#container-management)
- [API Endpoints](#api-endpoints)
- [Data Management](#data-management)
- [Troubleshooting](#troubleshooting)
  - [Container Won't Start](#container-wont-start)
  - [Health Check Failing](#health-check-failing)
  - [Data Download Issues](#data-download-issues)
  - [Logs Not Persisting](#logs-not-persisting)
  - [High Memory Usage](#high-memory-usage)
  - [Port Conflicts](#port-conflicts)
- [Advanced Usage](#advanced-usage)
  - [Custom Environment Variables](#custom-environment-variables)
  - [Running Multiple Instances](#running-multiple-instances)
  - [Debugging](#debugging)
  - [Performance Testing](#performance-testing)
- [Security Considerations](#security-considerations)
  - [Non-Root User](#non-root-user)
  - [Network Isolation](#network-isolation)
  - [Volume Permissions](#volume-permissions)
- [Monitoring](#monitoring)
  - [Container Metrics](#container-metrics)
  - [Application Metrics](#application-metrics)
  - [Log Monitoring](#log-monitoring)
- [Cleanup](#cleanup)
- [Production Differences](#production-differences)
- [CI/CD Integration](#cicd-integration)
- [Support](#support)
- [Appendix](#appendix)

---

## Quick Start

### Prerequisites

- Docker Engine 20.10+ or Docker Desktop 4.0+
- At least 1GB available disk space
- Network connection for BDPM data download

### Get Started Immediately

```bash
# Docker Compose (recommended)
docker-compose up -d

# View logs
docker-compose logs -f

# Check health
curl http://localhost:8030/health

# Stop
docker-compose down
```

### What Happens on First Run

1. **Docker builds image** (~1-2 minutes)
2. **Container starts** as non-root user (UID 65534/nobody)
3. **BDPM data downloads** from external sources (~10-30 seconds)
4. **HTTP server starts** on port 8000
5. **Health check begins** after 10-second start period
6. **API is ready** at http://localhost:8030

---

## Essential Commands

### 🚀 Start & Stop

```bash
docker-compose up -d             # Start detached
docker-compose down               # Stop & remove containers
docker-compose restart            # Restart container
```

### 📋 Logs

```bash
docker-compose logs -f           # Follow logs in real-time
docker-compose logs --tail=100   # Last 100 lines
docker-compose logs | grep error   # Search for errors
```

### 🔍 Status & Health

```bash
docker-compose ps                 # Container status
curl http://localhost:8030/health # Health check
docker stats medicaments-api # Resource usage
```

### 🛠️ Build & Rebuild

```bash
docker-compose build             # Build image
docker-compose up -d --build     # Rebuild & start
docker-compose build --no-cache  # Clean build (no cache)
```

### 🏗️ Multi-Architecture Builds

```bash
# Build for host architecture (auto-detected)
make build

# Force specific architecture
make build-amd64
make build-arm64

# Start services
make up

# View all available commands
make help
```

**Docker Compose (auto-detects platform):**

```bash
docker-compose up -d    # Builds for your native platform
docker-compose build      # Builds for your native platform
```

**Supported Platforms:**

| Architecture | Description | Target Platforms |
|--------------|-------------|------------------|
| **amd64** | Intel/AMD x86_64 | Intel/AMD servers, cloud instances, Intel Macs |
| **arm64** | ARM 64-bit | Apple Silicon (M1/M2/M3), Raspberry Pi 4, AWS Graviton |

**Note:** Use `--load` flag to make image available locally. Without it, image only exists in BuildKit cache.

---

## Project Overview

### What Was Created

The following files were added to set up your Docker staging environment:

#### 1. **Dockerfile**

Multi-stage Docker build optimized for production:

- **Stage 1 - Builder**: `golang:1.26-alpine`
  - Uses `syntax=docker/dockerfile:1` for buildkit support
  - Cache mounts for Go packages and build cache (faster rebuilds)
  - Multi-architecture support via `$TARGETARCH` BuildKit variable
  - Copies HTML documentation from `/build/html`
- **Stage 2 - Runtime**: `scratch` (~8-10MB final image, minimal attack surface)
- **Security**: Non-root user (UID 65534/nobody)
- **Health Check**: Built-in using HEALTHCHECK instruction with healthcheck subcommand
- **Files**: Copies binary, CA certificates, and HTML documentation
- **Build Dependencies**: Only `ca-certificates`

#### 2. **docker-compose.yml**

Docker Compose orchestration:

- **Port Mapping**: 8030 (host) → 8000 (container)
- **Environment**: Variables from `.env.docker`
- **Logs**: Persistent via named volume (`logs_data:/app/logs`)
- **Security**: Read-only filesystem, no-new-privileges, tmpfs for /app/files
- **Resources**:
  - medicaments-api: 512MB/0.5CPU limits, 256MB/0.25CPU reservations
  - grafana-alloy: 256MB/0.5CPU limits, 128MB/0.1CPU reservations
  - loki: 512MB/1.0CPU limits, 256MB/0.2CPU reservations
  - prometheus: 1G/1.0CPU limits, 512MB/0.3CPU reservations
  - grafana: 512MB/0.5CPU limits, 256MB/0.1CPU reservations
- **Health Check**: Delegated to Dockerfile (30s interval, 5s timeout, 10s start period, 3 retries)
- **Restart**: Policy `unless-stopped`
- **Network**: Custom bridge network for isolation
- **Container Labels**: Metadata for identification and management

#### 3. **.dockerignore**

Optimizes Docker build context:

- Excludes: logs, git, vendor, test files, \*.md (except README.md), observability/ (except config files)
- Keeps: source code and HTML docs
- Reduces: build time and image size
- Observability configs: Explicitly includes necessary config files from observability/

#### 4. **.env.docker**

Docker environment configuration:
| Variable | Value | Description |
|----------|-------|-------------|
| `ADDRESS` | `0.0.0.0` | Listen on all interfaces in container |
| `PORT` | `8000` | Port inside container |
| `ENV` | `production` | Environment mode |
| `ALLOW_DIRECT_ACCESS` | `true` | Allow binding to all interfaces (Docker only) |
| `LOG_LEVEL` | `info` | Logging level (debug/info/warn/error) |
| `LOG_RETENTION_WEEKS` | `2` | Keep logs for 2 weeks |
| `MAX_LOG_FILE_SIZE` | `52428800` | Rotate at 50MB |
| `MAX_REQUEST_BODY` | `2097152` | 2MB max request body |
| `MAX_HEADER_SIZE` | `2097152` | 2MB max header size |
| `GRAFANA_ADMIN_USER` | `giygas` | Grafana admin username |
| `GRAFANA_ADMIN_PASSWORD` | `paquito` | Grafana admin password |

#### 5. **Makefile**

Unified build and development commands:

- Auto-detects host architecture (amd64 or arm64)
- Provides unified interface for Docker, testing, and benchmarking
- Supports explicit architecture targeting: `make build-amd64` or `make build-arm64`
- Common operations: `make build`, `make up`, `make down`, `make logs`, `make test`, `make bench`
- View all commands: `make help`

#### 6. **.gitignore** (updated)

Added comprehensive exclusions including:

- `.env.docker` and other environment files
- `observability/` directory (with exceptions for config files)
- Standard Git, CI/CD, IDE, and OS files
- Test artifacts and build files

### Project Structure

```
medicaments-api/
├── Dockerfile              # Multi-stage Docker build
├── docker-compose.yml      # Docker Compose orchestration (includes observability stack)
├── .dockerignore          # Files excluded from build context
├── .env.docker             # Docker environment variables
├── Makefile               # Unified build and development commands
├── logs/                  # Persistent logs directory
├── html/                  # Documentation files (served by API)
├── observability/         # Grafana stack configuration
│   ├── alloy/              # Alloy config
│   ├── loki/               # Loki config
│   ├── prometheus/          # Prometheus config
│   │   └── alerts/          # Prometheus alert rules
│   └── grafana/             # Grafana config
│       ├── provisioning/      # Auto-provisioning
│       │   ├── datasources/   # Loki & Prometheus datasources
│       │   └── dashboards/     # Dashboard provisioning
│       └── dashboards/        # Dashboard JSON files
```

### Configuration

#### Port Mapping

- **Host Port**: 8030
- **Container Port**: 8000

Access the API at: `http://localhost:8030`

#### Resource Limits

Staging container has the following limits:

- **CPU**: 0.5 cores max, 0.25 cores reserved
- **Memory**: 512MB max, 256MB reserved

---

## Observability Stack

The staging setup includes a complete observability stack with Grafana, Loki, Prometheus, and Alloy for logs and metrics monitoring.

### Observability Architecture

```
medicaments-api (logs + metrics)
          ↓
grafana-alloy (collector)
          ↓         ↓
       loki    prometheus
          ↓         ↓
          grafana (visualization)
```

### Port Architecture

| Service         | Container Port | Host Port | External Access                | Internal Communication |
| --------------- | -------------- | --------- | ------------------------------ | ---------------------- |
| medicaments-api | 8000 (API)     | 8030      | http://localhost:8030          | medicaments-api:8000   |
| medicaments-api | 9090 (metrics) | internal  | N/A                            | medicaments-api:9090   |
| grafana-alloy   | 12345          | 12345     | http://localhost:12345/metrics | grafana-alloy:12345    |
| loki            | 3100           | internal  | N/A                            | loki:3100              |
| prometheus      | 9090           | 9090      | http://localhost:9090          | prometheus:9090        |
| grafana         | 3000           | 3000      | http://localhost:3000          | grafana:3000           |

**Key Points:**

- Grafana connects to Prometheus at `prometheus:9090` (container port)
- External access to Prometheus is via `localhost:9090` (host port mapping)
- All service-to-service communication uses container ports within Docker network
- Host ports are only for accessing services from the host machine
- Some services (medicaments-api metrics, Loki) are only exposed internally to the Docker network for security

### Observability Services

#### grafana-alloy

Collects logs and metrics from medicaments-api and system metrics.

- **Image**: `grafana/alloy:v1.4.0`
- **Configuration**: `alloy/config.alloy`
- **Port**: 12345 (Alloy's own metrics)
- **Functions**:
  - Read logs from `./logs/` directory
  - Scrape application metrics from `medicaments-api:9090/metrics` (every 30s)
  - Collect system metrics via Unix exporter (every 60s)
  - Forward to local Loki and Prometheus
  - Filter out Go runtime metrics (keep only HTTP and system metrics)
- **Resource Usage**: ~150MB RAM

#### loki

Log aggregation and storage.

- **Image**: `grafana/loki:2.9.10`
- **Configuration**: `loki/config.yaml`
- **Port**: 3100 (internal only - exposed to Docker network)
- **Storage**: Filesystem (chunks in `/loki/chunks`, rules in `/loki/rules`)
- **Retention**: 30 days (720 hours)
- **Data Volume**: `loki-data`
- **Resource Usage**: ~100MB RAM + ~100MB disk
- **Health Check**: Available via `/ready` endpoint

#### prometheus

Metric storage and querying.

- **Image**: `prom/prometheus:v2.48.0`
- **Configuration**: `prometheus/prometheus.yml`
- **Port**: 9090 (host and container)
  - Host port 9090 provides external access to Prometheus UI
  - Container port 9090 is used for service-to-service communication
- **Retention**: 30 days (720 hours)
- **Data Volume**: `prometheus-data`
- **Resource Usage**: ~150MB RAM + ~200MB disk
- **Scraping**: Receives metrics via `remote_write` from Grafana Alloy (no `scrape_configs` needed)

#### grafana

Visualization for logs and metrics.

- **Image**: `grafana/grafana:10.2.4`
- **Port**: 3000
- **Default Credentials**: giygas/paquito (change after first login)
- **Data Volume**: `grafana-data`
- **Resource Usage**: ~200MB RAM + ~50MB disk
- **Auto-Provisioning**: Datasources configured automatically

### Observability Access Points

```bash
# Grafana UI (visualization)
open http://localhost:3000

# Prometheus UI (metrics browsing)
open http://localhost:9090

# medicaments-api metrics (application metrics)
# Available only via Docker network for internal scraping
curl http://localhost:9090/metrics

# Alloy metrics (collector status)
curl http://localhost:12345/metrics
```

**Note**: Loki and medicaments-api metrics are only exposed internally to the Docker network.
They are scraped by Alloy and not directly accessible from the host machine for security.

### Observability Log Format

Your application should output JSON logs with `level` and `path` fields for proper parsing:

```json
{
  "time": "2025-02-08T12:00:00Z",
  "level": "info",
  "path": "/v1/medicaments?search=paracetamol",
  "msg": "Request completed",
  "status": 200,
  "duration_ms": 15
}
```

If logs are plain text, update `alloy/config.alloy` to remove `stage.json` block and use regex parsing.

### Observability Metrics Collected

From your `/metrics` endpoint (via `metrics/metrics.go`):

- **`http_request_total`** - Total HTTP requests
  - Labels: `method`, `path`, `status`
  - Type: Counter
  - Example: `http_request_total{method="GET",path="/v1/medicaments",status="200"}`

- **`http_request_duration_seconds`** - Request latency histogram
  - Labels: `method`, `path`
  - Type: Histogram
  - Buckets: .001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5
  - Example: `http_request_duration_seconds_sum{method="GET",path="/v1/medicaments"}`

- **`http_request_in_flight`** - Current in-flight requests
  - Type: Gauge
  - Example: `http_request_in_flight`

From system metrics via `prometheus.exporter.unix`:

- **CPU metrics**: Process and system CPU usage
- **Memory metrics**: Process and system memory usage
- **File descriptors**: Open file descriptors
- **Network metrics**: Network I/O statistics
- **Disk metrics**: Disk I/O and filesystem statistics

**Note**: Alloy configuration filters out Go runtime metrics from both application and system scrapers, keeping only HTTP and relevant system metrics.

### Observability Default Credentials

**Grafana**:

- Username: `giygas`
- Password: `paquito`
- **Important**: Change password after first login (Configuration → Users → Change Password)

**Other Services**:

- No authentication required (local network only)

### Observability Resource Usage

| Service         | RAM        | Disk          | Retention      |
| --------------- | ---------- | ------------- | -------------- |
| medicaments-api | ~50MB      | ~20MB         | N/A            |
| grafana-alloy   | ~150MB     | ~10MB         | N/A            |
| loki            | ~100MB     | ~100MB (data) | 30 days        |
| prometheus      | ~150MB     | ~200MB (data) | 30 days (720h) |
| grafana         | ~200MB     | ~50MB         | N/A            |
| **Total**       | **~650MB** | **~380MB**    | 30 days (both) |

### Observability Troubleshooting

#### Grafana can't connect to datasources

```bash
# Check containers are running
docker-compose ps

# Verify network connectivity
# Note: Grafana connects to Prometheus on container port 9090
docker-compose exec grafana wget -O- http://loki:3100/ready
docker-compose exec grafana wget -O- http://prometheus:9090/-/ready

# Check datasource configuration
docker-compose logs grafana | grep -i datasource

# Verify Prometheus datasource configuration
cat observability/grafana/provisioning/datasources/prometheus.yml
# Should show: url: http://prometheus:9090
```

#### Logs not appearing in Grafana

```bash
# Check Alloy is reading logs
docker-compose logs grafana-alloy | grep -i logs

# Verify log files exist
docker-compose exec grafana-alloy ls -la /var/log/app/

# Check Loki is receiving logs
docker-compose logs loki | grep -i received

# Query logs from within the Docker network
docker-compose exec loki wget -O- 'http://localhost:3100/loki/api/v1/labels'
```

#### Metrics not appearing in Grafana

```bash
# Check Alloy is scraping metrics
docker-compose logs grafana-alloy | grep -i scrape

# Verify metrics endpoint is accessible
docker-compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics

# Check Prometheus is receiving metrics
docker-compose logs prometheus | grep -i received

# Test Prometheus query
curl 'http://localhost:9090/api/v1/query?query=http_request_total'
```

#### Loki fails to start with storage error

```bash
# Check Loki logs for storage configuration errors
docker-compose logs loki | grep -i "storage\|ruler"

# Common error: "field filesystem not found in type base.RuleStoreConfig"
# This occurs in Loki 2.9+ when ruler storage type is not explicitly specified

# Fix: Ensure loki/config.yaml ruler section has explicit local storage:
#   ruler:
#     storage:
#       type: local
#       local:
#         directory: /loki/rules

# Restart Loki after fixing config
docker-compose restart loki
```

#### High resource usage

```bash
# Check resource usage for all services
docker stats medicaments-api grafana-alloy loki prometheus grafana

# Check disk usage for volumes
docker system df -v

# Reduce retention if needed (edit loki/config.yaml or prometheus/prometheus.yml)
```

### Observability Cleanup

```bash
# Stop observability services only
docker-compose stop grafana-alloy loki prometheus grafana

# Remove observability services (keeps volumes)
docker-compose rm -f grafana-alloy loki prometheus grafana

# Remove observability services and all data (DELETES EVERYTHING)
docker-compose down -v

# Remove only observability volumes
docker volume rm medicaments-api_loki-data medicaments-api_prometheus-data medicaments-api_grafana-data
```

### Observability Configuration Files

| File                                                            | Purpose                                                                                               |
| --------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `observability/alloy/config.alloy`                              | Alloy configuration (logs + metrics collection, filters Go runtime metrics)                           |
| `observability/loki/config.yaml`                                | Loki configuration (log storage, filesystem ruler storage, 30-day retention, 16MB/sec ingestion rate) |
| `observability/prometheus/prometheus.yml`                       | Prometheus configuration (metric storage, 30-day retention, alert rules)                              |
| `observability/prometheus/alerts/medicaments-api.yml`           | Prometheus alert rules (service down, high error rate, high latency)                                  |
| `observability/grafana/provisioning/datasources/loki.yml`       | Auto-configure Loki datasource                                                                        |
| `observability/grafana/provisioning/datasources/prometheus.yml` | Auto-configure Prometheus datasource                                                                  |
| `observability/grafana/provisioning/dashboards/dashboard.yml`   | Auto-import Grafana dashboards                                                                        |
| `observability/grafana/dashboards/api-health.json`              | Pre-configured API health dashboard                                                                   |

---

## Docker Compose Commands

### Build and Run

```bash
# Build Docker image
docker-compose build

# Start container in detached mode
docker-compose up -d

# Start with log output
docker-compose up

# Rebuild and start
docker-compose up -d --build
```

### View Logs

```bash
# Follow logs in real-time
docker-compose logs -f

# View logs for last 100 lines
docker-compose logs --tail=100

# View logs with timestamps
docker-compose logs -f -t

# View persistent logs from named volume
docker-compose exec medicaments-api ls -la /app/logs/
docker-compose exec medicaments-api tail -f /app/logs/app-*.log
```

### Container Management

```bash
# Check container status
docker-compose ps

# View detailed container info
docker inspect medicaments-api

# View resource usage
docker stats medicaments-api

# Restart container
docker-compose restart

# Stop container
docker-compose stop

# Stop and remove containers
docker-compose down

# Remove containers and volumes
docker-compose down -v

# Remove containers, volumes, and images
docker-compose down -v --rmi all
```

---

## API Endpoints

Access all endpoints via `http://localhost:8030`

### V1 Endpoints (Recommended)

```bash
# Health check
curl http://localhost:8030/health

# Get all medicaments (paginated)
curl http://localhost:8030/v1/medicaments?page=1

# Search by name
curl http://localhost:8030/v1/medicaments?search=paracetamol

# Lookup by CIS
curl http://localhost:8030/v1/medicaments?cis=61504672

# Lookup by CIP
curl http://localhost:8030/v1/medicaments?cip=3400936403114

# Get generics by libellé
curl http://localhost:8030/v1/generiques?libelle=paracetamol

# Get generics by group ID
curl http://localhost:8030/v1/generiques?group=1234

# Get presentations by CIP
curl http://localhost:8030/v1/presentations?cip=3400936403114

# Export all data
curl http://localhost:8030/v1/medicaments?export=all
```

### Documentation

```bash
# Interactive Swagger UI
open http://localhost:8030/docs

# OpenAPI specification
curl http://localhost:8030/docs/openapi.yaml
```

---

## Data Management

### Data Download

The application automatically downloads BDPM data from external sources:

- **Initial Download**: Happens on container startup (takes 10-30 seconds)
- **Automatic Updates**: Scheduled twice daily (6h and 18h)
- **Zero-Downtime**: Updates don't interrupt API access

Monitor data download:

```bash
# Watch logs during startup
docker-compose logs -f

# Check data status via health endpoint
curl http://localhost:8030/health | jq '.data'
```

### Health Checks

The container includes a health check using `/health` endpoint:

- **Interval**: 30 seconds
- **Timeout**: 5 seconds
- **Retries**: 3
- **Start Period**: 10 seconds

Check health status:

```bash
# Check Docker health status
docker-compose ps

# Check health endpoint
curl http://localhost:8030/health

# Health response example
{
  "status": "healthy",
  "last_update": "2025-02-08T12:00:00+01:00",
  "data_age_hours": 0.5,
  "medicaments": 15822,
  "generiques": 1645,
  "is_updating": false
}
```

### Diagnostics

For detailed system metrics and data integrity information, use the `/v1/diagnostics` endpoint:

**What it returns:**
- System metrics: uptime, goroutines, memory usage
- Data age and next scheduled update
- Data integrity checks: orphaned records, missing associations
- Sample CIS codes for troubleshooting

**Example usage:**

```bash
# Get full diagnostics
curl http://localhost:8030/v1/diagnostics | jq

# System metrics only
curl http://localhost:8030/v1/diagnostics | jq '.system'

# Memory usage
curl http://localhost:8030/v1/diagnostics | jq '.system.memory'

# Data integrity summary
curl http://localhost:8030/v1/diagnostics | jq '.data_integrity'

# Uptime
curl http://localhost:8030/v1/diagnostics | jq '.uptime_seconds'
```

**Response example:**

```json
{
  "timestamp": "2025-02-08T13:00:00+01:00",
  "uptime_seconds": 3600,
  "next_update": "2025-02-08T18:00:00+01:00",
  "data_age_hours": 0.3,
  "system": {
    "goroutines": 16,
    "memory": {
      "alloc_mb": 45,
      "num_gc": 20,
      "sys_mb": 65
    }
  },
  "data_integrity": {
    "medicaments_without_conditions": {"count": 3368, "sample_cis": [...]},
    "medicaments_without_generiques": {"count": 7714, "sample_cis": [...]},
    "medicaments_without_presentations": {"count": 1267, "sample_cis": [...]},
    "medicaments_without_compositions": {"count": 2, "sample_cis": [...]},
    "generique_only_cis": {"count": 2440, "sample_cis": [...]},
    "presentations_with_orphaned_cis": {"count": 6, "sample_cip": [...]}
  }
}
```

**Data Integrity Checks:**

| Check | Description | Sample Field |
|--------|-------------|---------------|
| `medicaments_without_conditions` | Medicaments missing condition data | `sample_cis` |
| `medicaments_without_generiques` | Medicaments not in generic groups | `sample_cis` |
| `medicaments_without_presentations` | Medicaments missing presentation data | `sample_cis` |
| `medicaments_without_compositions` | Medicaments missing composition data | `sample_cis` |
| `generique_only_cis` | CIS codes only in generic groups | `sample_cis` |
| `presentations_with_orphaned_cis` | Presentations referencing non-existent medicaments | `sample_cip` |
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check for errors
docker-compose logs

# Verify port 8030 is not in use
lsof -i :8030

# Check disk space
df -h

# Rebuild from scratch
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Health Check Failing

```bash
# Check container is running
docker-compose ps

# View health check logs
docker inspect medicaments-api | jq '.[0].State.Health'

# Test health endpoint manually
docker-compose exec medicaments-api wget -O- http://localhost:8000/health

# Check for data download errors
docker-compose logs | grep -i error
```

### Data Download Issues

```bash
# Check network connectivity
docker-compose exec medicaments-api wget -O- https://base-donnees-publique.medicaments.gouv.fr

# View download logs
docker-compose logs | grep -i download

# Restart to trigger download
docker-compose restart
```

### Logs Not Persisting

```bash
# Check volume mount
docker inspect medicaments-api | jq '.[0].Mounts'

# Verify logs directory permissions
ls -la logs/

# Check logs inside container
docker-compose exec medicaments-api ls -la /app/logs/
```

### High Memory Usage

```bash
# Check current memory usage
docker stats medicaments-api

# View memory metrics
curl http://localhost:8030/v1/diagnostics | jq '.system.memory'

# Restart to clear memory
docker-compose restart
```

### Port Conflicts

If port 8030 is already in use:

```bash
# Change port in docker-compose.yml
ports:
  - "8031:8000"  # Use different host port

# Or stop conflicting service
lsof -i :8030
```

### Service Communication Issues

**Grafana can't connect to Prometheus:**

```bash
# Check Grafana datasource configuration
cat observability/grafana/provisioning/datasources/prometheus.yml

# Ensure it uses container port (9090)
# Correct: url: http://prometheus:9090
# Wrong:   url: http://prometheus:9090

# Restart Grafana to reload datasource config
docker-compose restart grafana

# Verify connectivity from Grafana container
docker-compose exec grafana wget -O- http://prometheus:9090/-/ready
```

**Metrics not appearing in Grafana:**

```bash
# Check if Alloy is scraping metrics from medicaments-api
docker-compose logs grafana-alloy | grep -i "medicaments-api:9090"

# Verify medicaments-api metrics endpoint is accessible
curl http://localhost:9090/metrics

# Check if Prometheus is receiving metrics from Alloy
docker-compose logs prometheus | grep -i "received from Alloy"

# Test Prometheus query for app metrics
curl 'http://localhost:9090/api/v1/query?query=http_request_total'
```

---

## Advanced Usage

### Custom Environment Variables

Create a custom `.env` file:

```bash
# Override any environment variable
LOG_LEVEL=debug
LOG_RETENTION_WEEKS=1
```

Then use it:

```bash
docker-compose --env-file .env.custom up -d
```

### Running Multiple Instances

```bash
# Create multiple compose files
cp docker-compose.yml docker-compose.staging.yml

# Edit port mapping in new file
# ports:
#   - "8031:8000"

# Start both instances
docker-compose -f docker-compose.yml up -d
docker-compose -f docker-compose.staging.yml up -d
```

### Debugging

```bash
# Run with debug logging
# Edit .env.docker: LOG_LEVEL=debug
docker-compose restart

# View real-time logs
docker-compose logs -f

# Enter container shell (scratch image has no shell - use logs only)
# docker-compose exec medicaments-api sh  # Not available

# Check processes (scratch image has no ps - use health endpoint)
# docker-compose exec medicaments-api ps aux  # Not available

# Monitor file changes
docker-compose exec medicaments-api ls -la /app/logs/
```

### Performance Testing

```bash
# Install hey (load testing tool)
go install github.com/rakyll/hey@latest

# Test health endpoint
hey -n 1000 -c 10 http://localhost:8030/health

# Test medicament lookup
hey -n 1000 -c 10 http://localhost:8030/v1/medicaments?cis=61504672

# Test search endpoint
hey -n 100 -c 5 http://localhost:8030/v1/medicaments?search=paracetamol
```

---

## Security Considerations

### Non-Root User

Container runs as non-root user (`UID 65534` / `nobody`) for security:

```bash
# Verify user (scratch image may not have whoami)
# docker-compose exec medicaments-api whoami  # May not be available

# Check user ID
docker-compose exec medicaments-api id
```

### Port Exposure Strategy

For security, some services are only exposed internally to the Docker network:

| Service                   | Exposure Level | Rationale                                    |
| ------------------------- | -------------- | -------------------------------------------- |
| medicaments-api (API)     | Host + Network | Required for external API access             |
| medicaments-api (metrics) | Network only   | Scraped by Alloy internally                  |
| loki                      | Network only   | Scraped by Alloy internally                  |
| grafana-alloy             | Host + Network | Optional debugging endpoint                  |
| prometheus                | Host + Network | Required for Grafana UI and external queries |
| grafana                   | Host + Network | Required for dashboard access                |

**Benefits of internal-only exposure:**

- Reduces attack surface from external access
- Prevents unauthorized direct scraping of metrics/logs
- Forces access through controlled Grafana UI
- Maintains observability functionality within Docker network

**To access internal-only services for debugging:**

```bash
# Access Loki from within Docker network
docker-compose exec loki wget -O- 'http://localhost:3100/loki/api/v1/labels'

# Check logs inside containers
docker-compose logs loki
docker-compose logs grafana-alloy
```

### Network Isolation

Container uses custom bridge network for isolation:

```bash
# List networks
docker network ls

# Inspect network
docker network inspect medicaments_medicaments-network
```

### Volume Permissions

Logs directory is owned by `appuser`:

```bash
# Check permissions
ls -la logs/

# Fix permissions if needed
sudo chown -R 1000:1000 logs/
```

---

## Monitoring

### Container Metrics

```bash
# Real-time stats
docker stats medicaments-api

# Specific metrics
docker stats --no-stream medicaments-api
```

### Application Metrics

```bash
# Full health metrics
curl http://localhost:8030/health | jq

# Memory usage only
curl -s http://localhost:8030/v1/diagnostics | jq '.system.memory'

# Data age
curl -s http://localhost:8030/health | jq '.data_age_hours'
```

### Log Monitoring

```bash
# Follow application logs
docker-compose logs -f

# Watch for errors
docker-compose logs -f | grep -i error

# Watch for data updates
docker-compose logs -f | grep -i update

# Count log entries
docker-compose logs | wc -l
```

### Prometheus Alerting

The monitoring stack includes Prometheus alerting rules that automatically detect issues and display alerts in Grafana.

**Alert Rules Location:** `observability/prometheus/alerts/medicaments-api.yml`

**Critical Alerts:**

| Alert              | Description             | Threshold          | Duration |
| ------------------ | ----------------------- | ------------------ | -------- |
| ServiceDown        | Service unreachable     | `up == 0`          | 5m       |
| High5xxErrorRate   | Too many server errors  | 5xx rate > 5%      | 5m       |
| HighTotalErrorRate | Too many errors overall | 4xx+5xx rate > 10% | 5m       |

**Warning Alerts:**

| Alert            | Description            | Threshold               | Duration |
| ---------------- | ---------------------- | ----------------------- | -------- |
| HighLatencyP95   | Slow response times    | P95 latency > 200ms     | 10m      |
| HighRequestRate  | High traffic volume    | Request rate > 1000/sec | 5m       |
| Sustained4xxRate | High client error rate | 4xx rate > 5%           | 10m      |

**Viewing Alerts in Grafana:**

1. Navigate to `http://localhost:3000`
2. Go to **Alerting** → **Alert Rules** (in the left sidebar)
3. Filter by job `medicaments-api`
4. View active alerts, silenced alerts, and alert history

**Customizing Alert Thresholds:**

Edit `observability/prometheus/alerts/medicaments-api.yml` to adjust thresholds:

```yaml
# Example: Change P95 latency threshold
- alert: HighLatencyP95
  expr: |
    histogram_quantile(0.95,
      rate(http_request_duration_seconds_bucket{job="medicaments-api"}[10m])
    ) > 0.5  # Change from 0.2 (200ms) to 0.5 (500ms)
  for: 10m
```

After editing, reload Prometheus configuration:

```bash
# Restart Prometheus to apply changes
docker-compose restart prometheus

# Or use SIGHUP for hot reload (if configured)
docker exec prometheus kill -HUP 1
```

**Health Check Monitoring:**

For monitoring system metrics and data integrity, use the `/v1/diagnostics` endpoint in Grafana alerts. The `/health` endpoint is used for data health status only.

Create a Grafana alert panel based on diagnostics data:

1. Go to **Dashboards** → **medicaments-api Health**
2. Add a new panel or edit existing
3. Set up an alert based on health status or data age
4. Configure alert conditions (e.g., `data_age_hours > 24`)

**Endpoint Usage:**
- `/health` → Data health status (medicaments count, generiques count, data age)
- `/v1/diagnostics` → System metrics + data integrity (uptime, memory, data integrity checks)

---

## Cleanup

### Remove Staging Environment

```bash
# Stop and remove containers
docker-compose down

# Remove persistent logs (optional)
rm -rf logs/

# Remove Docker images
docker rmi medicaments_medicaments-api

# Remove all unused resources
docker system prune -a
```

### Clean Docker Resources

```bash
# Remove stopped containers
docker container prune

# Remove unused volumes
docker volume prune

# Remove unused images
docker image prune

# Remove everything (use with caution)
docker system prune -a --volumes
```

---

## Production Differences

| Feature                 | Staging        | Production          |
| ----------------------- | -------------- | ------------------- |
| **Deployment**          | Docker Compose | SSH + systemd       |
| **Port**                | 8030           | 8000 (configurable) |
| **LOG_LEVEL**           | info           | info                |
| **LOG_RETENTION_WEEKS** | 2              | 4                   |
| **MAX_LOG_FILE_SIZE**   | 50MB           | 100MB               |
| **Resource Limits**     | 512MB/0.5CPU   | None (systemd)      |
| **Logs Location**       | `./logs/`      | Server logs         |

---

## CI/CD Integration

This Docker setup can be integrated with your existing CI/CD pipeline:

```bash
# In CI/CD pipeline
docker-compose -f docker-compose.yml -f docker-compose.ci.yml up -d

# Run tests
docker-compose exec medicaments-api go test ./...

# Get coverage
docker-compose exec medicaments-api go test -coverprofile=coverage.out ./...

# Teardown
docker-compose down -v
```

---

## Support

For issues or questions:

1. Check the [troubleshooting section](#troubleshooting) above
2. Review the main README.md
3. Check application logs: `docker-compose logs -f`
4. Check health status: `curl http://localhost:8030/health`
5. Open an issue on GitHub

---

## Appendix

### Docker Image Details

- **Base Image**: `scratch` (empty filesystem, minimal attack surface)
- **Builder Image**: `golang:1.26-alpine`
- **Final Image Size**: ~8-10MB
- **Binary Size**: ~8-10MB (statically linked, stripped)
- **Supported Architectures**: amd64, arm64
- **Build Tool**: Docker BuildKit with automatic platform detection ($TARGETOS/$TARGETARCH)

### File Locations

| Type          | Location                              |
| ------------- | ------------------------------------- |
| **Binary**    | `/app/medicaments-api`                |
| **HTML Docs** | `/app/html/`                          |
| **Logs**      | `/app/logs/` (mounted to `logs_data`) |
| **Config**    | Environment variables                 |

### Startup Process

1. Container starts as non-root user (UID 65534/nobody)
2. Application loads environment variables from `.env.docker`
3. Logging system initialized
4. Data container and parser created
5. Scheduler starts (6h/18h updates)
6. BDPM data downloaded from external sources
7. HTTP server starts on port 8000
8. Docker healthcheck begins after 10s start period
9. Grafana Alloy starts collecting logs and metrics
10. Loki and Prometheus begin receiving data

### Tips

- Container downloads BDPM data on first startup (10-30s)
- Health check passes after ~10s start period
- Logs persist even after container removal (volume mount)
- Use `docker-compose exec medicaments-api sh` to enter container (if available)
- Check the [troubleshooting section](#troubleshooting) for detailed help

---

**Last updated: 2026-02-17**
