# Observability Stack

A reusable, standalone observability stack for monitoring applications. Can be used independently as a remote service or as a submodule in app repos.

## Features

- **Grafana** - Visualization and dashboards
- **Prometheus** - Metrics collection and storage
- **Loki** - Log aggregation
- **Multiple Tunnel Options** - Cloudflare, Tailscale, WireGuard
- **App-Agnostic** - Works with any application using Alloy

## Quick Start

### Local Development (No Tunnel)

```bash
make setup
make up
```

### Remote with Cloudflare Tunnel

```bash
make setup
make up-cloudflare
```

### Remote with Tailscale

```bash
make setup
make up-tailscale
```

### Remote with WireGuard

```bash
make setup
make up-wireguard
```

## Architecture

Two operation modes:

### Mode 1: Standalone Remote (Production)

- Runs on a dedicated server/PC
- Exposes endpoints via tunnel (Cloudflare/Tailscale/WireGuard)
- Multiple apps connect remotely via Grafana Alloy
- **Alloy Config**: `config.remote.alloy` with WAL buffering

### Mode 2: Local Submodule (Dev/Staging)

- Added as git submodule in app repo
- Spins up with app via `make up`
- Everything runs on same Docker network
- **Alloy Config**: `config.alloy` (no WAL, direct container DNS)

### Safety: Default to Local Mode

**Default is always local** — remote mode requires explicit configuration:

```yaml
# docker-compose.yml in app repository
grafana-alloy:
  volumes:
    # Falls back to local config if ALLOY_CONFIG is not set
    - ./configs/alloy/${ALLOY_CONFIG:-config.alloy}:/etc/alloy/config.alloy:ro
```

```bash
# .env in app repository
ALLOY_CONFIG=config.alloy          # Local mode (default)
ALLOY_CONFIG=config.remote.alloy   # Remote mode (explicit)
```

This ensures production deployments don't accidentally use remote endpoints without proper configuration.

## Services

| Service    | Port | Description           |
| ---------- | ---- | --------------------- |
| Grafana    | 3000 | Web UI for dashboards |
| Prometheus | 9090 | Metrics endpoint      |
| Loki       | 3100 | Logs endpoint         |

## Usage

### Add to App Repository

```bash
git submodule add https://github.com/you/observability-stack.git observability
```

### Start Stack

```bash
make up                    # Local mode
make up-cloudflare         # Cloudflare tunnel
make up-tailscale          # Tailscale VPN
make up-wireguard          # WireGuard VPN
```

### Stop Stack

```bash
make down
```

### View Logs

```bash
make logs
```

### Check Status

```bash
make status
```

## Configuration

Environment variables are set in `.env`:

```bash
cp .env.example .env
```

**Note**: For detailed instructions on tunnels, see [Tunnel Setup Guide](docs/tunnels.md#cloudflare-tunnel-recommended).

### Grafana

```bash
GRAFANA_ADMIN_USER=admin
GRAFANA_PORT=3000
```

### Prometheus

```bash
PROMETHEUS_PORT=9090
PROMETHEUS_RETENTION=720h    # 30 days
```

### Loki

```bash
LOKI_PORT=3100
```

### Cloudflare Tunnel

```bash
CLOUDFLARE_TUNNEL_TOKEN=your_token
CF_ACCESS_CLIENT_ID=your_id
CF_ACCESS_CLIENT_SECRET=your_secret
```

### Tailscale

```bash
TAILSCALE_AUTHKEY=your_authkey
TS_HOSTNAME=obs-stack
```

### WireGuard

```bash
WG_SERVER_URL=your_server.com
WG_PORT=51820
WG_PEERS=1
```

## Connecting Applications

Applications connect via **Grafana Alloy** with two modes:

### Local Mode (`config.alloy`)

- Alloy in same app repo
- Direct container DNS resolution
- No auth headers needed

### Remote Mode (`config.remote.alloy`)

- Alloy in app repo, obs stack on remote machine
- WAL buffering for outage protection
- Optional auth headers (Cloudflare Access)

Example remote Alloy config:

```alloy
prometheus.remote_write "obs" {
  endpoint {
    url = env("PROMETHEUS_URL")
    headers = {
      "CF-Access-Client-Id"     = env("CF_ACCESS_CLIENT_ID"),
      "CF-Access-Client-Secret" = env("CF_ACCESS_CLIENT_SECRET"),
    }
  }
}
```

## Documentation

- [Tunnel Setup Guide](docs/tunnels.md)
- [Remote Mode Setup](docs/remote-setup.md)
- [Local Mode Setup](docs/local-setup.md)

## Default Credentials

- **Grafana**: `admin` / `admin` (change in `secrets/grafana_password.txt`)

## Data Retention

- **Prometheus**: 30 days (720h) - configurable
- **Loki**: 30 days (720h) - configurable

## Volumes

Persistent data is stored in named volumes:

- `loki-data` - Log chunks
- `prometheus-data` - Time-series data
- `grafana-data` - Dashboards and settings

## Security

- Grafana password file (secrets/)
- Optional Cloudflare Access protection
- Tunnel-based encryption (no public ports)
- Network isolation via Docker networks

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT
