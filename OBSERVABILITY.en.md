# Observability Guide - medicaments-api

**Guide for configuring and using the observability stack with medicaments-api**

---

[🇫🇷 Français](OBSERVABILITY.md) | **🇬🇧 English**

---

## Overview

The staging setup includes a complete observability stack with Grafana, Loki, Prometheus, and Alloy for logs and metrics monitoring.

**Architecture:**

The observability stack is organized via a separate **Git submodule**:

- **docker-compose.yml** (application): Contains `medicaments-api` and `grafana-alloy`
- **observability/** (submodule): Contains `loki`, `prometheus`, and `grafana`

Both components are connected via the external `obs-network` created by the submodule.

**Components:**

- **Grafana Alloy**: Collector agent that gathers logs and metrics
- **Loki**: Log aggregation and storage
- **Prometheus**: Metric storage and querying
- **Grafana**: Visualization and dashboards

**Benefits:**

- Centralized log viewing and searching
- Real-time metrics monitoring
- Alerting on service health and performance
- Pre-configured dashboards for quick insights

## Table of Contents

- [Overview](#overview)
- [Quick Setup](#quick-setup)
- [Architecture](#architecture)
- [Configuration Modes](#configuration-modes)
  - [Local Mode (Default)](#local-mode-default)
  - [Remote Mode (Production)](#remote-mode-production)
- [Application Configuration](#application-configuration)
  - [Grafana Alloy Configuration](#grafana-alloy-configuration)
  - [Environment Variables](#environment-variables)
- [Submodule Management](#submodule-management)
- [Submodule Documentation](#submodule-documentation)
- [Application Metrics](#application-metrics)
- [Log Format](#log-format)
- [Access Points](#access-points)
- [Troubleshooting](#troubleshooting)

---

## Quick Setup

### Prerequisites

- Docker installed
- Git installed
- Permissions to run `make`

### Installation

```bash
# 1. Initialize submodule (first time only)
make obs-init

# 2. Configure secrets
make setup-secrets

# 3. Start all services
make up
```

### Verification

```bash
# Check status
make ps

# Access Grafana
open http://localhost:3000

# Credentials: admin / (password in observability/secrets/grafana_password.txt)
```

---

## Architecture

### Global Diagram

```
┌───────────────────────────────────────────────────────────┐
│   medicaments-api/                                        │
│   (docker-compose.yml)                                    │
│                                                           │
│  ┌─────────────────┐       ┌─────────────────┐            │
│  │ medicaments-api │◀──────│ grafana-alloy   │            │
│  │  (logs/metrics) │       │   (collector)   │            │
│  └────────┬────────┘       └────────┬────────┘            │
│           │ logs volume             │                     │
│           └────────────────▶────────┘                     │
│                                  │  │                     │
│                    Network: obs-network (external)        │
└──────────────────────────────────┼──┼─────────────────────┘
                                   │  │
┌──────────────────────────────────┼──┼─────────────────────┐
│    observability/                │  │                     │
│    (git submodule)               │  │                     │
│                                  │  │                     │
│                       ┌──────────┘  └─────────┐           │
│                       │                       │           │
│                  ┌────▼────┐           ┌──────▼─────┐     │
│                  │  loki   │           │ prometheus │     │
│                  │ (logs)  │           │  (metrics) │     │
│                  └────┬────┘           └──────┬─────┘     │
│                       └──────────┬────────────┘           │
│                                  │                        │
│                           ┌──────▼────────┐               │
│                           │    grafana    │               │
│                           │(visualization)│               │
│                           └───────────────┘               │
└───────────────────────────────────────────────────────────┘
```

**Data Flow:**

1. **medicaments-api** generates logs (to `/app/logs/`) and metrics (to `/metrics` endpoint)
2. **Grafana Alloy** reads log files and scrapes metrics endpoints
3. **Loki** stores logs from Alloy
4. **Prometheus** stores metrics from Alloy via remote_write
5. **Grafana** queries both Loki and Prometheus for visualization

**Network:**

- The `obs-network` is **external** and created by the `observability/` submodule
- Both `docker-compose.yml` files use this network for inter-container communication
- The network is shared between the application and the observability stack

---

## Configuration Modes

### Local Mode (Default)

In local mode, Alloy connects directly to observability services via Docker container DNS:

**Configuration:**

- Environment variable: `ALLOY_CONFIG=config.alloy` (or leave empty)
- Grafana Alloy connects to:
  - `http://loki:3100` for logs
  - `http://prometheus:9090` for metrics

**Recommended Usage:**

- Local development
- Staging on the same machine
- Testing and validation

**Startup:**

```bash
make setup-secrets  # Secrets setup (first time)
make obs-init        # Initialize submodule (first time)
make up              # Start all services
```

### Remote Mode (Production)

In remote mode, Alloy connects to remote endpoints via secure tunnels:

**Configuration:**

- Environment variable: `ALLOY_CONFIG=config.remote.alloy`
- Required environment variables:
  - `PROMETHEUS_URL`: Remote Prometheus URL (ex: `https://prometheus-obs.example.com/api/v1/write`)
  - `LOKI_URL`: Remote Loki URL (ex: `https://loki-obs.example.com/loki/api/v1/push`)
- Tunnel options:
  - **Cloudflare Tunnel**: HTTPS URLs with Cloudflare Access
  - **Tailscale VPN**: HTTP URLs with private IP address
  - **VPN/Mesh**: HTTP URLs with private network

**Recommended Usage:**

- Production with centralized infrastructure
- Multi-site monitoring
- Cloud environments

**Configuration Example:**

```bash
# .env.docker
ALLOY_CONFIG=config.remote.alloy

# Cloudflare Tunnel
PROMETHEUS_URL=https://prometheus-obs.yourdomain.com/api/v1/write
LOKI_URL=https://loki-obs.yourdomain.com/loki/api/v1/push

# Cloudflare Access (optional)
CF_ACCESS_CLIENT_ID=your_client_id
CF_ACCESS_CLIENT_SECRET=your_client_secret

# Tailscale VPN
# PROMETHEUS_URL=http://100.x.x.x:9090/api/v1/write
# LOKI_URL=http://100.x.x.x:3100/loki/api/v1/push
```

**Startup:**

```bash
make setup-secrets  # Secrets setup (first time)
make obs-init        # Initialize submodule (first time)
make up              # Start all services
```

**Failover Protection:**
Remote mode uses Alloy's WAL (Write-Ahead Log) buffer to protect data during network outages:

- 2.5GB buffer
- Protection variable depending on data volume (several hours to several days depending on traffic)
- Automatic recovery on connection restoration

---

## Application Configuration

### Grafana Alloy Configuration

medicaments-api uses Grafana Alloy to collect logs and metrics. Alloy configurations are located in `configs/alloy/`:

| File                    | Mode        | Description                                            |
| -------------------------- | ----------- | ------------------------------------------------------ |
| `configs/alloy/config.alloy` | Local       | Direct connection to Loki/Prometheus via Docker network |
| `configs/alloy/config.remote.alloy` | Remote      | Connection via tunnel with auth and WAL buffering        |

**Main functions of Alloy configuration:**

- **Log collection**: Reading JSON files from `/var/log/app/*.log`
- **Log parsing**: Extracting labels (level, path, status, duration_ms)
- **DEBUG filtering**: Removing DEBUG logs before storing in Loki
- **Metrics scraping**: Retrieving HTTP metrics from `medicaments-api:9090/metrics`
- **System metrics**: Collecting CPU, memory, disk, network metrics

**Note**: Alloy configurations are specific to medicaments-api and are not part of the submodule.

### Environment Variables

Observability environment variables are defined in `.env.docker`:

| Variable                    | Default Value          | Description                                          |
| --------------------------- | --------------------- | ---------------------------------------------------- |
| `ALLOY_CONFIG`             | `config.alloy`         | Alloy configuration file to use                       |
| `PROMETHEUS_URL`          | -                      | Remote Prometheus URL (remote mode only)           |
| `LOKI_URL`                 | -                      | Remote Loki URL (remote mode only)                 |
| `CF_ACCESS_CLIENT_ID`      | -                      | Cloudflare Access client ID (optional, remote mode)    |
| `CF_ACCESS_CLIENT_SECRET`    | -                      | Cloudflare Access client secret (optional, remote mode)  |

**Note**: In local mode, only `ALLOY_CONFIG` is required. In remote mode, tunnel variables are needed.

---

## Submodule Management

The observability submodule is managed via the following Make commands:

### Available Commands

| Command          | Description                                         |
| ----------------- | --------------------------------------------------- |
| `make obs-init`   | Initialize submodule (first time)                    |
| `make obs-up`     | Start observability stack                             |
| `make obs-down`   | Stop observability stack                              |
| `make obs-logs`   | View observability stack logs                         |
| `make obs-status` | Check observability stack status                      |
| `make obs-update` | Update submodule to latest version                   |

### Submodule Structure

```
observability/
├── docker-compose.yml         # Stack orchestration (loki + prometheus + grafana)
├── configs/                  # Stack configurations
│   ├── loki/
│   ├── prometheus/
│   └── grafana/
├── secrets/                 # Stack secrets (gitignored)
│   └── grafana_password.txt
└── docs/                   # Complete stack documentation
    ├── README.md
    ├── local-setup.md
    ├── remote-setup.md
    └── tunnels.md
```

### Submodule Reset

If you have issues with the submodule, you can reset it:

```bash
# Remove submodule
rm -rf .git/modules/observability
git submodule deinit -f observability
git rm -f observability

# Re-initialize
make obs-init
```

---

## Submodule Documentation

For complete observability stack documentation, consult the files in the submodule:

### Main Documentation

- **[observability/README.md](observability/README.md)**: Main observability stack guide
  - Overview and architecture
  - Quick start and operation modes
  - Configuration and Make commands

### Configuration Guides

- **[observability/docs/local-setup.md](observability/docs/local-setup.md)**: Local mode configuration guide (submodule)
  - Adding submodule to your application
  - Configuring shared Docker network
  - Starting and using

- **[observability/docs/remote-setup.md](observability/docs/remote-setup.md)**: Remote mode configuration guide (tunnel)
  - Tunnel configuration (Cloudflare, Tailscale, WireGuard)
  - Authentication and security
  - Connecting remote applications

- **[observability/docs/tunnels.md](observability/docs/tunnels.md)**: Detailed tunnel guide
  - Cloudflare Tunnel with Cloudflare Access configuration
  - Tailscale VPN configuration
  - WireGuard configuration

### Submodule Resources

- **GitHub Repository**: https://github.com/Giygas/observability-stack
- **Documentation**: [observability/docs/](observability/docs/)
- **Contribution**: [observability/CONTRIBUTING.md](observability/CONTRIBUTING.md)

**Note**: For observability stack-specific questions (configuration, troubleshooting, alerts, dashboards), consult the submodule documentation mentioned above.

---

## Application Metrics

medicaments-api exposes Prometheus metrics on port 9090 (internal to Docker network).

### HTTP Metrics

Via `metrics/metrics.go`:

#### `http_request_total`

- **Type**: Counter
- **Labels**: `method`, `path`, `status`
- **Description**: Total HTTP requests
- **Example**: `http_request_total{method="GET",path="/v1/medicaments",status="200"}`

#### `http_request_duration_seconds`

- **Type**: Histogram
- **Labels**: `method`, `path`
- **Description**: Request latency histogram
- **Buckets**: .001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5
- **Example**: `http_request_duration_seconds_sum{method="GET",path="/v1/medicaments"}`

#### `http_request_in_flight`

- **Type**: Gauge
- **Description**: Currently in-flight requests
- **Example**: `http_request_in_flight`

### Metrics Visualization

Metrics are visualized in Grafana via pre-configured dashboards provided by the submodule.

**Metrics Access:**

- **Grafana**: http://localhost:3000 → Dashboards → medicaments-api Health
- **Prometheus**: http://localhost:9090 → direct query of metrics
- **Alloy**: http://localhost:12345/metrics (collector metrics)

---

## Log Format

medicaments-api generates structured JSON logs with the following fields:

### Expected Format

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

### Required Fields

| Field      | Type   | Description                       | Example                         |
| ---------- | ------ | --------------------------------- | ------------------------------- |
| `time`     | string | ISO 8601 timestamp                | `"2025-02-08T12:00:00Z"`      |
| `level`    | string | Log level (DEBUG/INFO/WARN/ERROR) | `"info"`                        |
| `path`     | string | HTTP request path                  | `"/v1/medicaments?search=paracetamol"` |
| `msg`      | string | Log message                       | `"Request completed"`            |
| `status`   | number | HTTP status code                  | `200`                           |
| `duration_ms` | number | Request duration in ms             | `15`                            |

### Alloy Parsing

The Alloy configuration uses `stage.json` to parse structured JSON logs and extract labels:

- **Extracted labels**: `level`, `path`, `status`, `duration_ms`
- **Filtering**: `DEBUG` level logs are removed before storing in Loki
- **Timestamp**: Using the `time` field with RFC3339Nano format

### Modifying Format

If you modify the log format in your application, update `configs/alloy/config.alloy`:

```alloy
// For plain text logs, replace stage.json with:
loki.process "process_logs" {
  stage.regex {
    expression: "^(?P<timestamp>\\S+) (?P<level>\\S+) (?P<message>.*)$"
  }
  // ... rest of configuration
}
```

---

## Access Points

### Application Services

| Service         | Host Port | URL                                     | Description                             |
| --------------- | ---------- | --------------------------------------- | ------------------------------------- |
| medicaments-api | 8030       | http://localhost:8030                     | Main API                        |
| grafana-alloy   | 12345      | http://localhost:12345/metrics           | Alloy collector metrics         |

### Observability Services (Submodule)

| Service    | Host Port | URL                                     | Description                             |
| ---------- | ---------- | --------------------------------------- | ------------------------------------- |
| loki       | Internal    | N/A                                     | Logs (access via Grafana only)      |
| prometheus | 9090       | http://localhost:9090                     | Prometheus UI                   |
| grafana    | 3000       | http://localhost:3000                     | Grafana UI with dashboards |

### Credentials

**Grafana:**

- Username: `admin` (configurable via `GRAFANA_ADMIN_USER` in submodule)
- Password: Stored in `observability/secrets/grafana_password.txt` (created via `make setup-secrets`)
- **Important**: Change password after first login

**Other Services:**

- No authentication required (local network only)

---

## Troubleshooting

### Submodule Issues

**Submodule not initialized:**

```bash
# Error: "fatal: not a git repository" or "network obs-network not found"
# Solution: Initialize submodule
make obs-init
make up
```

**Submodule out of sync:**

```bash
# Error: Observability services won't start
# Solution: Update submodule
make obs-update
make obs-down
make obs-up
```

### Startup Issues

**obs-network doesn't exist:**

```bash
# Error: "network obs-network not found"
# Solution: Start observability stack first
make obs-up
make up
```

**Observability services won't start:**

```bash
# Check observability stack logs
make obs-logs

# Check all containers status
docker compose ps

# Restart observability services
make obs-down
make obs-up
```

### Log Issues

**Logs not appearing in Grafana:**

```bash
# Check Alloy is reading logs
docker compose logs grafana-alloy | grep -i logs

# Check log files exist
docker compose exec grafana-alloy ls -la /var/log/app/

# Check Loki is receiving logs
make obs-logs | grep loki | grep -i received
```

### Metrics Issues

**Metrics not appearing in Grafana:**

```bash
# Check Alloy is scraping metrics
docker compose logs grafana-alloy | grep -i scrape

# Check metrics endpoint is accessible
docker compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics

# Check Prometheus is receiving metrics
make obs-logs | grep prometheus | grep -i received

# Test Prometheus query
curl 'http://localhost:9090/api/v1/query?query=http_request_total'
```

### Advanced Troubleshooting

For more complex issues regarding:

- Detailed configuration of Loki, Prometheus, Grafana
- Alerting rules and Prometheus
- Custom dashboards
- Tunnel configuration (Cloudflare, Tailscale, WireGuard)
- Performance and optimization

**Consult the submodule documentation:**

- **[observability/README.md](observability/README.md)**: Main guide
- **[observability/docs/local-setup.md](observability/docs/local-setup.md)**: Local mode
- **[observability/docs/remote-setup.md](observability/docs/remote-setup.md)**: Remote mode
- **[observability/docs/tunnels.md](observability/docs/tunnels.md)**: Tunnels

**Or report the issue on the submodule repository:**
https://github.com/Giygas/observability-stack/issues

---

## Resources

### medicaments-api Documentation

- **[DOCKER.en.md](DOCKER.en.md)**: Complete Docker guide for medicaments-api
- **[README.md](README.md)**: Project overview

### Submodule Documentation

- **[observability/README.md](observability/README.md)**: Main documentation
- **[observability/docs/local-setup.md](observability/docs/local-setup.md)**: Local configuration guide
- **[observability/docs/remote-setup.md](observability/docs/remote-setup.md)**: Remote configuration guide
- **[observability/docs/tunnels.md](observability/docs/tunnels.md)**: Tunnel guide

### External Links

- **Grafana**: https://grafana.com/docs/
- **Loki**: https://grafana.com/docs/loki/latest/
- **Prometheus**: https://prometheus.io/docs/
- **Grafana Alloy**: https://grafana.com/docs/alloy/
