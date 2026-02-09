# Observability Stack Setup Complete

## What Was Created

A complete local observability stack has been added to your Docker staging environment:

### Configuration Files

1. **`observability/alloy/config.alloy`** - Grafana Alloy configuration
   - Reads logs from `/var/log/app/*.log`
   - Scrapes metrics from `medicaments-api:9090/metrics`
   - Forwards to local Loki and Prometheus

2. **`observability/loki/config.yaml`** - Loki configuration
   - Stores logs for 30 days
   - Exposes API on port 3100

3. **`observability/prometheus/prometheus.yml`** - Prometheus configuration
   - Scrapes metrics from Alloy
   - Stores metrics for 30 days
   - Exposes UI on port 9091 (host)

4. **`observability/grafana/provisioning/datasources/loki.yml`** - Auto-configures Loki datasource
5. **`observability/grafana/provisioning/datasources/prometheus.yml`** - Auto-configures Prometheus datasource
6. **`observability/grafana/provisioning/dashboards/dashboard.yml`** - Auto-imports Grafana dashboards

### Updated Files

1. **`docker-compose.yml`** - Added 4 services:
   - `grafana-alloy`
   - `loki`
   - `prometheus`
   - `grafana`

2. **`.dockerignore`** - Added observability config directory

3. **`.env.staging`** - Removed Grafana Cloud credentials (no longer needed)

4. **`DOCKER.md`** - Added comprehensive observability section

5. **New `observability/` directory** - Groups all Grafana stack configurations

### New Docker Services

| Service | Port | RAM | Disk | Purpose |
|----------|------|------|---------|
| medicaments-api | 8030, 9090 | ~50MB | ~20MB | API + Metrics |
| grafana-alloy | 12345 | ~150MB | ~10MB | Log/Metric Collector |
| loki | 3100 | ~100MB | ~100MB | Log Storage |
| prometheus | 9091 | ~150MB | ~200MB | Metric Storage |
| grafana | 3000 | ~200MB | ~50MB | Visualization |
| **Total** | - | **~650MB** | **~380MB** |

### New Directory Structure

All observability configurations now organized in `observability/` directory:
- `observability/alloy/` - Alloy config
- `observability/loki/` - Loki config
- `observability/prometheus/` - Prometheus config (30-day retention)
- `observability/grafana/` - Grafana provisioning

## Port Mapping

- **8030** - medicaments-api (HTTP API)
- **9090** - medicaments-api (Prometheus metrics)
- **12345** - grafana-alloy (self-metrics)
- **3100** - loki (Loki API)
- **9091** - prometheus (Prometheus UI)
- **3000** - grafana (Grafana UI)

## Quick Start

### 1. Start the observability stack

```bash
# Start all services (including observability stack)
docker-compose up -d

# Or use the interactive script
./docker-staging.sh
```

### 2. Verify all services are running

```bash
# Check container status
docker-compose ps

# Expected output:
# NAME                    STATUS              PORTS
# medicaments-api-staging  Up                  0.0.0.0:8030->8000/tcp, 0.0.0.0:9090->9090/tcp
# grafana-alloy          Up                  0.0.0.0:12345->12345/tcp
# loki                   Up                  0.0.0.0:3100->3100/tcp
# prometheus             Up                  0.0.0.0:9091->9090/tcp
# grafana                Up                  0.0.0.0.0:3000->3000/tcp
```

### 3. Access Grafana

```bash
# Open Grafana UI
open http://localhost:3000

# Or use your browser:
# http://localhost:3000
```

### 4. Login to Grafana

- **Username**: `admin`
- **Password**: `admin`
- **Important**: Change password after first login!

### 5. Verify datasources

1. Go to **Configuration** → **Data Sources**
2. You should see:
   - **Prometheus** (default) - Green checkmark
   - **Loki** - Green checkmark

Both should show green checkmarks indicating successful connection.

## Next Steps

### 1. Explore Your Logs

1. In Grafana, go to **Explore**
2. Select **Loki** datasource
3. Query: `{app="medicaments_api"}`
4. Set time range: **Last 1 hour**
5. Click **Run query**

### 2. Explore Your Metrics

1. In Grafana, go to **Explore**
2. Select **Prometheus** datasource
3. Try queries:
   - `http_request_total` - Total requests
   - `rate(http_request_total[1m])` - Request rate
   - `http_request_duration_seconds_sum` - Total duration
   - `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[1m]))` - 95th percentile latency
4. Click **Run query**

### 3. Create Dashboards

