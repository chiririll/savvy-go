<p align="center">
  <img src="docs/images/logo-dark.svg#gh-light-mode-only" alt="Go Savvy" width="120">
  <img src="docs/images/logo-light.svg#gh-dark-mode-only" alt="Go Savvy" width="120">
</p>

<h1 align="center">Go Savvy</h1>

<p align="center">
  Selfhosted expense tracker with full multi-currency support and shared spaces. Re-written in go.
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
docker run -d -p 3000:80 -v savvy-go-data:/data chiririll/savvy-go
```

Open http://localhost:3000 and create your account. For Docker Compose, reverse proxies and the Debian package, see [Deployment](docs/deployment.md).

## ✨ Features

- **Multi-currency** — any fiat or crypto, transfers between them, rates updated automatically
- **Spaces** — separate books for yourself, your family or a project, each with its own accounts, categories and base currency
- **Sharing** — invite people to a space as admin, editor or viewer
- **Transfers between spaces** — send money from one linked space to another, each side in its own currency
- **Recurring transactions** — scheduled payments (daily, weekly, monthly, yearly)
- **Automation rules** — auto-categorize transactions based on conditions
- **Budgets and debts** — limits with progress, loans and borrowings with payment history
- **Categories & tags** — flexible organization
- **Rich analytics** — Sankey diagrams, heatmaps, net worth tracking, expense pace
- **CSV import** — import transactions from bank exports with duplicate detection
- **Backups** — signed backups of each space and of the whole server, restore or import as a new space
- **Sign-in** — passwords, two-factor (TOTP), passkeys and single sign-on (OIDC)
- **API** — tokens for scripts and other apps, with an OpenAPI reference

<p align="center">
  <img src="docs/images/report.png" alt="Go Savvy Reports" width="1920">
</p>

## 📱 Mobile-Friendly

Fully responsive design built with ShadCN/UI — track expenses from your phone right after purchase.

<p align="center">
  <img src="docs/images/mobile.png" alt="Mobile Dashboard" width="1920">
</p>

## 📚 Documentation

- [Spaces and roles](docs/spaces.md) — spaces, server and space roles, invitations, linked spaces and transfers
- [Deployment](docs/deployment.md) — Docker Compose, environment, reverse proxies, health checks, Kubernetes, Debian, updating
- [Data and backups](docs/data-and-backups.md) — the data directory, the signing key, backups and upgrading from the Laravel version
- [API](docs/api.md) — tokens, endpoints and the OpenAPI reference
- [Development](docs/development.md) — running from source and how the app is built

## 🔒 Privacy

Your data stays with you: SQLite databases in one data volume, no external services required. Server admins see only the names, sizes and members of other people's spaces, never their finances, and their actions on a space are logged for that space's admins.

## 🛠 Stack

Go • SQLite • React (Vite) • Docker • ShadCN/UI • Tailwind CSS

## 📄 License

[MIT](LICENSE)

---

<p align="center">
  Made with ❤️ for people who want control over their finances
</p>
