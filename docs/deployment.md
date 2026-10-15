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

## First run

A new server has no users, and creating the first admin needs a one-time **bootstrap password**, so nobody who finds the server first can claim it. The server prints it in the log at start and saves it in the file `bootstrap-password` in the data directory. Open the app, enter the password and create the admin account. The password stops working, and the file is removed, as soon as the first user exists.

| Install          | Read the password                                                           |
|------------------|-----------------------------------------------------------------------------|
| Docker           | `docker logs savvy` or `docker exec savvy cat /data/bootstrap-password`     |
| Debian/RPM       | `journalctl -u savvy-go` or `sudo cat /var/lib/savvy-go/bootstrap-password` |
| Standalone binary| the terminal, or `bootstrap-password` in the data directory                 |

A restart before the first user exists keeps the same password. A server started with demo data has users already and needs none.

Moving from another server? Instead of creating an admin, choose to restore a server backup on the same page: enter the bootstrap password and upload the backup (for a database of the Laravel version, also its `APP_KEY`). The users, spaces, settings and signing key of the backup replace the empty server, the password stops working, and you sign in as the admin from the backup; admin and server setup are skipped. A failed restore (wrong file or key) changes nothing and the password stays valid. See [Data and backups](data-and-backups.md#restoring-on-first-run).

The admin starts with no space; the first one is created from the app, see [Spaces](spaces.md#creating-a-space).

Right after the admin is created, a short **server setup** follows. Step 1 is required: the public URL, filled in with the address you opened the app at. Step 2 is a list of optional things, each opening on its own screen and marked **Done** once configured: email (the same form as in the admin panel, and a test message) and single sign-on (add an identity provider). Finish or skip whenever you like. The server remembers that setup is pending, so if you reload, close the tab or sign in again, it comes back until you finish or skip it; after that it is hidden for good, and the same settings stay in **Administration → System**. If `config.toml` is read-only, the URL step only explains how to set `app_url` in the file.

## Configuration

The application is configured by a TOML file and reads no environment variables for settings. `config.toml` is created with the defaults on the first start, in the data directory (`/data` in the image, `/var/lib/savvy-go` for the Debian package, `./savvy-data` otherwise); `-config <path>` or the `CONFIG_FILE` environment variable points elsewhere. Demo data is not configured here: see `--seed-config` in [Development](development.md). On every start the file is read and written back, so keys a newer version adds appear on their own; comments are regenerated, your values are kept. Edit it and restart. A read-only file works too, it is just not rewritten.

Keys, with their defaults, are listed in the generated file itself:

| Section      | Keys                                                                                              |
|--------------|---------------------------------------------------------------------------------------------------|
| `[server]`   | `listen` (`:80` in the image, `localhost:8080` otherwise), `app_url`, `timezone`                  |
| `[paths]`    | `data`, `uploads`, `backups`                                                                      |
| `[security]` | `session_ttl`, `remember_ttl`, `challenge_ttl`, `session_cookie`, `csrf_cookie`, `csrf_header` |
| `[mailer]`   | `enabled`, `host`, `port`, `protocol`, `user`, `password`, `from`, `skip_verify`, `timeout`       |

`app_url` is the public `https://` URL of your instance. SSO and passkeys stay disabled without it. It can also be changed in the admin panel (System → Server), which writes it back to the file. If Go Savvy cannot write the file (a read-only mount), the field is locked in the panel and you edit the file yourself. Behind Docker, set it there, or edit `/data/config.toml` once.

Limits such as spaces per user, space quotas and the number of kept backups are not in the file: server admins set them in **Administration → System**.

### Email

Without a mail server everything works; reset and invitation links are shown to copy and send yourself. With one, Go Savvy can email them. Set it up in **Administration → System → Email**, or in the `[mailer]` section of `config.toml`. Saving in the panel writes the section and takes effect at once, no restart; if `config.toml` is read-only, the form is locked and you edit the file yourself.

```toml
[mailer]
enabled = true
host = "smtp.example.com"
port = 587
protocol = "starttls"
user = "savvy@example.com"
password = "app-password"
from = "Go Savvy <savvy@example.com>"
```

| `protocol`           | Port | Connection                                                                           |
|----------------------|------|--------------------------------------------------------------------------------------|
| `starttls` (default) | 587  | Starts plain and upgrades to TLS. Never falls back to plain text if the server cannot. |
| `smtps`              | 465  | TLS from the first byte.                                                             |
| `smtp`               | 25   | No encryption. Only for a trusted relay on the same host or network.                 |

- Settings are saved as a whole. The password field is write-only: leave it empty to keep the stored one, and it is cleared when you remove the login.
- The TLS certificate is checked. `skip_verify = true` accepts any certificate, for a server with a self-signed one.
- `user` and `password` are optional (leave `user` empty for a relay without login). The password is stored in plain text in `config.toml`, so keep the file readable by the application only. The admin panel never shows it again after you save it.
- `timeout` is how many seconds to wait for the server (default 10).
- A mistake in the section does not stop the server: mail is switched off, the log says why, and **Administration → System → Email** shows the problem.

That page has a **Send test email** button, to try the saved settings. When mail works and `app_url` is set:

- **Administration → Users → Email a reset link** sends a user a one-time password link (valid 7 days). It is not offered for yourself, for SSO-only users, or when password sign-in is off. **Reset password** still shows a link to copy.
- Inviting someone by email from **Settings → Space** also emails the invitation link. The link is always shown too, in case the email fails.

Any space admin can make the server email an address by creating an invitation. There is no rate limit yet; keep that in mind on a server open to untrusted space admins.

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

## Standalone binary

Each GitHub release has a single-file build for Linux (`savvy-go-linux-amd64`) and Windows (`savvy-go-windows-amd64.exe`). The web UI is built in and there is nothing else to install. Run it and open `http://localhost:8080`. Data and `config.toml` go to a `savvy-data` folder in the working directory, not next to the executable (on Linux, `/data` or `/var/lib/savvy-go` is used instead when it exists). Double-clicking the `.exe` starts it in its own folder, but a shortcut or a scheduled task may start it somewhere else, so set the working directory there, or pass `--config <path>` (or `CONFIG_FILE`) to fix the location. `data` and the other folders can be changed under `[paths]` in the config. Compare the download with `SHA256SUMS` from the release.

It does not run as a service by itself; use the packages above on Linux, or a tool such as NSSM or Task Scheduler on Windows.

## Updating

```bash
docker compose pull
docker compose up -d
```

Data stays in the volume and migrations run on start. With the Debian package, install the newer `.deb` or `.rpm`.

Coming from Laravel: see [Upgrading from the Laravel version](data-and-backups.md#upgrading-from-the-laravel-version).
