# Local Mode Setup

This guide explains how to use the observability stack as a submodule in your app repository for local development or staging.

## Architecture

```
App Repository (your-app/)
├── docker-compose.yml         # app + alloy + obs-stack
├── .env                       # ALLOY_CONFIG setting (default: config.alloy)
├── observability/              # git submodule
│   ├── docker-compose.yml     # loki + prometheus + grafana
│   └── configs/
└── configs/
    └── alloy/
        ├── config.alloy         # local mode config (default)
        └── config.remote.alloy # remote mode config (explicit)
```

Docker Network: obs-network
┌─────────────────────────────────────────────────────┐
│ your-app ──┐ │
│ ├─→ Alloy ──→ Loki │
│ prometheus └─→ ──→ Prometheus │
│ loki ──→ Grafana │
│ grafana │
└─────────────────────────────────────────────────────┘

````

## Prerequisites

- Docker installed
- Git installed
- App repository

## Step 1: Add Observability as Submodule

```bash
# In your app repository
cd your-app-repo

# Add submodule
git submodule add https://github.com/you/observability-stack.git observability

# Initialize submodule
git submodule update --init --recursive

# Verify submodule
ls observability/
````

### .gitmodules

This file is created automatically:

```ini
[submodule "observability"]
    path = observability
    url = https://github.com/you/observability-stack.git
    branch = main
```

### .env

Create `.env` in your app repository root:

```bash
# ── Alloy Mode ─────────────────────────────────────────────
# Local mode (default): config.alloy
# Remote mode: config.remote.alloy (requires ALLOY_CONFIG setting)
# Default is always local — remote requires explicit flag
ALLOY_CONFIG=config.alloy
```

## Step 2: Setup Observability Stack

```bash
cd observability
make setup
cd ..

# Creates:
# - observability/secrets/grafana_password.txt
# - observability/.env
```

## Step 3: Update App docker-compose.yml

```yaml
services:
  your-app:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: your-app
    ports:
      - "8030:8000"
    expose:
      - "9090" # Metrics endpoint
    volumes:
      - logs_data:/app/logs # Log volume
    restart: unless-stopped
    networks:
      - obs-network
    healthcheck:
      test:
        [
          "CMD",
          "wget",
          "--quiet",
          "--tries=1",
          "--spider",
          "http://localhost:8000/health",
        ]
      interval: 30s
      timeout: 10s
      retries: 3

  grafana-alloy:
    image: grafana/alloy:v1.4.0
    container_name: grafana-alloy
    ports:
      - "12345:12345"
    volumes:
      - logs_data:/var/log/app:ro
      # Falls back to local config if ALLOY_CONFIG is not set
      - ./configs/alloy/${ALLOY_CONFIG:-config.alloy}:/etc/alloy/config.alloy:ro
    command:
      - "run"
      - "/etc/alloy/config.alloy"
      - "--server.http.listen-addr=0.0.0.0:12345"
    restart: unless-stopped
    networks:
      - obs-network
    depends_on:
      your-app:
        condition: service_healthy

networks:
  obs-network:
    external: true # Must exist before starting app
    name: obs-network

volumes:
  logs_data:
```

**Important**:

- `obs-network` must be `external: true` because it's created by the observability submodule.
- `${ALLOY_CONFIG:-config.alloy}` defaults to local mode if the env var is not set.
- **Safety**: Default is always local — remote requires explicit `ALLOY_CONFIG=config.remote.alloy` in `.env`.

## Step 4: Create Local Alloy Config

`configs/alloy/config.alloy`:

```alloy
// Local mode: obs stack on same Docker network
// No auth headers, no WAL needed, uses container DNS

// Scrape app metrics
prometheus.scrape "app" {
  targets = [{
    __address__ = "your-app:9090",
    job         = "your-app",
  }]
  scrape_interval = "15s"
  forward_to = [prometheus.remote_write.obs.receiver]
}

// Send to local Prometheus
prometheus.remote_write "obs" {
  endpoint {
    url = "http://prometheus:9090/api/v1/write"
  }
}

// Read app logs
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

// Send to local Loki
loki.write "obs" {
  endpoint {
    url = "http://loki:3100/loki/api/v1/push"
  }
}
```

## Step 5: Update Makefile (Optional)

```makefile
OBS_DIR = observability

.PHONY: up down logs status obs-up obs-down obs-update

# Full stack startup
up: obs-up
	docker compose up -d

# Full stack shutdown
down:
	docker compose down
	$(MAKE) obs-down

# Observability stack
obs-up:
	$(MAKE) -C $(OBS_DIR) up

obs-down:
	$(MAKE) -C $(OBS_DIR) down

obs-logs:
	$(MAKE) -C $(OBS_DIR) logs

obs-status:
	$(MAKE) -C $(OBS_DIR) status

