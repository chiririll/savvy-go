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

All data is in the `/data` volume, see [Data and backups](data-and-backups.md).

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

Set `APP_URL` to your public `https://` URL. The `X-Forwarded-Proto` and `X-Forwarded-For` headers are honored, no other setup needed.

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

| Endpoint  | Meaning                                                      | Healthy | Unhealthy |
|-----------|--------------------------------------------------------------|---------|-----------|
| `/livez`  | The process is up. Use for restart decisions.                | `200`   | —         |
| `/readyz` | Databases are reachable and all migrations have run. Use to gate traffic. | `200`   | `503`     |

During maintenance `/livez` stays up and `/readyz` returns `503`. A space whose migration fails answers `503` on its own; other spaces keep working.

## Kubernetes

Use a single-replica `Deployment` with a `PersistentVolumeClaim` at `/data` and `strategy: { type: Recreate }` (SQLite is single-writer). The container runs as non-root (`www-data`, uid 82), so grant `NET_BIND_SERVICE` to bind port 80. Probes:

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

It runs as the `savvy-go` systemd service (`/usr/bin/savvy-go`). Data is in `/var/lib/savvy-go`; settings (`APP_URL`, `TZ`) go in `/etc/savvy-go/install.env`. `apt purge savvy-go` deletes the data.

## RPM package

On Fedora, RHEL (and derivatives) or openSUSE, install the `.rpm` from the GitHub release:

```bash
curl -fsSLO https://github.com/chiririll/savvy-go/releases/latest/download/savvy-go.rpm
sudo dnf install ./savvy-go.rpm
```

It behaves like the Debian package but runs as its own `savvy-go` system user. Removing the package keeps `/var/lib/savvy-go` and `/etc/savvy-go`; delete them yourself to wipe the data.

## Updating

```bash
docker compose pull
docker compose up -d
```

Data stays in the volume and migrations run on start. With the Debian package, install the newer `.deb` or `.rpm`.

Coming from Laravel: see [Upgrading from the Laravel version](data-and-backups.md#upgrading-from-the-laravel-version).
