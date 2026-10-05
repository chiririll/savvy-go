# Development

## Backend (Go)

```bash
go generate ./internal/db   # sqlc: internal/db/queries → internal/db/sqlc
go test ./...
go run ./cmd/savvy-go
```

It listens on `:8080` by default (`LISTEN_ADDR`). State goes under `DATA_DIR` (`./data` locally); see [Data and backups](data-and-backups.md). The SPA is served from `public/` (Vite output in `public/build`). Env: `APP_URL`, `TZ`, `DATA_DIR`, `LISTEN_ADDR`, `PUBLIC_DIR`, `SEED_DEMO`.

With `SEED_DEMO=true` the first boot creates demo users and three spaces with ~12 months of data, linked spaces and open invitations:

| Email              | Password   | Server role |
|--------------------|------------|-------------|
| `admin@savvy.app`  | `password` | admin       |
| `editor@savvy.app` | `password` | user        |
| `guest@savvy.app`  | `password` | guest       |
| `demo@demo.com`    | `demo`     | user        |

Their roles in the spaces differ, so each one shows a different view of the app.

## How it is built

One Go process serves the HTTP API, the React SPA, the scheduler and background workers. Server-wide data is in `server.sqlite` and each space has its own SQLite file. The business logic only sees a storage interface (`internal/store`); everything SQLite-specific stays in `internal/store/sqlite`, and work that touches two spaces at once goes through `store.InSpaces`. Schema migrations run on startup for the server and every space.

## Frontend

React with Vite, react-query and ShadCN/UI. The source is in `resources/ts`; run the commands from the repository root:

```bash
npm install
npm run dev     # Vite dev server
npx tsc --noEmit
npm run build   # into public/build
```

Helper scripts for static assets are described in [Scripts](scripts.md).
