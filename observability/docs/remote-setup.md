# Remote Mode Setup

This guide explains how to use the observability stack in remote mode, where apps connect from production servers via a tunnel.

## Architecture

```
Production Server (Linux)              Observability Stack (Windows PC/Server)
┌─────────────────────────────┐        ┌──────────────────────────────────┐
│                             │        │                                  │
│ your-app:8030        │        │ Loki:3100                        │
│              │              │        │ Prometheus:9090                  │
│              ▼              │        │ Grafana:3000                      │
│ Grafana Alloy               ├────────┤ (Exposed via tunnel)              │
│    - WAL Buffering          │  VPN  │                                  │
│    - Auth Headers          │  Mesh └──────────────────────────────────┘
└─────────────────────────────┘
```

## Prerequisites

- Observability stack running with tunnel (Cloudflare/Tailscale/WireGuard)
- Tunnel endpoints configured and accessible
- App repository with Grafana Alloy configured

## Step 1: Configure Observability Stack

Choose your tunnel type and start the stack:

```bash
# On your Windows PC/server
cd observability-stack
make setup
make up-cloudflare    # or up-tailscale, up-wireguard
```

## Step 2: Get Remote Endpoint URLs

### Cloudflare Tunnel

```
PROMETHEUS_URL=https://prometheus-obs.yourdomain.com/api/v1/write
LOKI_URL=https://loki-obs.yourdomain.com/loki/api/v1/push
GRAFANA_URL=https://grafana-obs.yourdomain.com
```

### Tailscale VPN

```bash
# Get Tailscale IP
docker exec tailscale-obs tailscale ip -4
# Output: 100.x.x.x

PROMETHEUS_URL=http://100.x.x.x:9090/api/v1/write
LOKI_URL=http://100.x.x.x:3100/loki/api/v1/push
GRAFANA_URL=http://100.x.x.x:3000
```

### WireGuard VPN

```
PROMETHEUS_URL=http://10.13.13.1:9090/api/v1/write
LOKI_URL=http://10.13.13.1:3100/loki/api/v1/push
GRAFANA_URL=http://10.13.13.1:3000
```

## Step 3: Configure App Repository

### Add Alloy Service to docker-compose.yml

```yaml
services:
  your-app:
    # ... app config ...
    networks:
      - obs-network

  grafana-alloy:
    image: grafana/alloy:v1.4.0
    container_name: grafana-alloy
    ports:
      - "12345:12345"
    volumes:
      - ./logs:/var/log/app:ro
      # Falls back to local config if ALLOY_CONFIG is not set
      - ./configs/alloy/${ALLOY_CONFIG:-config.alloy}:/etc/alloy/config.alloy:ro
      - alloy-wal:/var/lib/alloy/wal
    command:
      - "run"
      - "/etc/alloy/config.alloy"
      - "--server.http.listen-addr=0.0.0.0:12345"
      - "--storage.path=/var/lib/alloy/wal"
    restart: unless-stopped
    networks:
      - obs-network
    depends_on:
      your-app:
        condition: service_healthy

networks:
  obs-network:
    external: true
    name: obs-network

volumes:
  alloy-wal:
```

**Safety**: `${ALLOY_CONFIG:-config.alloy}` defaults to local mode. Remote mode requires explicit `ALLOY_CONFIG=config.remote.alloy` in `.env`.

### Create Remote Alloy Config

`configs/alloy/config.remote.alloy`:

```alloy
// Remote mode: obs stack on remote machine via tunnel
// WAL enabled for buffering during outages

prometheus.scrape "app" {
  targets = [{
    __address__ = "your-app:9090",
    job         = "your-app",
  }]
  scrape_interval = "15s"
  forward_to = [prometheus.remote_write.obs.receiver]
}

prometheus.remote_write "obs" {
  endpoint {
    url = env("PROMETHEUS_URL")

    // Cloudflare Access headers (remove if not using)
    headers = {
      "CF-Access-Client-Id"     = env("CF_ACCESS_CLIENT_ID"),
      "CF-Access-Client-Secret" = env("CF_ACCESS_CLIENT_SECRET"),
    }

    queue_config {
      capacity             = 10000
      max_samples_per_send = 2000
      batch_send_deadline  = "5s"
      min_backoff          = "30ms"
      max_backoff          = "5s"
    }
  }

  // WAL Buffering for outages
  wal {
    truncate_frequency = "2h"
    max_size_bytes     = 2684354560  # 2.5GB
  }
}

local.file_match "app_logs" {
  path_targets = [{
    __path__ = "/var/log/app/*.log",
    job       = "your-app",
  }]
}

loki.source.file "app_logs" {
  targets    = local.file_match.app_logs.targets
  forward_to = [loki.write.obs.receiver]
}

loki.write "obs" {
  endpoint {
    url = env("LOKI_URL")

    // Cloudflare Access headers (remove if not using)
    headers = {
      "CF-Access-Client-Id"     = env("CF_ACCESS_CLIENT_ID"),
      "CF-Access-Client-Secret" = env("CF_ACCESS_CLIENT_SECRET"),
    }
  }

  // WAL Buffering for outages
  wal {
    truncate_frequency = "2h"
    max_size_bytes     = 2684354560  # 2.5GB
  }
}
```

