<p align="center">
  <img src="docs/images/logo-dark.svg#gh-light-mode-only" alt="Go Savvy" width="120">
  <img src="docs/images/logo-light.svg#gh-dark-mode-only" alt="Go Savvy" width="120">
</p>

<h1 align="center">Go Savvy</h1>

<p align="center">
  Selfhosted expense tracker with full multi-currency support. Re-written in go.
</p>

<p align="center">
<a href="https://hub.docker.com/r/chiririll/savvy-go"><img src="https://img.shields.io/badge/DOCKER-chiririll/savvy-go-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker"></a>
<img src="https://img.shields.io/github/v/tag/chiririll/savvy-go?style=for-the-badge&color=orange" alt="Version">
<img src="https://img.shields.io/badge/LICENSE-MIT-green?style=for-the-badge" alt="License">
</p>

---

<p align="center">
  <img src="docs/images/screenshot.png" alt="Go Savvy Screenshot" width="1920">
</p>

## ⚡ Quick Start
```bash
docker run -d -p 3000:80 -v savvy-data:/data chiririll/savvy-go:latest
```

Open `localhost:3000` and create your account.

## ✨ Features

- **Multi-currency** — any fiat or crypto, transfers between them
- **Auto exchange rates** — currency rates updated automatically via API
- **Recurring transactions** — scheduled payments (daily, weekly, monthly, yearly)
- **Automation rules** — auto-categorize transactions based on conditions
- **Debts** — track loans and borrowings with payment history
- **Budgets** — set limits and track progress
- **Categories & tags** — flexible organization
- **Multi-user** — share with family or team, role-based access (admin/user)
- **Rich analytics** — Sankey diagrams, heatmaps, net worth tracking, expense pace
- **CSV import** — import transactions from bank exports with duplicate detection
- **Backups** — create, restore and download database backups
- **2FA** — two-factor authentication via TOTP (Google Authenticator, etc.)

<p align="center">
  <img src="docs/images/report.png" alt="Go Savvy Reports" width="1920">
</p>

## 📱 Mobile-Friendly

Fully responsive design built with ShadCN/UI — track expenses from your phone right after purchase.

<p align="center">
  <img src="docs/images/mobile.png" alt="Mobile Dashboard" width="1920">
  &nbsp;&nbsp;&nbsp;
</p>

## 🚀 Deployment

### Docker Compose (Recommended)
```yaml
services:
  savvy:
    image: chiririll/savvy-go:latest
    container_name: savvy
    restart: unless-stopped
    ports:
      - "3000:80"
    volumes:
      - savvy-data:/data
    environment:
      - APP_URL=https://savvy.yourdomain.com
      - TZ=Europe/Kyiv
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://127.0.0.1/livez"]
      interval: 10s
      timeout: 3s
      start_period: 60s
      retries: 3

volumes:
  savvy-data:
```

### Environment Variables

| Variable    | Description                                                                 | Default            |
|-------------|-----------------------------------------------------------------------------|--------------------|
| `APP_URL`   | Public URL of your instance                                                 | `http://localhost` |
| `TZ`        | Timezone                                                                    | `UTC`              |
| `SEED_DEMO` | First boot only: seed demo users, accounts, and ~12 months of transactions | `false`            |

### Behind a Reverse Proxy

Set `APP_URL` to your public `https://` URL. Savvy honors the `X-Forwarded-Proto` and `X-Forwarded-For` headers from the proxy, so HTTPS link generation and real client IPs work automatically — no extra configuration needed.

### Health Checks

Two probe endpoints are exposed for orchestrators and uptime monitoring (responses use the IETF `application/health+json` format):

| Endpoint  | Purpose                                                                     | Healthy | Unhealthy |
|-----------|-----------------------------------------------------------------------------|---------|-----------|
| `/livez`  | Liveness — the app process is up. Use it for container restart decisions.   | `200`   | —         |
| `/readyz` | Readiness — database is reachable and migrations are applied. Gate traffic. | `200`   | `503`     |

`/livez` stays up during maintenance mode; `/readyz` returns `503` so traffic drains while the instance is not ready.

### With Traefik (HTTPS)
```yaml
services:
  savvy:
    image: chiririll/savvy-go:latest
    container_name: savvy
    restart: unless-stopped
    volumes:
      - savvy-data:/data
    environment:
      - APP_URL=https://savvy.yourdomain.com
      - TZ=Europe/Kyiv
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.savvy.rule=Host(`savvy.yourdomain.com`)"
      - "traefik.http.routers.savvy.entrypoints=websecure"
      - "traefik.http.routers.savvy.tls.certresolver=letsencrypt"
      - "traefik.http.services.savvy.loadbalancer.server.port=80"
    networks:
      - traefik

volumes:
  savvy-data:

networks:
  traefik:
    external: true
```