1. Go to **+** → **Dashboard**
2. Click **Add visualization**
3. Select your datasource and query
4. Configure panel settings
5. Click **Apply**
6. Save dashboard

### Example Dashboard Queries

#### HTTP Requests Rate

```
# Prometheus
rate(http_request_total[1m])

# Break down by path
sum by (path) (rate(http_request_total[1m]))

# Break down by status
sum by (status) (rate(http_request_total[1m]))
```

#### Request Duration (P95 Latency)

```
# Prometheus
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[1m]))
```

#### Error Rate

```
# Prometheus
rate(http_request_total{status=~"5.."}[1m])
```

## Troubleshooting

### Grafana won't start

```bash
# Check logs
docker-compose logs grafana

# Verify port 3000 is not in use
lsof -i :3000

# Restart Grafana
docker-compose restart grafana
```

### Datasources showing errors

```bash
# Verify services are reachable
docker-compose exec grafana wget -O- http://loki:3100/ready
docker-compose exec grafana wget -O- http://prometheus:9090/-/healthy

# Check Loki logs
docker-compose logs loki

# Check Prometheus logs
docker-compose logs prometheus
```

### No logs appearing

```bash
# Check Alloy logs
docker-compose logs grafana-alloy | grep -i logs

# Verify log files exist
ls -la logs/

# Check logs inside container
docker-compose exec grafana-alloy ls -la /var/log/app/

# Test Loki query
curl -G 'http://localhost:3100/loki/api/v1/query_range' \
  --data-urlencode 'query={app="medicaments_api"}' \
  --data-urlencode 'start=1709452800000000000' \
  --data-urlencode 'end=17094564000000000'
```

### No metrics appearing

```bash
# Check Alloy is scraping
docker-compose logs grafana-alloy | grep -i scrape

# Verify metrics endpoint
curl http://localhost:9090/metrics

# Check Prometheus logs
docker-compose logs prometheus | grep -i received

# Test Prometheus query
curl 'http://localhost:9091/api/v1/query?query=http_request_total'
```

## Maintenance

### Clean up old data

```bash
# Stop observability services
docker-compose stop grafana-alloy loki prometheus grafana

# Remove only observability volumes (keeps app data)
docker volume rm medicaments-api_loki-data medicaments-api_prometheus-data medicaments-api_grafana-data

# Start services again
docker-compose up -d
```

### Reduce retention periods

**Loki** (30 days): Edit `loki/config.yaml`:
```yaml
table_manager:
  retention_period: 720h  # Change to desired hours
```

**Prometheus** (15 days): Edit `prometheus/prometheus.yml`:
```yaml
# Add to global section
global:
  retention:
    time: 720h  # Change to desired hours
```

## Performance Tips

### 1. Reduce memory usage

If experiencing memory issues, reduce retention periods or resource limits in `docker-compose.yml`:

```yaml
services:
  loki:
    deploy:
      resources:
        limits:
          memory: 128M  # Reduce from default

  prometheus:
    deploy:
      resources:
        limits:
          memory: 256M  # Reduce from default
```

### 2. Increase scraping intervals

Reduce scraping frequency to decrease resource usage in `alloy/config.alloy`:

```hcl
prometheus.scrape "app_metrics" {
  scrape_interval = "60s"  # Increase from 30s
  # ...
}
```

## Clean Up Old Documentation Files

```bash
# Remove old redundant Docker documentation files
rm DOCKER_STAGING.md
rm DOCKER_SETUP_SUMMARY.md
rm DOCKER_QUICK_REFERENCE.md

# Verify only DOCKER.md remains
ls -lh DOCKER*.md
```

## Summary

✅ **Complete observability stack** - Logs + Metrics + Visualization
✅ **Local-only** - No external dependencies or Grafana Cloud
✅ **Auto-provisioned** - Datasources configured automatically
✅ **~650MB RAM** - Reasonable resource usage
✅ **~380MB disk** - With 30-day log retention
✅ **All services** - Running via Docker Compose
✅ **Ready to use** - Access Grafana at http://localhost:3000

## Documentation

- **Full Docker Guide**: `DOCKER.md` (includes observability section)
- **Quick Reference**: `DOCKER.md` → Essential Commands
- **Troubleshooting**: `DOCKER.md` → Observability Troubleshooting

## Support

- **Grafana Docs**: https://grafana.com/docs/
- **Prometheus Docs**: https://prometheus.io/docs/
- **Loki Docs**: https://grafana.com/docs/loki/latest/
- **Alloy Docs**: https://grafana.com/docs/alloy/latest/

---

**Last updated: 2025-02-09**
