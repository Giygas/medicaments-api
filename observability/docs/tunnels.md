# Tunnel Setup Guide

This guide explains how to configure tunnels for remote observability access.

## Overview

Tunnels allow your observability stack to be accessed from remote production servers without exposing public ports. This is more secure than opening ports and easier to manage.

## Available Tunnels

| Tunnel         | Use Case                | Setup Difficulty | Features                                     |
| -------------- | ----------------------- | ---------------- | -------------------------------------------- |
| **Cloudflare** | Public access with auth | Easy             | Built-in auth, CDN, SSL, DDoS protection     |
| **Tailscale**  | Private mesh network    | Medium           | Encrypted mesh, NAT traversal, simple        |
| **WireGuard**  | Self-hosted VPN         | Hard             | Lightweight, high performance, manual config |

## Cloudflare Tunnel (Recommended)

### Prerequisites

- Cloudflare account
- Domain managed by Cloudflare

### Setup

1. **Create Tunnel**

   ```bash
   # Go to Cloudflare Zero Trust → Tunnels → Create tunnel
   # Choose "Cloudflared" → Copy your token
   ```

2. **Configure Tunnel Hostnames**

   In Cloudflare Zero Trust → Tunnels → your tunnel → Public Hostnames:

   | Hostname                        | Service    | URL                      |
   | ------------------------------- | ---------- | ------------------------ |
   | `prometheus-obs.yourdomain.com` | Prometheus | `http://prometheus:9090` |
   | `loki-obs.yourdomain.com`       | Loki       | `http://loki:3100`       |
   | `grafana-obs.yourdomain.com`    | Grafana    | `http://grafana:3000`    |

3. **Setup Cloudflare Access (Optional but Recommended)**

   Go to Zero Trust → Access → Service Auth → Service Tokens:
   - Create service token → Copy ID and Secret

4. **Configure .env**

   ```bash
   CLOUDFLARE_TUNNEL_TOKEN=your_token_here
   CF_ACCESS_CLIENT_ID=your_client_id
   CF_ACCESS_CLIENT_SECRET=your_client_secret
   ```

5. **Start Stack**
   ```bash
   make up-cloudflare
   ```

### URLs for Remote Apps

After setup, your remote endpoints are:

```
PROMETHEUS_URL=https://prometheus-obs.yourdomain.com/api/v1/write
LOKI_URL=https://loki-obs.yourdomain.com/loki/api/v1/push
GRAFANA_URL=https://grafana-obs.yourdomain.com
```

## Tailscale VPN

### Prerequisites

- Tailscale account

### Setup

1. **Create Auth Key**

   ```bash
   # Go to tailscale.com → Settings → Auth Keys → Generate auth key
   # Copy the key
   ```

2. **Configure .env**

   ```bash
   TAILSCALE_AUTHKEY=tskey-auth-your-key-here
   TS_HOSTNAME=obs-stack
   ```

3. **Start Stack**

   ```bash
   make up-tailscale
   ```

4. **Find Your Tailscale IP**

   ```bash
   docker exec tailscale-obs tailscale ip -4
   # Example: 100.x.x.x
   ```

5. **Connect Production Server**

   ```bash
   # Install Tailscale on production server
   curl -fsSL https://tailscale.com/install.sh | sh

   # Connect to your tailnet
   sudo tailscale up --authkey=your_auth_key
   ```

### URLs for Remote Apps

```
PROMETHEUS_URL=http://100.x.x.x:9090/api/v1/write
LOKI_URL=http://100.x.x.x:3100/loki/api/v1/push
GRAFANA_URL=http://100.x.x.x:3000
```

## WireGuard VPN

### Prerequisites

- Public IP or domain name
- Router with port forwarding capability

### Setup

1. **Configure .env**

   ```bash
   WG_SERVER_URL=your-public-ip.com
   WG_PORT=51820
   WG_PEERS=1
   ```

2. **Start Stack**

   ```bash
   make up-wireguard
   ```

3. **Get Peer Configuration**

   ```bash
   docker logs wireguard-obs
   # Look for "Peer #1" section with config
   ```

4. **Configure Production Server**

   Copy the peer config to production server `/etc/wireguard/wg0.conf`:

   ```ini
   [Interface]
   PrivateKey = your_private_key
   Address = 10.13.13.2/24
   DNS = 1.1.1.1

   [Peer]
   PublicKey = server_public_key_from_logs
   Endpoint = your-server-ip:51820
   AllowedIPs = 10.13.13.0/24
   PersistentKeepalive = 25
   ```

5. **Start WireGuard on Production**
   ```bash
   sudo wg-quick up wg0
   ```

### URLs for Remote Apps

```
PROMETHEUS_URL=http://10.13.13.1:9090/api/v1/write
LOKI_URL=http://10.13.13.1:3100/loki/api/v1/push
GRAFANA_URL=http://10.13.13.1:3000
```

## Comparison

| Feature         | Cloudflare          | Tailscale         | WireGuard |
| --------------- | ------------------- | ----------------- | --------- |
| Public Access   | ✅ Yes              | ❌ No             | ❌ No     |
| Built-in Auth   | ✅ Yes              | ❌ No             | ❌ No     |
| SSL/TLS         | ✅ Automatic        | ✅ Automatic      | ⚠️ Manual |
| DDoS Protection | ✅ Yes              | ❌ No             | ❌ No     |
| Setup Time      | ~5 min              | ~10 min           | ~30 min   |
| Cost            | Free tier available | Free for personal | Free      |
| Maintenance     | Low                 | Low               | Medium    |

## Troubleshooting

### Cloudflare Tunnel Not Starting

```bash
# Check logs
docker logs cloudflared-obs

# Verify token
echo $CLOUDFLARE_TUNNEL_TOKEN | grep -q "^ey" && echo "Valid" || echo "Invalid"
```

### Tailscale Not Connecting

```bash
# Check logs
docker logs tailscale-obs

# Verify auth key
echo $TAILSCALE_AUTHKEY | grep -q "^tskey" && echo "Valid" || echo "Invalid"

# Check IP
docker exec tailscale-obs tailscale status
```

### WireGuard Connection Issues

```bash
# Check logs
docker logs wireguard-obs

# Verify port forwarding
sudo netstat -tulpn | grep 51820

# Check peer status
sudo wg show
```

## Security Best Practices

1. **Always use Cloudflare Access** with Cloudflare Tunnel
2. **Rotate auth keys** regularly (Tailscale/WireGuard)
3. **Use strong passwords** for Grafana
4. **Limit access IPs** in Cloudflare Access policies
5. **Monitor logs** for suspicious activity
6. **Keep Docker images** updated
7. **Use secrets management** (don't commit secrets)

## Next Steps

- [Remote Mode Setup](remote-setup.md)
- [Local Mode Setup](local-setup.md)