### With Nginx Proxy Manager

1. Run Savvy on internal port:
```yaml
services:
  savvy:
    image: chiririll/savvy-go:latest
    container_name: savvy
    restart: unless-stopped
    expose:
      - "80"
    volumes:
      - savvy-data:/data
    environment:
      - APP_URL=https://savvy.yourdomain.com
    networks:
      - npm-network

volumes:
  savvy-data:

networks:
  npm-network:
    external: true
```

2. In Nginx Proxy Manager, create proxy host pointing to `savvy:80`

### Kubernetes

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

### Debian package

On Debian 13 (Trixie) or later, install the `.deb` from the GitHub release:

```bash
curl -fsSLO https://github.com/chiririll/savvy-go/releases/latest/download/savvy-go.deb
sudo apt install ./savvy-go.deb
```

The binary is installed to `/usr/bin/savvy-go` and runs as the `savvy-go` systemd service. Data lives in `/var/lib/savvy-go`. Optional settings (`APP_URL`, `TZ`) go in `/etc/savvy-go/install.env`. `apt purge savvy-go` removes the data directory.

### Tarball

Release `savvy-go.tar.gz` contains the `savvy-go` binary and the static SPA assets (`public/`). On any Linux host:

```bash
mkdir -p /opt/savvy-go && tar -C /opt/savvy-go -xzf savvy-go.tar.gz
DATA_DIR=/var/lib/savvy-go PUBLIC_DIR=/opt/savvy-go/public LISTEN_ADDR=:8080 APP_URL=https://savvy.example.com /opt/savvy-go/savvy-go
```

Point a reverse proxy at `:8080`.

## 🔄 Updating
```bash
docker compose pull
docker compose up -d
```

Your data is safe in the `/data` volume.

Debian: install the newer `.deb`. Data stays in `/var/lib/savvy-go`.

## 💾 Backups

Backups can be managed directly from the UI (Settings → Backups).

> [!WARNING]
> The database runs in **WAL mode**, so recent writes may still live in the `database.sqlite-wal` file and **won't be in `database.sqlite` yet**. Copying `database.sqlite` alone can silently lose the latest data. Always checkpoint the WAL into the main file first (or use the in-app backup, which handles this for you).

Manual backup (stop the container first so the WAL is flushed):
```bash
docker compose down
docker cp savvy:/data/database.sqlite ./backup-$(date +%Y%m%d).sqlite
```

Debian package:
```bash
sqlite3 /var/lib/savvy-go/database.sqlite ".backup savvy-go-$(date +%Y%m%d).sqlite"
```

Restore (stop writers first so the WAL doesn't fight the swap):
```bash
docker compose down
docker cp ./backup.sqlite savvy:/data/database.sqlite
docker compose up -d
```

## 🔌 API

Connect scripts and external apps with an API token.

1. Open **Settings → API** and create a token (name, `read` or `read-write` access, optional expiration). The token is shown once.
2. Send it as a bearer token:

```bash
curl -H "Authorization: Bearer svy_xxxxxxxx" https://your-savvy.example/api/accounts

curl -X POST https://your-savvy.example/api/tags \
  -H "Authorization: Bearer svy_xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"name": "from-script"}'
```

A token acts as its owner, capped by the owner's role. `read` tokens can only make `GET` requests. Tokens cannot manage users, identity providers, passwords, 2FA, passkeys, API tokens, settings or backups. Revoke a token in the same screen to cut an app off immediately.

Interactive reference (Swagger UI): `/api/docs`. Raw OpenAPI 3 spec: `/api/openapi.yaml`.

## 🔒 Privacy

Your data stays with you. SQLite database stored in `/data` volume — no external services required.

## ⚙️ How It Works

One Go process serves the HTTP API, the React SPA, the scheduler and background workers. SQLite lives in `/data`. Schema migrations run automatically on startup.

The Debian package ships the same binary with a systemd unit and SQLite in `/var/lib/savvy-go`.

## 🛠 Stack

Go • SQLite • React (Vite) • Docker • ShadCN/UI • Tailwind CSS

### Local backend (Go)

```bash
go generate ./internal/db   # sqlc: internal/db/queries → internal/db/sqlc
go test ./...
go run ./cmd/savvy-go
```

Listens on `:8080` by default (`LISTEN_ADDR`). SQLite and uploads go under `DATA_DIR` (`./data` locally, `/data` or `/var/lib/savvy-go` in deploy). The SPA is served from `public/` (Vite output in `public/build`). Env: `APP_URL`, `TZ`, `DATA_DIR`, `LISTEN_ADDR`, `SEED_DEMO`.

## 🤝 Contributing

Contributions are welcome! Please open an issue first to discuss what you would like to change.

## 📄 License

[MIT](LICENSE)

---

<p align="center">
  Made with ❤️ for people who want control over their finances
</p>
