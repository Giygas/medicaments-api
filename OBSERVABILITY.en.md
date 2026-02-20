# Observability Stack Guide

**Complete guide for Grafana, Loki, Prometheus, and Alloy observability stack**

---

[🇫🇷 Français](OBSERVABILITY.md) | **🇬🇧 English**

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Port Architecture](#port-architecture)
- [Services](#services)
  - [Grafana Alloy](#grafana-alloy)
  - [Loki](#loki)
  - [Prometheus](#prometheus)
  - [Grafana](#grafana)
- [Access Points](#access-points)
- [Log Format](#log-format)
- [Metrics Collected](#metrics-collected)
- [Default Credentials](#default-credentials)
- [Resource Usage](#resource-usage)
- [Configuration Files](#configuration-files)
- [Troubleshooting](#troubleshooting)
- [Cleanup](#cleanup)
- [Prometheus Alerting](#prometheus-alerting)
  - [Alert Rules](#alert-rules)
  - [Viewing Alerts](#viewing-alerts)
  - [Customizing Alerts](#customizing-alerts)
  - [Health Check Monitoring](#health-check-monitoring)
- [Advanced Topics](#advanced-topics)

---

## Overview

The staging setup includes a complete observability stack with Grafana, Loki, Prometheus, and Alloy for logs and metrics monitoring.

**Components:**

- **Grafana Alloy**: Collector agent that gathers logs and metrics
- **Loki**: Log aggregation and storage
- **Prometheus**: Metric storage and querying
- **Grafana**: Visualization and dashboarding

**Benefits:**

- Centralized log viewing and searching
- Real-time metrics monitoring
- Alerting on service health and performance
- Pre-configured dashboards for quick insights

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

## Architecture

```
medicaments-api (logs + metrics)
          ↓
grafana-alloy (collector)
          ↓         ↓
        loki    prometheus
          ↓         ↓
          grafana (visualization)
```

**Data Flow:**

1. **medicaments-api** generates logs (to `/app/logs/`) and metrics (to `/metrics` endpoint)
2. **Grafana Alloy** reads log files and scrapes metrics endpoints
3. **Loki** stores logs from Alloy
4. **Prometheus** stores metrics from Alloy via remote_write
5. **Grafana** queries both Loki and Prometheus for visualization

---

## Port Architecture

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

---

## Services

### Grafana Alloy

Collects logs and metrics from medicaments-api and system metrics.

- **Image**: `grafana/alloy:v1.4.0`
- **Configuration**: `configs/alloy/config.alloy`
- **Port**: 12345 (Alloy's own metrics)
- **Functions**:
  - Read logs from `./logs/` directory
  - Scrape application metrics from `medicaments-api:9090/metrics` (every 30s)
  - Collect system metrics via Unix exporter (every 60s)
  - Forward to local Loki and Prometheus
  - Filter out Go runtime metrics (keep only HTTP and system metrics)
- **Resource Usage**: ~150MB RAM

**Configuration Highlights:**

```alloy
// Logs collection
loki.source.file "read_logs" {
  targets    = [{__path__ = "/var/log/app/*.log"}]
  forward_to = [loki.write.local.receiver]
}

// Metrics scraping
prometheus.scrape "medicaments" {
  targets    = [{__address__ = "medicaments-api:9090"}]
  forward_to = [prometheus.remote_write.local.receiver]
  scrape_interval = "30s"
}

// System metrics
prometheus.exporter.unix "system" {
  collectors = ["cpu", "meminfo", "filesystem", "network"]
}
```

### Loki

Log aggregation and storage.

- **Image**: `grafana/loki:2.9.10`
- **Configuration**: `observability/loki/config.yaml`
- **Port**: 3100 (internal only - exposed to Docker network)
- **Storage**: Filesystem (chunks in `/loki/chunks`, rules in `/loki/rules`)
- **Retention**: 30 days (720 hours)
- **Data Volume**: `loki-data`
- **Resource Usage**: ~100MB RAM + ~100MB disk
- **Health Check**: Available via `/ready` endpoint

**Configuration Highlights:**

```yaml
limits_config:
  retention_period: 720h    # 30 days
  ingestion_rate_mb: 16    # 16MB/sec

schema_config:
  configs:
    - from: 2024-01-01
      store: tsdb
      object_store: filesystem
      schema: v13

storage_config:
  filesystem:
    directory: /loki/chunks
  ruler:
    storage:
      type: local
      local:
        directory: /loki/rules
```

**Important:** The ruler storage type must be explicitly set to `local` in Loki 2.9+ to avoid startup errors.

### Prometheus

Metric storage and querying.

- **Image**: `prom/prometheus:v2.48.0`
- **Configuration**: `observability/prometheus/prometheus.yml`
- **Port**: 9090 (host and container)
  - Host port 9090 provides external access to Prometheus UI
  - Container port 9090 is used for service-to-service communication
- **Retention**: 30 days (720 hours) - configured via CLI flag `--storage.tsdb.retention.time` in `observability/docker-compose.yml`
- **Data Volume**: `prometheus-data`
- **Resource Usage**: ~150MB RAM + ~200MB disk
- **Scraping**: Receives metrics via `remote_write` from Grafana Alloy (no `scrape_configs` needed)

**Configuration Highlights:**

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'medicaments-staging'

# Remote write from Alloy (receives via CLI flag)
remote_write:
  - url: http://localhost:9090/api/v1/write

# Alert rules
rule_files:
  - '/etc/prometheus/rules/*.yml'
```

**Note:** Data retention is configured via the CLI flag `--storage.tsdb.retention.time` in `observability/docker-compose.yml`, not in the `prometheus.yml` configuration file. Prometheus does not support retention configuration in its config file - this is a Prometheus design decision.

### Grafana

Visualization for logs and metrics.

- **Image**: `grafana/grafana:10.2.4`
- **Port**: 3000
- **Default Credentials**:
  - Username: `admin` (configurable via `GRAFANA_ADMIN_USER`)
  - Password: Stored in `secrets/grafana_password.txt` (created via `make setup-secrets`)
- **Important**: Change password after first login (Configuration → Users → Change Password)
- **Data Volume**: `grafana-data`
- **Resource Usage**: ~200MB RAM + ~50MB disk
- **Auto-Provisioning**: Datasources configured automatically

**Auto-Provisioning:**

- **Datasources**: Automatically configured from `observability/grafana/provisioning/datasources/`
  - Loki: `observability/grafana/provisioning/datasources/loki.yml`
  - Prometheus: `observability/grafana/provisioning/datasources/prometheus.yml`
- **Dashboards**: Automatically imported from `observability/grafana/provisioning/dashboards/`

---

## Access Points

```bash
# Grafana UI (visualization)
open http://localhost:3000

# Prometheus UI (metrics browsing)
open http://localhost:9090

# medicaments-api metrics (application metrics)
# Via Alloy (from Docker network only)
docker compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics

# Alloy metrics (collector status)
curl http://localhost:12345/metrics
```

**Note**: Loki and medicaments-api metrics are only exposed internally to the Docker network.
They are scraped by Alloy and not directly accessible from the host machine for security.

---

## Log Format

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

**Alloy Log Parsing:**

The Alloy configuration uses `stage.json` to parse structured JSON logs:

```alloy
loki.source.file "read_logs" {
  targets    = [{__path__ = "/var/log/app/*.log"}]
  forward_to = [loki.process.process_logs.receiver]
}

loki.process "process_logs" {
  stage.json {}
  stage.labels {
    values = {
      level   = "level",
      path    = "path",
      status  = "status"
    }
  }
  forward_to = [loki.write.local.receiver]
}
```

If logs are plain text, update `configs/alloy/config.alloy` to remove `stage.json` block and use regex parsing.

---

## Metrics Collected

### From Application (`/metrics` endpoint)

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
- **Description**: Current in-flight requests
- **Example**: `http_request_in_flight`

### From System Metrics (via `prometheus.exporter.unix`)

Alloy collects the following system metrics:

- **CPU metrics**: Process and system CPU usage
  - `process_cpu_seconds_total`
  - `node_cpu_seconds_total`

- **Memory metrics**: Process and system memory usage
  - `process_resident_memory_bytes`
  - `process_virtual_memory_bytes`
  - `node_memory_MemAvailable_bytes`
  - `node_memory_MemTotal_bytes`

- **File descriptors**: Open file descriptors
  - `process_open_fds`
  - `process_max_fds`

- **Network metrics**: Network I/O statistics
  - `node_network_receive_bytes_total`
  - `node_network_transmit_bytes_total`

- **Disk metrics**: Disk I/O and filesystem statistics
  - `node_filesystem_size_bytes`
  - `node_filesystem_avail_bytes`
  - `node_filesystem_read_bytes_total`

**Note:** Alloy configuration filters out Go runtime metrics from both application and system scrapers, keeping only HTTP and relevant system metrics.

---

## Default Credentials

**Grafana:**

- Username: `admin` (configurable via `GRAFANA_ADMIN_USER`)
- Password: Stored in `secrets/grafana_password.txt` (created via `make setup-secrets`)
- **Important**: Change password after first login (Configuration → Users → Change Password)

**Other Services:**

- No authentication required (local network only)

---

## Resource Usage

| Service         | RAM        | Disk          | Retention      |
| --------------- | ---------- | ------------- | -------------- |
| medicaments-api | ~50MB      | ~20MB         | N/A            |
| grafana-alloy   | ~150MB     | ~10MB         | N/A            |
| loki            | ~100MB     | ~100MB (data) | 30 days        |
| prometheus      | ~150MB     | ~200MB (data) | 30 days (720h) |
| grafana         | ~200MB     | ~50MB         | N/A            |
| **Total**       | **~650MB** | **~380MB**    | 30 days (both) |

---

## Configuration Files

| File                                                            | Purpose                                                                                               |
| --------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `configs/alloy/config.alloy`                              | Alloy configuration (logs + metrics collection, filters Go runtime metrics)                           |
| `observability/loki/config.yaml`                                | Loki configuration (log storage, filesystem ruler storage, 30-day retention, 16MB/sec ingestion rate) |
| `observability/prometheus/prometheus.yml`                       | Prometheus configuration (metric storage, 30-day retention, alert rules)                              |
| `observability/prometheus/alerts/medicaments-api.yml`           | Prometheus alert rules (service down, high error rate, high latency)                                  |
| `observability/grafana/provisioning/datasources/loki.yml`       | Auto-configure Loki datasource                                                                        |
| `observability/grafana/provisioning/datasources/prometheus.yml` | Auto-configure Prometheus datasource                                                                  |
| `observability/grafana/provisioning/dashboards/dashboard.yml`   | Auto-import Grafana dashboards                                                                        |
| `observability/grafana/dashboards/api-health.json`              | Pre-configured API health dashboard                                                                   |

---

## Troubleshooting

### Grafana can't connect to datasources

```bash
# Check containers are running
docker compose ps

# Verify network connectivity
# Note: Grafana connects to Prometheus on container port 9090
docker compose exec grafana wget -O- http://loki:3100/ready
docker compose exec grafana wget -O- http://prometheus:9090/-/ready

# Check datasource configuration
docker compose logs grafana | grep -i datasource

# Verify Prometheus datasource configuration
cat observability/grafana/provisioning/datasources/prometheus.yml
# Should show: url: http://prometheus:9090
```

### Logs not appearing in Grafana

```bash
# Check Alloy is reading logs
docker compose logs grafana-alloy | grep -i logs

# Verify log files exist
docker compose exec grafana-alloy ls -la /var/log/app/

# Check Loki is receiving logs
docker compose logs loki | grep -i received

# Query logs from within the Docker network
docker compose exec loki wget -O- 'http://localhost:3100/loki/api/v1/labels'
```

### Metrics not appearing in Grafana

```bash
# Check Alloy is scraping metrics
docker compose logs grafana-alloy | grep -i scrape

# Verify metrics endpoint is accessible
docker compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics

# Check Prometheus is receiving metrics
docker compose logs prometheus | grep -i received

# Test Prometheus query
curl 'http://localhost:9090/api/v1/query?query=http_request_total'
```

### Loki fails to start with storage error

```bash
# Check Loki logs for storage configuration errors
docker compose logs loki | grep -i "storage\|ruler"

# Common error: "field filesystem not found in type base.RuleStoreConfig"
# This occurs in Loki 2.9+ when ruler storage type is not explicitly specified

# Fix: Ensure loki/config.yaml ruler section has explicit local storage:
#   ruler:
#     storage:
#       type: local
#       local:
#         directory: /loki/rules

# Restart Loki after fixing config
docker compose restart loki
```

### High resource usage

```bash
# Check resource usage for all services
docker stats medicaments-api grafana-alloy loki prometheus grafana

# Check disk usage for volumes
docker system df -v

# Reduce retention if needed (edit loki/config.yaml or prometheus/prometheus.yml)
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
docker compose restart grafana

# Verify connectivity from Grafana container
docker compose exec grafana wget -O- http://prometheus:9090/-/ready
```

**Metrics not appearing in Grafana:**

```bash
# Check if Alloy is scraping metrics from medicaments-api
docker compose logs grafana-alloy | grep -i "medicaments-api:9090"

# Verify medicaments-api metrics endpoint is accessible (via Docker)
docker compose exec grafana-alloy wget -O- http://medicaments-api:9090/metrics

# Check if Prometheus is receiving metrics from Alloy
docker compose logs prometheus | grep -i "received from Alloy"

# Test Prometheus query for app metrics
curl 'http://localhost:9090/api/v1/query?query=http_request_total'
```

---

## Cleanup

```bash
# Stop observability services only
docker compose stop grafana-alloy loki prometheus grafana

# Remove observability services (keeps volumes)
docker compose rm -f grafana-alloy loki prometheus grafana

# Remove observability services and all data (DELETES EVERYTHING)
docker compose down -v

# Remove only observability volumes
docker volume rm medicaments-api_loki-data medicaments-api_prometheus-data medicaments-api_grafana-data
```

---

## Prometheus Alerting

The monitoring stack includes Prometheus alerting rules that automatically detect issues and display alerts in Grafana.

### Alert Rules

**Alert Rules Location:** `observability/prometheus/alerts/medicaments-api.yml`

#### Critical Alerts

| Alert              | Description             | Threshold          | Duration |
| ------------------ | ----------------------- | ------------------ | -------- |
| ServiceDown        | Service unreachable     | `up == 0`          | 5m       |
| High5xxErrorRate   | Too many server errors  | 5xx rate > 5%      | 5m       |
| HighTotalErrorRate | Too many errors overall | 4xx+5xx rate > 10% | 5m       |

#### Warning Alerts

| Alert            | Description            | Threshold               | Duration |
| ---------------- | ---------------------- | ----------------------- | -------- |
| HighLatencyP95   | Slow response times    | P95 latency > 200ms     | 10m      |
| HighRequestRate  | High traffic volume    | Request rate > 1000/sec | 5m       |
| Sustained4xxRate | High client error rate | 4xx rate > 5%           | 10m      |

### Viewing Alerts in Grafana

1. Navigate to `http://localhost:3000`
2. Go to **Alerting** → **Alert Rules** (in the left sidebar)
3. Filter by job `medicaments-api`
4. View active alerts, silenced alerts, and alert history

### Customizing Alerts

Edit `observability/prometheus/alerts/medicaments-api.yml` to adjust thresholds:

```yaml
# Example: Change P95 latency threshold
- alert: HighLatencyP95
  expr: |
    histogram_quantile(0.95,
      rate(http_request_duration_seconds_bucket{job="medicaments-api"}[10m])
    ) > 0.5  # Change from 0.2 (200ms) to 0.5 (500ms)
  for: 10m
  annotations:
    summary: "High P95 latency detected"
    description: "P95 latency is {{ $value }}s for job {{ $labels.job }}"
```

After editing, reload Prometheus configuration:

```bash
# Restart Prometheus to apply changes
docker compose restart prometheus

# Or use SIGHUP for hot reload (if configured)
docker exec prometheus kill -HUP 1
```

### Health Check Monitoring

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

## Advanced Topics

### Adding Custom Dashboards

1. Create a new JSON dashboard in `observability/grafana/dashboards/`
2. Update `observability/grafana/provisioning/dashboards/dashboard.yml`:

```yaml
apiVersion: 1

providers:
  - name: 'medicaments-api'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
      foldersFromFilesStructure: true
```

3. Restart Grafana:

```bash
docker compose restart grafana
```

### Custom Log Parsers

Modify `configs/alloy/config.alloy` to add custom log parsing:

```alloy
loki.process "custom_parser" {
  stage.regex {
    expression: "^(?P<timestamp>\\S+) (?P<level>\\S+) (?P<message>.*)$"
  }
  stage.labels {
    values = {
      timestamp = "timestamp",
      level     = "level"
    }
  }
  forward_to = [loki.write.local.receiver]
}
```

### Reducing Retention

To reduce disk usage, edit retention settings:

**Loki** (`observability/configs/loki/config.yaml`):

```yaml
limits_config:
  retention_period: 168h  # 7 days (was 720h)
```

**Prometheus** (edit CLI flag in `observability/docker-compose.yml`):

```yaml
prometheus:
  command:
    - "--config.file=/etc/prometheus/prometheus.yml"
    - "--storage.tsdb.path=/prometheus"
    - "--storage.tsdb.retention.time=${PROMETHEUS_RETENTION:-168h}"  # Change here
```

**Note:** Prometheus retention is configured via the CLI flag `--storage.tsdb.retention.time` in `docker-compose.yml`, not in the config file. Prometheus does not support retention configuration in its config file - this is a Prometheus design decision. Loki, on the other hand, does support retention configuration in its config file.

Then restart:

```bash
docker compose restart loki prometheus
```

### Exporting Metrics

To export metrics to external Prometheus:

```yaml
# In observability/prometheus/prometheus.yml
remote_write:
  - url: https://external-prometheus.example.com/api/v1/write
    basic_auth:
      username: ${EXTERNAL_PROMETHEUS_USER}
      password: ${EXTERNAL_PROMETHEUS_PASSWORD}
```

---

