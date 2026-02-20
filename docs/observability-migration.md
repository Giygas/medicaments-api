# Observability Migration Guide

This guide documents the migration from bundled observability configs to the modular observability-stack submodule.

## What Changed

### Before Migration
```
docker-compose.yml (5 services)
├── medicaments-api
├── grafana-alloy
├── loki              # bundled config
├── prometheus         # bundled config
└── grafana            # bundled config

observability/ (local configs)
├── alloy/config.alloy
├── loki/config.yaml
├── prometheus/prometheus.yml
└── grafana/provisioning/ + dashboards/
```

### After Migration
```
docker-compose.yml (2 services)
├── medicaments-api
└── grafana-alloy

observability/ (git submodule)
├── docker-compose.yml  # loki + prometheus + grafana
└── configs/           # from observability-stack repo

configs/alloy/ (in app repo)
├── config.alloy         # local mode (default)
└── config.remote.alloy  # remote mode (explicit)
```

## Key Changes

### 1. Network: `medicaments-network` → `obs-network`

The Docker network is now **external** and created by the observability submodule:

```yaml
networks:
  obs-network:
    external: true
    name: obs-network
```

### 2. Alloy Config: Safety Pattern

Alloy now uses a fallback to default to local mode:

```yaml
volumes:
  # Falls back to local config if ALLOY_CONFIG is not set
  - ./configs/alloy/${ALLOY_CONFIG:-config.alloy}:/etc/alloy/config.alloy:ro
```

### 3. New Environment Variables

`.env.docker` now includes:

```bash
# ── Observability ─────────────────────────────────────────────
# Local mode (default): config.alloy
# Remote mode: config.remote.alloy
# Default is always local — remote requires explicit flag
ALLOY_CONFIG=config.alloy

# Remote endpoint URLs (only needed when ALLOY_CONFIG=config.remote.alloy)
PROMETHEUS_URL=
LOKI_URL=
```

### 4. New Makefile Commands

| Command | Description |
|---------|-------------|
| `make obs-init` | Initialize observability submodule (first time) |
| `make obs-up` | Start observability stack (Loki, Prometheus, Grafana) |
| `make obs-down` | Stop observability stack |
| `make obs-update` | Update observability submodule to latest |
| `make obs-logs` | View observability stack logs |
| `make obs-status` | Show observability stack status |

### 5. Modified Existing Commands

| Command | Change |
|---------|---------|
| `make up` | Now runs `obs-up` first, then starts app |
| `make down` | Now stops app, then runs `obs-down` |

## Usage

### First Time Setup

```bash
# 1. Initialize the submodule
make obs-init

# 2. Start everything
make up

# This will:
# - Clone observability-stack submodule
# - Setup observability secrets/configs
# - Start obs stack (Loki, Prometheus, Grafana)
# - Start medicaments-api with Alloy
```

### Daily Use

```bash
# Start everything (app + obs stack)
make up

# Stop everything
make down

# Restart everything
make restart

# View app logs
make logs

# View observability logs
make obs-logs

# Check status
make ps
make obs-status
```

### Update Submodule

```bash
# Get latest observability-stack updates
make obs-update
```

## Switching Modes

### Local Mode (Default)

Works out of the box - just run `make up`:

```bash
# .env.docker
ALLOY_CONFIG=config.alloy  # or leave unset
```

Alloy connects to `http://loki:3100` and `http://prometheus:9090` via container DNS.

### Remote Mode

Edit `.env.docker` and set:

```bash
# .env.docker
ALLOY_CONFIG=config.remote.alloy

# Cloudflare tunnel
PROMETHEUS_URL=https://prometheus-obs.yourdomain.com/api/v1/write
LOKI_URL=https://loki-obs.yourdomain.com/loki/api/v1/push

# Tailscale VPN
# PROMETHEUS_URL=http://100.x.x.x:9090/api/v1/write
# LOKI_URL=http://100.x.x.x:3100/loki/api/v1/push

# Optional: Cloudflare Access
CF_ACCESS_CLIENT_ID=your_id
CF_ACCESS_CLIENT_SECRET=your_secret
```

Then restart:

```bash
make down
make up
```

Alloy will connect to remote endpoints with WAL buffering enabled (2.5GB buffer, ~5-10 days protection).

## Benefits

| Benefit | Description |
|----------|-------------|
| **Reusability** | Observability stack can be shared across multiple projects |
| **Maintainability** | Single source of truth for observability configs |
| **Flexibility** | Easy switch between local/remote modes |
| **Safety** | Default to local mode prevents accidental remote connections |
| **Outage Protection** | WAL buffering in remote mode protects data during outages |
| **Isolation** | Separates app code from observability infrastructure |

## Troubleshooting

### Network Issues

```bash
# If you get "network obs-network not found"
make obs-up
docker compose up -d

# If you get "network already in use"
docker network inspect obs-network
# If empty, remove it:
docker network rm obs-network
```

### Submodule Issues

```bash
# Check submodule status
git submodule status

# Reinitialize if needed
rm -rf .git/modules/observability
git submodule deinit -f observability
git submodule update --init --recursive observability
```

### Alloy Config Issues

```bash
# Check which config is being used
docker exec grafana-alloy cat /etc/alloy/config.alloy

# Check for parsing errors
docker logs grafana-alloy

# Check Alloy targets (metrics scraping)
curl http://localhost:12345/agent/api/v1/targets
```

### Migration Checklist

- [ ] Backup existing configs (if needed)
- [ ] Run `make obs-init` to initialize submodule
- [ ] Run `make up` to test full stack
- [ ] Verify Grafana at http://localhost:3000
- [ ] Verify metrics in Prometheus
- [ ] Verify logs in Loki
- [ ] Test `make down` and `make up` cycle
- [ ] (Optional) Configure remote mode for production

## Rollback

If you need to roll back, use git:

```bash
# Checkout pre-migration commit
git log --oneline --all
git checkout <commit-hash>

# Remove submodule
git submodule deinit -f observability
rm -rf .git/modules/observability
git rm -f observability

# Remove new configs
rm -rf configs/alloy
```

## Support

For observability-stack issues, visit:
- https://github.com/Giygas/observability-stack
- Read documentation in `observability/docs/`

For medicaments-api specific issues:
- Check `Makefile` targets: `make help`
- Check `.env.docker` configuration
- Check `configs/alloy/` configs
