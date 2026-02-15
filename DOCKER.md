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
# Option 1: Interactive script (recommended)
./docker-staging.sh

# Option 2: Docker Compose
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
2. **Container starts** as non-root user (appuser:1000)
3. **BDPM data downloads** from external sources (~10-30 seconds)
4. **HTTP server starts** on port 8000
5. **Health check begins** after 40-second start period
6. **API is ready** at http://localhost:8030

---

## Essential Commands

### 🚀 Start & Stop

```bash
./docker-staging.sh              # Interactive menu
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
docker stats medicaments-api-staging # Resource usage
```

### 🛠️ Build & Rebuild

```bash
docker-compose build             # Build image
docker-compose up -d --build     # Rebuild & start
docker-compose build --no-cache  # Clean build (no cache)
```

### 🗑️ Cleanup

```bash
docker-compose down -v           # Stop, remove containers & volumes
docker-compose down -v --rmi all # Remove everything (including images)
docker system prune -a           # Remove all unused Docker resources
```

---

## Project Overview

### What Was Created

The following files were added to set up your Docker staging environment:

#### 1. **Dockerfile**

Multi-stage Docker build optimized for production:

- **Stage 1 - Builder**: `golang:1.24-alpine`
- **Stage 2 - Runtime**: `alpine:3.20` (~15-20MB final image)
- **Security**: Non-root user (appuser:1000)
- **Health Check**: Built-in using `/health` endpoint
- **Files**: Copies binary and HTML documentation

#### 2. **docker-compose.yml**

Docker Compose orchestration:

- **Port Mapping**: 8030 (host) → 8000 (container)
- **Environment**: Variables from `.env.staging`
- **Logs**: Persistent via volume mount (`./logs:/app/logs`)
- **Resources**: 512MB RAM, 0.5 CPU limits
- **Health Check**: 30s interval, 10s timeout, 3 retries
- **Restart**: Policy `unless-stopped`
- **Network**: Custom bridge network for isolation

#### 3. **.dockerignore**

Optimizes Docker build context:

- Excludes: logs, git, vendor, test files, \*.md (except README.md)
- Keeps: source code and HTML docs
- Reduces: build time and image size

#### 4. **.env.staging**

Staging environment configuration:
| Variable | Value | Description |
|----------|-------|-------------|
| `ADDRESS` | `0.0.0.0` | Listen on all interfaces in container |
| `PORT` | `8000` | Port inside container |
| `ENV` | `staging` | Environment mode |
| `LOG_LEVEL` | `info` | Logging level (debug/info/warn/error) |
| `LOG_RETENTION_WEEKS` | `2` | Keep logs for 2 weeks |
| `MAX_LOG_FILE_SIZE` | `52428800` | Rotate at 50MB |
| `MAX_REQUEST_BODY` | `2097152` | 2MB max request body |
| `MAX_HEADER_SIZE` | `2097152` | 2MB max header size |

#### 5. **docker-staging.sh**

Interactive quick-start script:

- Validates Docker/Docker Compose installation
- Creates logs directory
- Provides menu for common operations
- Waits for application to be ready
- Shows health check on startup

#### 6. **.gitignore** (updated)

Added `.env.staging` to prevent committing staging configuration.

### Project Structure

