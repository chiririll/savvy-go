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

## Configuration

The application is configured by a TOML file and reads no environment variables for settings. `config.toml` is created with the defaults on the first start, in the data directory (`/data` in the image, `/var/lib/savvy-go` for the Debian package, `./data` otherwise); `-config <path>` or the `CONFIG_FILE` environment variable points elsewhere. Demo data is not configured here: see `--seed-config` in [Development](development.md). On every start the file is read and written back, so keys a newer version adds appear on their own; comments are regenerated, your values are kept. Edit it and restart. A read-only file works too, it is just not rewritten.

Keys, with their defaults, are listed in the generated file itself:

| Section      | Keys                                                                                              |
|--------------|---------------------------------------------------------------------------------------------------|
| `[server]`   | `listen` (`:80` in the image, `localhost:8080` otherwise), `app_url`, `timezone`                  |
| `[paths]`    | `data`, `uploads`, `backups`, `public`                                                            |
| `[security]` | `session_ttl`, `remember_ttl`, `challenge_ttl`, `session_cookie`, `csrf_cookie`, `csrf_header` |

`app_url` is the public `https://` URL of your instance. SSO and passkeys stay disabled without it. It can also be changed in the admin panel (System → Server), which writes it back to the file. Behind Docker, set it there, or edit `/data/config.toml` once.

Limits such as spaces per user, space quotas and the number of kept backups are not in the file: server admins set them in **Administration → System**.

## Behind a reverse proxy

Set `app_url` to your public `https://` URL. The `X-Forwarded-Proto` and `X-Forwarded-For` headers are honored, no other setup needed.

### Traefik (HTTPS)

```yaml
services:
  savvy:
    image: chiririll/savvy-go:latest
    container_name: savvy
    restart: unless-stopped
    volumes:
      - savvy-go-data:/data
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

It runs as the `savvy-go` systemd service (`/usr/bin/savvy-go`). Data is in `/var/lib/savvy-go`; settings are in `/etc/savvy-go/config.toml`, which the service creates and maintains itself (restart after editing). It listens on `127.0.0.1:8080` only; put a reverse proxy in front, or set `listen = ":8080"` under `[server]` to expose it. `apt purge savvy-go` deletes the data.

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
