# Development

## Backend (Go)

```bash
go generate ./internal/db   # sqlc: internal/db/queries → internal/db/sqlc
go test ./...
go run ./cmd/savvy-go
```

Listens on `:8080` (`LISTEN_ADDR`). Data goes to `./data` (`DATA_DIR`), see [Data and backups](data-and-backups.md). The frontend is served from `public/` (`PUBLIC_DIR`).

With `SEED_DEMO=true`, the first boot creates demo users, three spaces with ~12 months of data, linked spaces and open invitations:

| Email              | Password   | Server role |
|--------------------|------------|-------------|
| `admin@savvy.app`  | `password` | admin       |
| `editor@savvy.app` | `password` | user        |
| `guest@savvy.app`  | `password` | guest       |
| `demo@demo.com`    | `demo`     | user        |

Roles differ per space, so each user sees a different view.

`SEED_DATE` (`YYYY-MM-DD`) places the data relative to that day instead of today. `SEED_MANIFEST` is a file the seed writes users, spaces and invitation tokens to; [Scripts](scripts.md) uses it.

## Architecture

One Go process serves the API, the React frontend, the scheduler and workers. Server data is in `server.sqlite`, each space has its own SQLite file. Business logic uses the storage interface in `internal/store`; SQLite code stays in `internal/store/sqlite`, and operations spanning two spaces use `store.InSpaces`. Migrations run on startup.

## Frontend

React, Vite, react-query, ShadCN/UI. Source in `resources/ts`; run from the repo root:

```bash
npm install
npm run dev     # Vite dev server
npx tsc --noEmit
npm run build   # into public/build
```

Logo and screenshot scripts: [Scripts](scripts.md).
