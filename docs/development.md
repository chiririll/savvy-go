# Development

## Backend (Go)

```bash
go generate ./internal/db   # sqlc: internal/db/queries → internal/db/sqlc
go test ./...
go run ./cmd/savvy-go
```

Listens on `localhost:8080`. Data and the generated `config.toml` go to `./data`, see [Data and backups](data-and-backups.md) and [Configuration](deployment.md#configuration). The frontend is served from `public/`.

With `--seed-config <file>` (any TOML file, even an empty one), the first boot creates demo users, three spaces with ~12 months of data, linked spaces and open invitations:

| Email              | Password   | Server role |
|--------------------|------------|-------------|
| `admin@savvy.app`  | `password` | admin       |
| `editor@savvy.app` | `password` | user        |
| `guest@savvy.app`  | `password` | guest       |
| `demo@demo.com`    | `demo`     | user        |

Roles differ per space, so each user sees a different view.

In the seed config file, `date` (`YYYY-MM-DD`) places the data relative to that day instead of today. `manifest` is a file the seed writes users, spaces and invitation tokens to; [Scripts](scripts.md) uses it.

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

## Build tasks

[Mage](https://magefile.org) runs the builds from `magefile.go`, pinned in `go.mod` so nothing is installed globally. `go tool mage -l` lists the targets:

```bash
go tool mage build:release   # frontend + dist/savvy-go with the frontend embedded
go tool mage build:dev       # dist/savvy-go that serves public/ from disk
go tool mage build:embed     # release without rebuilding the frontend
go tool mage test:all        # Go (with and without the embed tag) and frontend tests
```

`build:release` honors `APP_VERSION`, `APP_ENV` (default `production`), `GOOS` and `GOARCH`. It copies `public/` to `internal/webui/dist` (git-ignored) and builds with the `embed` tag; without the tag the Go build never touches the frontend, so `go test ./...` needs no frontend build. `paths.public` in `config.toml` overrides the built-in copy.

Logo and screenshot scripts: [Scripts](scripts.md).

AI agent setup: [Agentic development](agentic-development.md).

## Workflows

In `.github/workflows`. All of them can also be started manually.

### Testing

Run on PRs and pushes to `main`, only when relevant files change.

- `ci-go`: backend tests.
- `ci-frontend`: frontend type checking and tests.

### Release

Usually run on a `v*` tag push.

#### dist

Builds the Linux packages (`.deb`, `.rpm`) and publishes a GitHub release.

#### docker

Builds and pushes the Docker Hub image. Also runs on every push to `main`.

Skipped while the `DOCKER_IMAGE` variable is unset, so forks don't fail. To enable, set under Settings → Secrets and variables → Actions:

- variable `DOCKER_IMAGE`: Docker Hub repository, e.g. `chiririll/savvy-go`
- secret `DOCKERHUB_USERNAME`: Docker Hub username
- secret `DOCKERHUB_TOKEN`: Docker Hub API token