```
medicaments-api/
├── Dockerfile              # Multi-stage Docker build
├── docker-compose.yml      # Docker Compose orchestration (includes observability stack)
├── .dockerignore          # Files excluded from build context
├── .env.staging           # Staging environment variables
├── docker-staging.sh      # Interactive quick-start script
├── logs/                  # Persistent logs directory
├── html/                  # Documentation files (served by API)
├── observability/         # Grafana stack configuration
│   ├── alloy/              # Alloy config
│   ├── loki/               # Loki config
│   ├── prometheus/          # Prometheus config
│   └── grafana/             # Grafana config
│       └── provisioning/      # Auto-provisioning
│           ├── datasources/   # Loki & Prometheus datasources
│           └── dashboards/     # Dashboard imports
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
| medicaments-api | 9090 (metrics) | 9090      | http://localhost:9090/metrics  | medicaments-api:9090   |
| grafana-alloy   | 12345          | 12345     | http://localhost:12345/metrics | grafana-alloy:12345    |
| loki            | 3100           | 3150      | http://localhost:3150          | loki:3100              |
| prometheus      | 9090           | 9091      | http://localhost:9091          | prometheus:9090        |
| grafana         | 3000           | 3000      | http://localhost:3000          | grafana:3000           |

**Key Points:**

- Grafana connects to Prometheus at `prometheus:9090` (container port)
- External access to Prometheus is via `localhost:9091` (host port mapping)
- All service-to-service communication uses container ports within Docker network
- Host ports are only for accessing services from the host machine

### Observability Services

#### grafana-alloy

Collects logs and metrics from medicaments-api.

- **Image**: `grafana/alloy:v1.4.0`
- **Configuration**: `alloy/config.alloy`
- **Port**: 12345 (Alloy's own metrics)
- **Functions**:
  - Read logs from `./logs/` directory
  - Scrape metrics from `medicaments-api:9090/metrics`
  - Forward to local Loki and Prometheus
- **Resource Usage**: ~150MB RAM

#### loki

Log aggregation and storage.

- **Image**: `grafana/loki:2.9.10`
- **Configuration**: `loki/config.yaml`
- **Port**: 3150
- **Storage**: Filesystem (chunks in `/loki/chunks`, rules in `/loki/rules`)
- **Retention**: 30 days
- **Data Volume**: `loki-data`
- **Resource Usage**: ~100MB RAM + ~100MB disk

#### prometheus

Metric storage and querying.

- **Image**: `prom/prometheus:v2.48.0`
- **Configuration**: `prometheus/prometheus.yml`
- **Port**: 9091 (host) → 9090 (container)
  - Host port 9091 provides external access to Prometheus UI
  - Container port 9090 is used for service-to-service communication
- **Retention**: 15 days (default)
- **Data Volume**: `prometheus-data`
- **Resource Usage**: ~150MB RAM + ~200MB disk

#### grafana

Visualization for logs and metrics.

- **Image**: `grafana/grafana:10.2.4`
- **Port**: 3000
- **Default Credentials**: admin/admin (change after first login)
- **Data Volume**: `grafana-data`
- **Resource Usage**: ~200MB RAM + ~50MB disk
- **Auto-Provisioning**: Datasources configured automatically

### Observability Access Points

```bash
# Grafana UI (visualization)
open http://localhost:3000

# Prometheus UI (metrics browsing)
open http://localhost:9091

# Loki API (log queries)
curl http://localhost:3150/loki/api/v1/labels

# Alloy metrics (collector status)
curl http://localhost:12345/metrics

# medicaments-api metrics (application metrics)
curl http://localhost:9090/metrics
```

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

**Note**: Alloy configuration filters out Go runtime metrics, keeping only HTTP metrics.

### Observability Default Credentials

**Grafana**:

- Username: `admin`
- Password: `admin`
- **Important**: Change password after first login (Configuration → Users → Change Password)

**Other Services**:

- No authentication required (local network only)

### Observability Resource Usage

| Service         | RAM        | Disk          | Retention      |
| --------------- | ---------- | ------------- | -------------- |
| medicaments-api | ~50MB      | ~20MB         | N/A            |
| grafana-alloy   | ~150MB     | ~10MB         | N/A            |
| loki            | ~100MB     | ~100MB (data) | 30 days        |
| prometheus      | ~150MB     | ~200MB (data) | 30 days        |
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

# Test Loki query
curl -G 'http://localhost:3150/loki/api/v1/query_range' \
  --data-urlencode 'query={app="medicaments_api"}' \
  --data-urlencode 'start=1699488000000000000' \
  --data-urlencode 'end=1699491600000000000'
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
curl 'http://localhost:9091/api/v1/query?query=http_request_total'
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

| File                                                            | Purpose                                                                     |
| --------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `observability/alloy/config.alloy`                              | Alloy configuration (logs + metrics collection)                             |
| `observability/loki/config.yaml`                                | Loki configuration (log storage, filesystem ruler storage, 30-day retention, 16MB/sec ingestion rate) |
| `observability/prometheus/prometheus.yml`                       | Prometheus configuration (metric storage, 30-day retention)                 |
| `observability/grafana/provisioning/datasources/loki.yml`       | Auto-configure Loki datasource                                              |
| `observability/grafana/provisioning/datasources/prometheus.yml` | Auto-configure Prometheus datasource                                        |
| `observability/grafana/provisioning/dashboards/dashboard.yml`   | Auto-import Grafana dashboards                                              |

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

# View persistent logs from volume
ls -la logs/
tail -f logs/app-*.log
```

