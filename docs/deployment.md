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

Right after the admin is created, a short **server setup** follows. Step 1 is required: the public URL, filled in with the address you opened the app at. Step 2 is a list of optional things, each opening on its own screen and marked **Done** once configured: email (the same form as in the admin panel, and a test message) and single sign-on (add an identity provider). Finish or skip whenever you like. The server remembers that setup is pending, so if you reload, close the tab or sign in again, it comes back until you finish or skip it; after that it is hidden for good, and the same settings stay in **Administration → System**. While the application does not change `config.toml`, the URL step only explains how to set `app_url` in the file.

## Configuration

The application is configured by a TOML file and reads no environment variables for configuration. `config.toml` is created with the defaults on the first start, in the data directory (`/data` in the image, `/var/lib/savvy-go` for the Debian package, `./savvy-data` otherwise); `-config <path>` or the `CONFIG_FILE` environment variable points elsewhere. Demo data is not configured here: see `--seed-config` in [Development](development.md). On every start the file is read and written back, so keys a newer version adds appear on their own; comments are regenerated, your values are kept. Edit it and restart. A read-only file works too, it is just not rewritten.

Keys, with their defaults, are listed in the generated file itself:

| Section      | Keys                                                                                              |
|--------------|---------------------------------------------------------------------------------------------------|
| `[server]`   | `listen` (`:80` in the image, `localhost:8080` otherwise), `app_url`, `timezone`                  |
| `[paths]`    | `data`, `uploads`, `backups`                                                                      |
| `[security]` | `session_ttl`, `remember_ttl`, `challenge_ttl`, `reset_ttl`, `ip_rate_limit`, `account_rate_limit`, `session_cookie`, `csrf_cookie`, `csrf_header` |
| `[mailer]`   | `enabled`, `host`, `port`, `protocol`, `user`, `password`, `from`, `skip_verify`, `timeout`       |

