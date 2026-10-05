# Deployment

## Docker Compose (recommended)

```yaml
services:
  savvy:
    image: chiririll/savvy-go:latest
    container_name: savvy
    restart: unless-stopped
    ports:
      - "3000:80"
    volumes:
      - savvy-go-data:/data
    environment:
      - APP_URL=https://savvy.yourdomain.com
      - TZ=Europe/Belgrade
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://127.0.0.1/livez"]
      interval: 10s
      timeout: 3s
      start_period: 60s
      retries: 3

volumes:
  savvy-go-data:
```

Everything the instance keeps is in the one volume mounted at `/data`; see [Data and backups](data-and-backups.md).

## Environment variables

| Variable      | Description                                                                       | Default             |
|---------------|-----------------------------------------------------------------------------------|---------------------|
| `APP_URL`     | Public URL of your instance                                                       | `http://localhost`  |
| `TZ`          | Timezone                                                                          | `UTC`               |
| `SEED_DEMO`   | First boot only: seed demo users, spaces, accounts and ~12 months of transactions | `false`             |
| `SEED_DATE`   | With `SEED_DEMO`: the day (`YYYY-MM-DD`) the demo data is placed relative to, for reproducible data | today |
| `SEED_MANIFEST` | With `SEED_DEMO`: file to write the seeded users, spaces, invitation tokens and rule ids to (JSON) | none |
| `DATA_DIR`    | Where all state lives                                                             | `/data` if it exists, else `/var/lib/savvy-go`, else `./data` |
| `UPLOAD_ROOT` | Optional override for uploads                                                     | `$DATA_DIR/uploads` |
| `BACKUP_PATH` | Optional override for backups                                                     | `$DATA_DIR/backups` |

Limits such as spaces per user, space quotas and the number of kept backups are not environment variables: server admins set them in **Administration → System**.

## Behind a reverse proxy

Set `APP_URL` to your public `https://` URL. Savvy honors the `X-Forwarded-Proto` and `X-Forwarded-For` headers from the proxy, so HTTPS link generation and real client IPs work without extra configuration.

### Traefik (HTTPS)

```yaml
services:
  savvy:
    image: chiririll/savvy-go:latest
    container_name: savvy
    restart: unless-stopped
    volumes:
      - savvy-go-data:/data
    environment:
      - APP_URL=https://savvy.yourdomain.com
      - TZ=Europe/Belgrade
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.savvy.rule=Host(`savvy.yourdomain.com`)"
      - "traefik.http.routers.savvy.entrypoints=websecure"
      - "traefik.http.routers.savvy.tls.certresolver=letsencrypt"
      - "traefik.http.services.savvy.loadbalancer.server.port=80"
    networks:
      - traefik

volumes:
  savvy-go-data:

networks:
  traefik:
    external: true
```

### Nginx Proxy Manager

1. Run Savvy on an internal port:

```yaml
services:
  savvy:
    image: chiririll/savvy-go:latest
    container_name: savvy
    restart: unless-stopped
    expose:
      - "80"
    volumes:
      - savvy-go-data:/data
    environment:
      - APP_URL=https://savvy.yourdomain.com
    networks:
      - npm-network

volumes:
  savvy-go-data:

networks:
  npm-network:
    external: true
```

2. In Nginx Proxy Manager, create a proxy host pointing to `savvy:80`.

## Health checks

Two probe endpoints are exposed for orchestrators and uptime monitoring (responses use the IETF `application/health+json` format):

| Endpoint  | Purpose                                                                                                  | Healthy | Unhealthy |
|-----------|----------------------------------------------------------------------------------------------------------|---------|-----------|
| `/livez`  | Liveness — the app process is up. Use it for container restart decisions.                                | `200`   | —         |
| `/readyz` | Readiness — databases are reachable and migrations of the server and every space have run. Gate traffic. | `200`   | `503`     |

`/livez` stays up during maintenance mode; `/readyz` returns `503` so traffic drains while the instance is not ready. A space whose migration fails is marked unavailable and answers `503` for its own requests while the other spaces keep working.

## Kubernetes

Deploy as a single-replica `Deployment` with a `PersistentVolumeClaim` mounted at `/data`. SQLite is single-writer, so use `strategy: { type: Recreate }`. The container runs as non-root (`www-data`, uid 82) — grant `NET_BIND_SERVICE` so it can bind port 80. Wire the probes to the health endpoints:

```yaml
        startupProbe:
          httpGet: { path: /livez, port: 80 }
          periodSeconds: 3
          failureThreshold: 30
        livenessProbe:
          httpGet: { path: /livez, port: 80 }
          periodSeconds: 10
        readinessProbe:
          httpGet: { path: /readyz, port: 80 }
          periodSeconds: 10
```

## Debian package

On Debian 13 (Trixie) or later, install the `.deb` from the GitHub release:

```bash
curl -fsSLO https://github.com/chiririll/savvy-go/releases/latest/download/savvy-go.deb
sudo apt install ./savvy-go.deb
```

The binary is installed to `/usr/bin/savvy-go` and runs as the `savvy-go` systemd service. Data lives in `/var/lib/savvy-go`. Optional settings (`APP_URL`, `TZ`) go in `/etc/savvy-go/install.env`. `apt purge savvy-go` removes the data directory.

## Tarball

Release `savvy-go.tar.gz` contains the `savvy-go` binary and the static SPA assets (`public/`). On any Linux host:

```bash
mkdir -p /opt/savvy-go && tar -C /opt/savvy-go -xzf savvy-go.tar.gz
DATA_DIR=/var/lib/savvy-go PUBLIC_DIR=/opt/savvy-go/public LISTEN_ADDR=:8080 APP_URL=https://savvy.example.com /opt/savvy-go/savvy-go
```

Point a reverse proxy at `:8080`.

## Updating

```bash
docker compose pull
docker compose up -d
```

Your data stays in the `/data` volume, and migrations run on start. With the Debian package, install the newer `.deb`; data stays in `/var/lib/savvy-go`.

A Laravel install's `database.sqlite` is converted on first start; see [Upgrading from the Laravel version](data-and-backups.md#upgrading-from-the-laravel-version).