### Container Management

```bash
# Check container status
docker-compose ps

# View detailed container info
docker inspect medicaments-api-staging

# View resource usage
docker stats medicaments-api-staging

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
- **Timeout**: 10 seconds
- **Retries**: 3
- **Start Period**: 40 seconds

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
  "uptime_seconds": 3600,
  "data": {
    "api_version": "1.0",
    "generiques": 1628,
    "is_updating": false,
    "medicaments": 15803,
    "next_update": "2025-02-08T18:00:00+01:00"
  },
  "system": {
    "goroutines": 16,
    "memory": {
      "alloc_mb": 45,
      "num_gc": 20,
      "sys_mb": 65,
      "total_alloc_mb": 150
    }
  }
}
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
docker inspect medicaments-api-staging | jq '.[0].State.Health'

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
docker inspect medicaments-api-staging | jq '.[0].Mounts'

# Verify logs directory permissions
ls -la logs/

# Check logs inside container
docker-compose exec medicaments-api ls -la /app/logs/
```

### High Memory Usage

```bash
# Check current memory usage
docker stats medicaments-api-staging

# View memory metrics
curl http://localhost:8030/health | jq '.system.memory'

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
# Wrong:   url: http://prometheus:9091

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
curl 'http://localhost:9091/api/v1/query?query=http_request_total'
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
# Edit .env.staging: LOG_LEVEL=debug
docker-compose restart

# View real-time logs
docker-compose logs -f

# Enter container shell
docker-compose exec medicaments-api sh

# Check processes
docker-compose exec medicaments-api ps aux

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

Container runs as non-root user (`appuser:1000`) for security:

```bash
# Verify user
docker-compose exec medicaments-api whoami

# Check user ID
docker-compose exec medicaments-api id
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
docker stats medicaments-api-staging

# Specific metrics
docker stats --no-stream medicaments-api-staging
```

### Application Metrics

```bash
# Full health metrics
curl http://localhost:8030/health | jq

# Memory usage only
curl -s http://localhost:8030/health | jq '.system.memory'

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

- **Base Image**: `alpine:3.20` (~5MB)
- **Builder Image**: `golang:1.24-alpine`
- **Final Image Size**: ~15-20MB
- **Binary Size**: ~8-10MB (stripped)

### File Locations

| Type          | Location                            |
| ------------- | ----------------------------------- |
| **Binary**    | `/app/medicaments-api`              |
| **HTML Docs** | `/app/html/`                        |
| **Logs**      | `/app/logs/` (mounted to `./logs/`) |
| **Config**    | Environment variables               |

### Startup Process

1. Container starts as `appuser`
2. Application loads environment variables
3. Logging system initialized
4. Data container and parser created
5. Scheduler starts (6h/18h updates)
6. BDPM data downloaded
7. HTTP server starts on port 8000
8. Health check begins after 40s

### Tips

- Container downloads BDPM data on first startup (10-30s)
- Health check passes after ~40s start period
- Logs persist even after container removal (volume mount)
- Use `docker-compose exec medicaments-api sh` to enter container
- Check the [troubleshooting section](#troubleshooting) for detailed help

---

**Last updated: 2026-02-11**
