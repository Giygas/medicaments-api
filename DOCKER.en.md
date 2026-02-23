# Docker Setup Guide

**Complete guide for running medicaments-api in Docker**

---

[🇫🇷 Français](DOCKER.md) | **🇬🇧 English**

---

## Table of Contents

- [Quick Start](#quick-start)
- [Essential Commands](#essential-commands)
- [Project Overview](#project-overview)
  - [What Was Created](#what-was-created)
  - [Project Structure](#project-structure)
  - [Configuration](#configuration)
  - [Docker compose Commands](#docker-compose-commands)
  - [Build and Run](#build-and-run)
  - [View Logs](#view-logs)
  - [Container Management](#container-management)
- [API Endpoints](#api-endpoints)
- [Data Management](#data-management)
- [Troubleshooting](#troubleshooting)
- [Advanced Usage](#advanced-usage)
- [Security Considerations](#security-considerations)
- [Monitoring](#monitoring)
- [Cleanup](#cleanup)
- [Production Differences](#production-differences)
- [CI/CD Integration](#cicd-integration)
- [Observability Stack](#observability-stack)
- [Support](#support)
- [Appendix](#appendix)

---

## Quick Start

### Prerequisites

- Docker Engine 20.10+ or Docker Desktop 4.0+
- At least 1GB available disk space
- Network connection for BDPM data download
- Secrets setup: Run `make setup-secrets` (creates `secrets/grafana_password.txt`)

### Secrets Setup (Required First Step)

Before running Docker services, you need to set up the Grafana password secret:

```bash
# Create Grafana password secret
make setup-secrets

# This prompts for a password and creates secrets/grafana_password.txt with secure permissions (600)
```

**Why Secrets?**

- Grafana requires an admin password for secure access
- Storing passwords in environment variables or configuration files is insecure
- Docker secrets provide a secure way to manage sensitive data
- The `secrets/` directory is excluded from version control (`.gitignore`)

**Security Best Practices:**

- ✅ Use strong passwords (minimum 12 characters, mixed case, numbers, symbols)
- ✅ Never commit secrets to version control
- ✅ Set restrictive file permissions (600)
- ❌ Don't reuse passwords across services

### Get Started Immediately

```bash
# Docker compose (recommended)
docker compose up -d

# View logs
docker compose logs -f

# Check health
curl http://localhost:8030/health

# Stop
docker compose down
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
docker compose up -d             # Start detached
docker compose down               # Stop & remove containers
docker compose restart            # Restart container
```

### 📋 Logs

```bash
docker compose logs -f           # Follow logs in real-time
docker compose logs --tail=100   # Last 100 lines
docker compose logs | grep error   # Search for errors
```

### 🔍 Status & Health

```bash
docker compose ps                 # Container status
curl http://localhost:8030/health # Health check
docker stats medicaments-api # Resource usage
```

### 🛠️ Build & Rebuild

```bash
docker compose build             # Build image
docker compose up -d --build     # Rebuild & start
docker compose build --no-cache  # Clean build (no cache)
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

**Docker compose (auto-detects platform):**

```bash
docker compose up -d    # Builds for your native platform
docker compose build      # Builds for your native platform
```

**Supported Platforms:**

| Architecture | Description      | Target Platforms                                       |
| ------------ | ---------------- | ------------------------------------------------------ |
| **amd64**    | Intel/AMD x86_64 | Intel/AMD servers, cloud instances, Intel Macs         |
| **arm64**    | ARM 64-bit       | Apple Silicon (M1/M2/M3), Raspberry Pi 4, AWS Graviton |

**Note:** Use `--load` flag to make image available locally. Without it, image only exists in BuildKit cache.

---

## Project Overview

### What Was Created

The following files were added to set up your Docker environment:

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

Docker compose orchestration:

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
├── secrets/              # Docker secrets (gitignored)
│   └── grafana_password.txt
└── observability/         # Grafana stack configuration
    ├── alloy/              # Alloy config
    ├── loki/               # Loki config
    ├── prometheus/          # Prometheus config
    │   └── alerts/          # Prometheus alert rules
    └── grafana/             # Grafana config
        ├── provisioning/      # Auto-provisioning
        │   ├── datasources/   # Loki & Prometheus datasources
        │   └── dashboards/     # Dashboard provisioning
        └── dashboards/        # Dashboard JSON files
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

## Docker compose Commands

### Build and Run

```bash
# Build Docker image
docker compose build

# Start container in detached mode
docker compose up -d

# Start with log output
docker compose up

# Rebuild and start
docker compose up -d --build
```

### View Logs

```bash
# Follow logs in real-time
docker compose logs -f

# View logs for last 100 lines
docker compose logs --tail=100

# View logs with timestamps
docker compose logs -f -t

# View persistent logs from named volume
docker compose exec medicaments-api ls -la /app/logs/
docker compose exec medicaments-api tail -f /app/logs/app-*.log
```

### Container Management

```bash
# Check container status
docker compose ps

# View detailed container info
docker inspect medicaments-api

# View resource usage
docker stats medicaments-api

# Restart container
docker compose restart

# Stop container
docker compose stop

# Stop and remove containers
docker compose down

# Remove containers and volumes
docker compose down -v

# Remove containers, volumes, and images
docker compose down -v --rmi all
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
curl http://localhost:8030/v1/medicaments/export
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
docker compose logs -f

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
docker compose ps

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

| Check                               | Description                                        | Sample Field |
| ----------------------------------- | -------------------------------------------------- | ------------ |
| `medicaments_without_conditions`    | Medicaments missing condition data                 | `sample_cis` |
| `medicaments_without_generiques`    | Medicaments not in generic groups                  | `sample_cis` |
| `medicaments_without_presentations` | Medicaments missing presentation data              | `sample_cis` |
| `medicaments_without_compositions`  | Medicaments missing composition data               | `sample_cis` |
| `generique_only_cis`                | CIS codes only in generic groups                   | `sample_cis` |
| `presentations_with_orphaned_cis`   | Presentations referencing non-existent medicaments | `sample_cip` |

---

## Troubleshooting

### Container Won't Start

```bash
# Check for errors
docker compose logs

# Verify port 8030 is not in use
lsof -i :8030

# Check disk space
df -h

# Rebuild from scratch
docker compose down
docker compose build --no-cache
docker compose up -d
```

### Health Check Failing

```bash
# Check container is running
docker compose ps

# View health check logs
docker inspect medicaments-api | jq '.[0].State.Health'

# Test health endpoint manually
docker compose exec medicaments-api wget -O- http://localhost:8000/health

# Check for data download errors
docker compose logs | grep -i error
```

### Data Download Issues

```bash
# Check network connectivity
docker compose exec medicaments-api wget -O- https://base-donnees-publique.medicaments.gouv.fr

# View download logs
docker compose logs | grep -i download

# Restart to trigger download
docker compose restart
```

### Logs Not Persisting

```bash
# Check volume mount
docker inspect medicaments-api | jq '.[0].Mounts'

# Verify logs directory permissions
ls -la logs/

# Check logs inside container
docker compose exec medicaments-api ls -la /app/logs/
```

### High Memory Usage

```bash
# Check current memory usage
docker stats medicaments-api

# View memory metrics
curl http://localhost:8030/v1/diagnostics | jq '.system.memory'

# Restart to clear memory
docker compose restart
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

### Secrets File Missing

If you encounter this error:

```
ERROR: for grafana  Cannot create container for service grafana:
stat /path/to/secrets/grafana_password.txt: no such file or directory
```

**Solution:**

```bash
# Option 1: Use Make (recommended)
make setup-secrets

# Option 2: Create manually
mkdir -p secrets
echo "your-secure-password" > secrets/grafana_password.txt
chmod 600 secrets/grafana_password.txt

# Option 3: Validate existing secrets
make validate-secrets
```

**Verify secrets are working:**

```bash
# Check file exists with correct permissions
ls -la secrets/grafana_password.txt

# Expected: -rw------- 1 user group date secrets/grafana_password.txt
```

### Observability Issues

For detailed troubleshooting of Grafana, Loki, Prometheus, and Alloy issues, see [OBSERVABILITY.md](OBSERVABILITY.md#troubleshooting).

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
docker compose --env-file .env.custom up -d
```

### Running Multiple Instances

```bash
# Create multiple compose files
cp docker-compose.yml docker-compose.staging.yml

# Edit port mapping in new file
# ports:
#   - "8031:8000"

# Start both instances
docker compose -f docker-compose.yml up -d
docker compose -f docker-compose.staging.yml up -d
```

### Debugging

```bash
# Run with debug logging
# Edit .env.docker: LOG_LEVEL=debug
docker compose restart

# View real-time logs
docker compose logs -f

# Enter container shell (scratch image has no shell - use logs only)
# docker compose exec medicaments-api sh  # Not available

# Check processes (scratch image has no ps - use health endpoint)
# docker compose exec medicaments-api ps aux  # Not available

# Monitor file changes
docker compose exec medicaments-api ls -la /app/logs/
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
# docker compose exec medicaments-api whoami  # May not be available

# Check user ID
docker compose exec medicaments-api id
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
docker compose exec loki wget -O- 'http://localhost:3100/loki/api/v1/labels'

# Check logs inside containers
docker compose logs loki
docker compose logs grafana-alloy
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
docker compose logs -f

# Watch for errors
docker compose logs -f | grep -i error

# Watch for data updates
docker compose logs -f | grep -i update

# Count log entries
docker compose logs | wc -l
```

### Prometheus Metrics Endpoint

The application exposes Prometheus metrics on internal port 9090, accessible only within the Docker network. Alloy collects these metrics automatically.

**To view metrics:**

1. Via Prometheus UI: http://localhost:9090 → search for `http_request_total`, `http_request_duration_seconds`, `http_request_in_flight`
2. Via Grafana: http://localhost:3000 → pre-configured dashboards
3. Via Alloy (development): `docker compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics`

**Metrics available:**

- `http_request_total` - Total HTTP requests with method, path, status labels
- `http_request_duration_seconds` - Request latency histogram
- `http_request_in_flight` - Current in-flight requests

For detailed observability setup with Grafana dashboards, alerts, and log aggregation, see [OBSERVABILITY.md](OBSERVABILITY.md).

---

## Cleanup

### Remove Staging Environment

```bash
# Stop and remove containers
docker compose down

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
| **Deployment**          | Docker compose | SSH + systemd       |
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
docker compose -f docker-compose.yml -f docker-compose.ci.yml up -d

# Run tests
docker compose exec medicaments-api go test ./...

# Get coverage
docker compose exec medicaments-api go test -coverprofile=coverage.out ./...

# Teardown
docker compose down -v
```

---

## Observability Stack

For complete observability setup (Grafana, Loki, Prometheus, Alloy), see [OBSERVABILITY.md](OBSERVABILITY.md).

**Quick access:**

- Grafana: http://localhost:3000
- Prometheus: http://localhost:9090
- Credentials: see secrets/grafana_password.txt

---

## Support

For issues or questions:

1. Check the [troubleshooting section](#troubleshooting) above
2. See [OBSERVABILITY.md](OBSERVABILITY.md) for observability-specific issues
3. Review the main README.md
4. Check application logs: `docker compose logs -f`
5. Check health status: `curl http://localhost:8030/health`
6. Open an issue on GitHub

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
2. Observability stack (`loki`, `prometheus`, `grafana`) starts via submodule
3. `grafana-alloy` starts via app's docker-compose.yml, after medicaments-api healthcheck
4. Application loads environment variables from `.env.docker`
5. Logging system initialized
6. Data container and parser created
7. Scheduler starts (6h/18h updates)
8. BDPM data downloaded from external sources
9. HTTP server starts on port 8000
10. Docker healthcheck passes after 10s start period
11. Grafana Alloy starts collecting logs and metrics
12. Loki and Prometheus begin receiving data via Alloy

### Tips

- Container downloads BDPM data on first startup (10-30s)
- Health check passes after ~10s start period
- Logs persist even after container removal (volume mount)
- Use `docker compose exec medicaments-api sh` to enter container (if available)
- Check the [troubleshooting section](#troubleshooting) for detailed help
