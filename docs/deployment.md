# Deployment

Email, sign-in limits and password links are in [Email and security](email-and-security.md).

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

## First run

A new server has no users. To stop others from claiming it first, creating the admin needs a one-time **bootstrap password**. The server prints it in the log and saves it in `bootstrap-password` in the data directory. It stops working, and the file is removed, once the first user exists.

| Install           | Read the password                                                           |
|-------------------|-----------------------------------------------------------------------------|
| Docker            | `docker logs savvy` or `docker exec savvy cat /data/bootstrap-password`     |
| Debian/RPM        | `journalctl -u savvy-go` or `sudo cat /var/lib/savvy-go/bootstrap-password` |
| Standalone binary | the terminal, or `bootstrap-password` in the data directory                 |

1. Open the app, enter the password and create the admin account.
2. A short **server setup** follows. Step 1 (required) is the public URL. Click **Use <address>** to fill in the address you opened, then continue; **Administration → System** has the same button. Step 2 (optional) is [email](email-and-security.md#email) and single sign-on; each is marked **Done** once set. Skip or finish whenever you like: setup comes back after a reload or new sign-in until you do. The same settings stay in **Administration → System**.
3. The admin starts with no space. Create the first one in the app, see [Spaces](spaces.md#creating-a-space).

Good to know:

- A restart before the first user exists keeps the same password.
- A server started with demo data has users already and needs none.
- The URL step saves `app_url` to `config.toml`. When the file is read-only it only explains how to set it by hand.

**Moving from another server?** On the same page, restore a server backup instead of creating an admin. Enter the bootstrap password and upload the backup (for a Laravel database, also its `APP_KEY`). Users, spaces, settings and the signing key replace the empty server, and you sign in as the backup's admin. A failed restore changes nothing. See [Restoring on first run](data-and-backups.md#restoring-on-first-run).

## Configuration

Configuration is a TOML file; no environment variables are read.

- `config.toml` is created with defaults on first start, in the data directory (`/data` in the image, `/var/lib/savvy-go` for the Debian package, `./savvy-data` otherwise).
- `-config <path>` or `CONFIG_FILE` points elsewhere.
- On every start the file is read and written back: new keys appear, comments are regenerated, your values are kept. A read-only file works, it is just not rewritten.
- Edit it and restart, or change `app_url` and the `[mailer]` section in **Administration → System**, which writes the file and applies at once. Everything else (`listen`, `timezone`, paths, security) is edited by hand and needs a restart.
- Demo data is not configured here, see `--seed-config` in [Development](development.md).

Keys and defaults are listed in the generated file.

| Section      | Keys |
|--------------|------|
| `[server]`   | `listen` (`:80` in the image, `localhost:8080` otherwise), `app_url`, `timezone` |
| `[paths]`    | `data`, `uploads`, `backups` |
| `[security]` | `session_ttl`, `remember_ttl`, `challenge_ttl`, `reset_ttl`, `ip_rate_limit`, `account_rate_limit`, `session_cookie`, `csrf_cookie`, `csrf_header` |
| `[mailer]`   | `enabled`, `host`, `port`, `protocol`, `user`, `password`, `from`, `skip_verify`, `timeout` |

An unusable value (a lifetime that is not a positive number, an empty cookie name) stops the server at start, naming the section and key. `[mailer]` is the exception, see [Email](email-and-security.md#email).

**`app_url`** is the public `https://` URL of your instance. SSO and passkeys stay disabled without it. Passkeys need user verification (PIN or biometric) to register and sign in, so a security key without a PIN cannot be used. Change it in the admin panel (System → Server) or in the file. The panel applies it at once, with no restart. Mind the scheme: the session cookie depends on it (see below), so switching between `http://` and `https://` signs everyone out, and if you reach the app over plain `http://` you cannot sign back in until you fix `app_url` in the file (in Docker, `/data/config.toml`) and restart.

Limits such as spaces per user, space quotas and kept backups are not in the file. Server admins set them in **Administration → System**.

## Behind a reverse proxy

Set `app_url` to your public `https://` URL. `X-Forwarded-For` is honored; nothing else to set up.

- **HTTPS comes from `app_url`, not request headers.** With an `https://` URL the session cookie is `Secure` and named `__Host-svy_session`, and only it signs anyone in. `X-Forwarded-Proto` is not trusted, since anyone who can reach the server directly could set it. Without `app_url`, a request counts as HTTPS only when the connection itself is TLS. Behind a TLS-terminating proxy, `app_url` must be `https://`, or cookies stay plain.
- **Cross-site requests are refused.** Sign-in and the other public POSTs (login, 2FA, password links) need an `Origin` equal to the origin of `app_url` (the request's own host if unset) and a `application/json` body. A proxy that rewrites `Origin` breaks sign-in. Scripts that send no `Origin` are unaffected.

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

| Endpoint  | Meaning                                                       | Healthy | Unhealthy |
|-----------|---------------------------------------------------------------|---------|-----------|
| `/livez`  | The process is up. Use for restart decisions.                 | `200`   | —         |
| `/readyz` | Databases are reachable and migrations have run. Gate traffic on it. | `200` | `503` |

During maintenance `/livez` stays up and `/readyz` returns `503`. A space whose migration fails answers `503` on its own; other spaces keep working.

## Server checks

**Administration → Status** (server admins only) lists what needs attention; a dot on the Administration menu shows when something does. Each entry links to the page that fixes it. The list is computed fresh each time, so fixed problems disappear. Also available at `GET /api/admin/checks` (session only).

| Check                      | Level                        | When |
|----------------------------|------------------------------|------|
| Public URL is not set      | warning                      | `app_url` is empty |
| Email is not set up        | warning                      | `[mailer]` is off |
| Email settings are invalid | error                        | `[mailer]` failed validation at start, so mail is off |
| No server backup           | warning                      | no valid server backup exists |
| Server backup is old       | warning                      | newest valid server backup is over 30 days old |
| Data disk is filling       | warning at 75%, error at 90% | the volume that holds the uploads |
| Config file is read-only   | info                         | `config.toml` cannot be written (read-only mount), so `app_url` and the mailer are edited by hand |

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

For Debian 13 (Trixie) or later:

```bash
curl -fsSLO https://github.com/chiririll/savvy-go/releases/latest/download/savvy-go.deb
sudo apt install ./savvy-go.deb
```

- Runs as the `savvy-go` systemd service (`/usr/bin/savvy-go`).
- Data: `/var/lib/savvy-go`. Config: `/etc/savvy-go/config.toml`, created and maintained by the service (restart after editing).
- Listens on `127.0.0.1:8080` only. Put a reverse proxy in front, or set `listen = ":8080"` under `[server]`.
- `apt purge savvy-go` deletes the data.

## RPM package

For Fedora, RHEL (and derivatives) or openSUSE:

```bash
curl -fsSLO https://github.com/chiririll/savvy-go/releases/latest/download/savvy-go.rpm
sudo dnf install ./savvy-go.rpm
```

Same as the Debian package, but runs as its own `savvy-go` system user. Removing the package keeps `/var/lib/savvy-go` and `/etc/savvy-go`; delete them to wipe the data.

## Standalone binary

Each GitHub release has a single-file build for Linux (`savvy-go-linux-amd64`) and Windows (`savvy-go-windows-amd64.exe`). The web UI is built in. Run it and open `http://localhost:8080`. Check the download against `SHA256SUMS` from the release.

- Data and `config.toml` go to a `savvy-data` folder in the **working directory**, not next to the executable (on Linux, `/data` or `/var/lib/savvy-go` is used instead when it exists).
- Double-clicking the `.exe` starts it in its own folder, but a shortcut or scheduled task may not. Set the working directory, or pass `--config <path>` (or `CONFIG_FILE`).
- Change folders under `[paths]` in the config.
- It is not a service. On Linux use the packages; on Windows use NSSM or Task Scheduler.

## Updating

```bash
docker compose pull
docker compose up -d
```

Data stays in the volume and migrations run on start. With the Debian or RPM package, install the newer package.

Coming from Laravel: see [Upgrading from the Laravel version](data-and-backups.md#upgrading-from-the-laravel-version).