A value the application cannot run with, such as a lifetime that is not a positive number or an empty cookie name under `[security]`, stops it at start with the section and key in the error. `[mailer]` is the exception: see [Email](#email).

`app_url` is the public `https://` URL of your instance. SSO and passkeys stay disabled without it. A passkey needs user verification (a PIN or biometric) both to register and to sign in, so a security key with no PIN set cannot be used. The admin panel (System → Server) shows it but does not change it for now: edit the file and restart. Behind Docker, edit `/data/config.toml` once.

Limits such as spaces per user, space quotas and the number of kept backups are not in the file: server admins set them in **Administration → System**.

### Email

Without a mail server everything works; reset and invitation links are shown to copy and send yourself. With one, Go Savvy can email them, and some things need it: see [how invitations are made](spaces.md#invitations). Set it up in the `[mailer]` section of `config.toml` and restart. **Administration → System → Email** shows the mail configuration and tests it, but does not save changes for now: the form is locked.

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

- The TLS certificate is checked. `skip_verify = true` accepts any certificate, for a server with a self-signed one.
- `user` and `password` are optional (leave `user` empty for a relay without login). The password is stored in plain text in `config.toml`, so keep the file readable by the application only. The admin panel never shows it.
- `timeout` is how many seconds to wait for the server (default 10).
- A mistake in the section does not stop the server: mail is switched off, the log says why, and **Administration → System → Email** shows the problem.

That page has a **Send test email** button, to try the saved settings. When mail works and `app_url` is set:

- **Administration → Users → Email a reset link** sends a user a one-time password link (valid 1 hour, see [Password links](#password-links)). It is not offered for yourself, for SSO-only users, or when password sign-in is off. **Reset password** still shows a link to copy.
- **Administration → Users → New user** (without a password) can email the set-password link (valid 7 days) to the new user instead of showing it. With email verification switched on, emailing is the only option, since a link you hand over proves nothing about the mailbox.
- Inviting someone by email from **Settings → Space** emails them the invitation link. The link is not shown to the inviter, so it can only be used by that mailbox. Which invitations a server allows is described under [Invitations](spaces.md#invitations).

### Email confirmation

Every account has a confirmed or unconfirmed email address. An address becomes confirmed when:

- the user opens the link in a confirmation email (valid 7 days);
- they register from the link in an invitation **email**. The link goes only to the invited address and nobody else sees it, so using it proves the mailbox. An account created from a link someone copied from the app starts unconfirmed;
- they sign in through SSO and the provider reports the address as verified;
- a server admin marks it in **Administration → Users** (**Mark email as confirmed**). Changing a user's address clears the mark.

By default nothing depends on this. Turn on **Administration → System → Require email confirmation** and users with an unconfirmed address can still sign in, but see only a "confirm your email" page (with a button to resend the link) until they confirm. The same applies to their API tokens. The first admin starts unconfirmed too.

The switch can only be turned on when mail and `app_url` are set up and your own address is confirmed, so you cannot lock yourself out; the page offers to email you a link. Turning it off is always possible. The setting is stored in the database, so restoring a backup brings it back as it was: if you restore a server whose admins are all unconfirmed while it is on, nobody can pass the page until mail works and someone confirms through the link (or you restore a backup where it is off). A confirmation link is a random token stored hashed in the database. It works once, a newer link replaces the older one, and changing the address ends it.

Any space admin can make the server email an address by creating an invitation. To keep that from being used to flood a mailbox, an address cannot be invited again for one to two hours after an invitation to it was revoked or ran out, whichever space it was for. A server admin can end that wait early in **Administration → System → Invitation cooldown**, for one address or all.

### Two-factor lockout

A wrong two-factor code cannot be guessed at forever. After the password step, a sign-in has **5 tries**: the fifth wrong code (authenticator or recovery code) ends it, and the user has to enter the password again.

Starting again does not give a fresh start. After **10 wrong codes in a row**, across sign-ins, the account gets no new two-factor sign-in for **1 minute**; every further wrong code doubles the wait, up to **1 hour**. During the wait, sign-in with the right password answers "Too many wrong verification codes" (`429`). A right code clears the count. The lockout is per account and cannot be changed in the settings.

Each code works once. An authenticator code is accepted a single time, so one that was seen over a shoulder or in a log cannot sign in again, and the next code has to come from a later 30 second step. After a sign-in, wait for the app to show a new code before the next step that asks for one, such as turning two-factor off. A recovery code is likewise used up by its first success.

### Password links

A link that sets a password works once, and a newer link for the same user replaces the older one. How long it works depends on what it is for:

- **Invitation**: the link of an account that has no password yet (**New user** without a password). It works for **7 days**, since its owner may not see it at once.
- **Reset**: the link of an account that already has a password (**Reset password**, **Email a reset link**). It works for **1 hour**. Change that with `reset_ttl`, in minutes:

```toml
[security]
reset_ttl = 60
```

Using a reset link signs the user out everywhere and **deletes all their API tokens**, including the ones of connected MCP apps, so nobody who had access before the reset keeps it. They have to create their tokens again. Changing your own password in **Settings** signs out your other sessions but keeps your API tokens.

The token is part of the link's address. Pages send `Referrer-Policy: no-referrer`, so a browser does not pass the address on to other sites. A reverse proxy still logs the full address by default: leave `/set-password/` and `/api/auth/password/` out of its access log, or redact them, and keep the log private. A used or expired link is worthless, so this matters most for the hour after a reset link is sent.

### Sign-in rate limits

The routes where someone can guess a secret are throttled: sign-in, the second-factor step, passkey sign-in, first-run registration and its bootstrap password check, password reset links, invitation links and email confirmation links. Over the limit they answer `429` with a `Retry-After` header (seconds).

- `ip_rate_limit` (default `10`): attempts per minute from one client address, counted together over all these routes.
- `account_rate_limit` (default `5`): sign-in attempts per minute against one email address, from any address. The address is compared without case or surrounding spaces. Attempts count whether or not the password was right.

```toml
[security]
ip_rate_limit = 10
account_rate_limit = 5
```

`0` turns a limit off; a negative number stops the server at start. The counters live in memory and start again after a restart. The client address is the one the reverse proxy reports in `X-Forwarded-For`: behind a proxy that does not set it, every user shares the proxy's address and the first limit. See [Behind a reverse proxy](#behind-a-reverse-proxy).

Passwords are at least 8 characters everywhere (first admin, change, reset, invitation and admin-created users). Two-factor recovery codes are stored as SHA-256 hashes, so one wrong code costs the server a single lookup.

## Behind a reverse proxy

Set `app_url` to your public `https://` URL. The `X-Forwarded-For` header is honored, no other setup needed.

HTTPS is detected from `app_url`, not from request headers: with an `https://` URL the session cookie is `Secure` and named `__Host-svy_session`, and only that cookie signs anyone in (a plain `svy_session` cookie is ignored). `X-Forwarded-Proto` is not trusted, because anyone who can reach the server directly could set it. Without `app_url`, a request counts as HTTPS only when the connection itself is TLS. So `app_url` must be `https://` behind a TLS-terminating proxy, or cookies stay plain and unprotected.

Sign-in and the other public POSTs (login, 2FA, password links) refuse a browser request from another site: `Origin` must be the origin of `app_url` (the request's own host when it is unset), and the body must be `application/json`. A proxy that rewrites `Origin` breaks sign-in; scripts that send no `Origin` are unaffected.

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

## Server checks

Administration → Status lists what needs attention, and a dot on the Administration menu shows when something does. Each entry links to the page that fixes it. The list is worked out from the current state each time, so a fixed problem disappears by itself. Only server admins see it.

| Check | Level | When |
|-------|-------|------|
| Public URL is not set | warning | `app_url` is empty |
| Email is not set up | warning | the `[mailer]` section is off |
| Email settings are invalid | error | the `[mailer]` section failed validation at start, so mail is off |
| No server backup | warning | there is no valid server backup |
| Server backup is old | warning | the newest valid server backup is over 30 days old |
| Data disk is filling | warning at 75%, error at 90% | the volume that holds the uploads |
| Config file is read-only | info | the application does not change `config.toml` for now, so its values are edited in the file |

The same list is available to admins at `GET /api/admin/checks` (session only).

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

It runs as the `savvy-go` systemd service (`/usr/bin/savvy-go`). Data is in `/var/lib/savvy-go`; configuration is in `/etc/savvy-go/config.toml`, which the service creates and maintains itself (restart after editing). It listens on `127.0.0.1:8080` only; put a reverse proxy in front, or set `listen = ":8080"` under `[server]` to expose it. `apt purge savvy-go` deletes the data.

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