# App logs
logs:
	docker compose logs -f

# Status
status:
	docker compose ps
	$(MAKE) -C $(OBS_DIR) status
```

## Step 6: Start Everything

```bash
# Option 1: Using Makefile
make up

# Option 2: Manual commands
cd observability && make up && cd ..
docker compose up -d
```

## Step 7: Verify

```bash
# Check all services
docker compose ps

# Expected output:
# NAME                  STATUS         PORTS
# your-app              Up (healthy)   0.0.0.0:8030->8000/tcp
# grafana-alloy         Up             0.0.0.0:12345->12345/tcp
# grafana               Up             0.0.0.0:3000->3000/tcp
# loki                  Up (healthy)
# prometheus            Up (healthy)

# Check logs
docker logs grafana-alloy

# Access Grafana
# URL: http://localhost:3000
# User: admin
# Password: admin (check observability/secrets/grafana_password.txt)
```

## Network Diagram

All containers share the `obs-network`:

```
┌─────────────────────────────────────────────────────┐
│              obs-network (bridge)                   │
│                                                     │
│  ┌──────────────┐      ┌──────────────┐            │
│  │ your-app     │─────▶│ Alloy        │            │
│  │ :8030, :9090│      │ :12345       │            │
│  └──────────────┘      └──────┬───────┘            │
│                                │                    │
│                          ┌─────▼─────┬─────────────┐ │
│                          │           │             │ │
│                    ┌─────▼────┐ ┌───▼──────┐ ┌─────▼──┐│
│                    │ Prometheus│ │ Loki    │ │ Grafana││
│                    │ :9090    │ │ :3100   │ │ :3000  ││
│                    └──────────┘ └─────────┘ └────────┘│
└─────────────────────────────────────────────────────┘
```

## Container Communication

- **Alloy → App**: `http://your-app:9090` (scrape metrics)
- **Alloy → Logs**: `/var/log/app/*.log` (read from volume)
- **Alloy → Prometheus**: `http://prometheus:9090/api/v1/write`
- **Alloy → Loki**: `http://loki:3100/loki/api/v1/push`
- **Grafana → Prometheus**: `http://prometheus:9090`
- **Grafana → Loki**: `http://loki:3100`

## Troubleshooting

### Network Already Exists

```bash
# If you get "network obs-network already exists" error
docker network rm obs-network

# Then restart
make up
```

### Containers Can't Reach Each Other

```bash
# Check network connectivity
docker exec your-app ping prometheus
docker exec alloy ping loki

# Check which network containers are on
docker network inspect obs-network

# Should show all 5 containers in the network
```

### Grafana Can't Connect to Datasources

```bash
# Check datasources are configured
curl http://localhost:3000/api/datasources

# Check Prometheus is reachable
docker exec grafana curl http://prometheus:9090/-/healthy

# Check Loki is reachable
docker exec grafana curl http://loki:3100/ready
```

### Alloy Not Scraping Metrics

```bash
# Check Alloy targets
curl http://localhost:12345/agent/api/v1/targets

# Should show:
# {
#   "activeTargets": [
#     {
#       "labels": { "job": "your-app" },
#       "health": "up",
#       "scrapeUrl": "http://your-app:9090"
#     }
#   ]
# }
```

### Logs Not Appearing in Grafana

```bash
# Check log file exists
docker exec alloy ls -la /var/log/app/

# Check Alloy is reading logs
docker logs grafana-alloy | grep "local.file_match"

# Check Loki is receiving
docker logs loki
```

## Updating the Submodule

```bash
# Get latest changes from observability repo
git submodule update --remote observability

# If observability has breaking changes, test locally:
cd observability
git checkout main
cd ..
```

## Removing the Submodule

```bash
# Stop everything
make down

# Remove submodule
git deinit -f observability
rm -rf .git/modules/observability
git rm -f observability

# Remove Docker network
docker network rm obs-network
```

## Advantages vs Remote Mode

| Feature              | Local (Submodule)   | Remote (Tunnel)     |
| -------------------- | ------------------- | ------------------- |
| Setup complexity     | Simple              | Medium              |
| Resource usage       | Higher (all-in-one) | Lower (distributed) |
| Outage resilience    | None                | WAL buffering       |
| Network requirements | None                | Tunnel needed       |
| Production ready     | ❌                  | ✅                  |
| Development          | ✅                  | ⚠️                  |

## Use Cases

### Use Local Mode When:

- Development environment
- Staging environment
- Testing new dashboards
- Learning observability stack
- No need for outage protection

### Use Remote Mode When:

- Production deployment
- Multiple apps sharing same stack
- Windows PC as observability server
- Need WAL buffering for outages
- Want centralized observability

## Next Steps

- [Remote Mode Setup](remote-setup.md)
- [Tunnel Setup Guide](tunnels.md)