### Configure Environment Variables

`.env`:

```bash
# ── Alloy Mode ─────────────────────────────────────────────
# Explicitly set to remote mode for production
# Default is config.alloy (local mode)
ALLOY_CONFIG=config.remote.alloy

# Remote endpoint URLs (from Step 2)
PROMETHEUS_URL=https://prometheus-obs.yourdomain.com/api/v1/write
LOKI_URL=https://loki-obs.yourdomain.com/loki/api/v1/push

# Cloudflare Access credentials (remove if not using Cloudflare)
CF_ACCESS_CLIENT_ID=your_client_id
CF_ACCESS_CLIENT_SECRET=your_client_secret
```

**Important**: Set `ALLOY_CONFIG=config.remote.alloy` to enable remote mode. If not set, defaults to `config.alloy` (local mode).

## Step 4: Start App on Production

```bash
# On production server
cd your-app
docker compose up -d
```

## Step 5: Verify Connection

```bash
# Check Alloy logs
docker logs grafana-alloy -f

# Should see:
# "Remote write endpoint is healthy"
# "Loki write endpoint is healthy"

# Check metrics in Grafana
# Navigate to your Grafana URL
# → Dashboards → API Health
```

## WAL Buffering

Alloy uses Write-Ahead Log (WAL) to buffer data when the remote endpoint is unavailable:

### How It Works

1. **Normal Operation**: Alloy sends data directly to remote
2. **Remote Offline**: Alloy writes to WAL disk buffer
3. **Buffer Full (~2.5GB)**: New data dropped (oldest preserved)
4. **Remote Online**: Alloy replays buffered data automatically

### Configuration

```alloy
prometheus.remote_write "obs" {
  endpoint { url = env("PROMETHEUS_URL") }
  wal {
    truncate_frequency = "2h"           # Truncate WAL every 2 hours
    max_size_bytes     = 2684354560     # 2.5GB max buffer
  }
}
```

### Estimated Buffer Duration

Based on typical usage:

- **Metrics**: ~500MB/day → ~5 days
- **Logs**: ~300MB/day → ~8 days
- **Total**: ~5-10 days per outage

### Check WAL Status

```bash
# Inside Alloy container
docker exec grafana-alloy du -sh /var/lib/alloy/wal

# Expected output: 2.5G (when full)
```

## Multiple Apps

To connect multiple apps to the same observability stack:

1. **Use unique job labels** in each app's Alloy config:

   ```alloy
   prometheus.scrape "app" {
     targets = [{
       __address__ = "your-app:9090",
       job         = "your-app-name",  // Unique identifier
     }]
   }
   ```

2. **Filter by job** in Grafana queries:

   ```promql
   rate(http_requests_total{job="your-app-name"}[5m])
   ```

3. **Use separate log directories**:
   ```yaml
   volumes:
     - ./logs/your-app:/var/log/app:ro
   ```

## Troubleshooting

### Connection Refused

```bash
# Check remote endpoint is reachable
docker exec grafana-alloy curl -v $PROMETHEUS_URL

# Check firewall rules
sudo ufw status

# Test tunnel connectivity
ping your-tunnel-ip
```

### WAL Not Truncating

```bash
# Check WAL size
docker exec grafana-alloy du -sh /var/lib/alloy/wal

# Check Alloy logs for truncation errors
docker logs grafana-alloy | grep truncat
```

### Metrics Not Appearing in Grafana

```bash
# Check Alloy is scraping
curl http://localhost:12345/agent/api/v1/targets

# Check remote write queue
docker logs grafana-alloy | grep "remote_write"

# Verify Prometheus is receiving
curl http://your-prometheus:9090/api/v1/query?query=up
```

### Logs Not Appearing in Grafana

```bash
# Check log file permissions
ls -la /var/log/app/

# Check Alloy log source
docker logs grafana-alloy | grep "loki.source"

# Verify Loki is receiving
curl http://your-loki:3100/ready
```

## Security

### Cloudflare Access (Recommended)

```bash
# Enable Cloudflare Access in tunnel dashboard
# Create policy for specific users/groups
# Generate service token for auth

CF_ACCESS_CLIENT_ID=your_id
CF_ACCESS_CLIENT_SECRET=your_secret
```

### Tailscale ACLs

```bash
# Create ACL in Tailscale admin console
# Restrict access to specific devices
# Limit which IPs can connect to obs stack
```

### Network Isolation

```yaml
# App docker-compose.yml
networks:
  obs-network:
    external: true # Join existing obs-network
    # Don't create internal network to avoid conflicts
```

## Monitoring

### Alloy Metrics

Prometheus endpoint on Alloy: `http://localhost:12345/metrics`

Key metrics:

- `alloy_wal_size_bytes` - Current WAL size
- `alloy_wal_replayed_bytes` - Bytes replayed after reconnect
- `alloy_remote_write_errors_total` - Remote write errors

### Grafana Dashboards

1. **API Health** - Request rate, error rate, latency
2. **Infrastructure** - CPU, memory, disk
3. **Alloy Health** - WAL size, remote write queue

## Next Steps

- [Tunnel Setup Guide](tunnels.md)
- [Local Mode Setup](local-setup.md)
