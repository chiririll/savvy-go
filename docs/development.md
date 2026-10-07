# Development

## Backend (Go)

```bash
go generate ./internal/db   # sqlc: internal/db/queries → internal/db/sqlc
go test ./...
go run ./cmd/savvy-go
```

Listens on `localhost:8080`. Data and the generated `config.toml` go to `./data`, see [Data and backups](data-and-backups.md) and [Configuration](deployment.md#configuration). A plain `go build` or `go run` serves the frontend from `public/` on disk; release builds have it built into the binary, see [Build tasks](#build-tasks).

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
go tool mage generate        # sqlc
go tool mage release         # everything a release ships, into dist/ (needs APP_VERSION and nfpm)
```

`build:release` honors `APP_VERSION`, `APP_ENV` (default `production`), `TARGET_GOOS` and `TARGET_GOARCH`. The target falls back to `GOOS` and `GOARCH`, but those also change how Mage itself is compiled, so cross-building from Windows or macOS needs `TARGET_*`. It copies `public/` to `internal/webui/dist` (git-ignored) and builds with the `embed` tag; without the tag the Go build never touches the frontend, so `go test ./...` needs no frontend build. `paths.public` in `config.toml` overrides the built-in copy.

Logo and screenshot scripts: [Scripts](scripts.md).

AI agent setup: [Agentic development](agentic-development.md).

## Workflows

In `.github/workflows`. All of them can also be started manually.

### Testing

Run on PRs and pushes to `main`, only when relevant files change.

- `ci-go`: backend tests.
- `ci-frontend`: frontend type checking and tests.

### Package check

`ci-packages` runs it on every push to `main` (not on tags). Run it by hand after changing anything in `deploy/`. Needs Docker, Go and Node; works the same on Linux, macOS and Windows:

```sh
go tool mage test:packages     # deb and rpm
go tool mage test:deb          # one format (test:rpm)
SKIP_FRONTEND=1 go tool mage test:packages   # reuse public/build (set the variable your shell's way on Windows)
```

It builds the Linux binary with the frontend embedded on the host, runs the `package:deb` and `package:rpm` targets (a compiled Mage, so no Go is needed) in a container that has nfpm, then installs the package in a Debian (deb) or Rocky Linux (rpm) container with systemd as PID 1, then checks install, health endpoints, the service user, crash restart, upgrade and removal (plus purge for deb). The container is privileged, so only run it on a machine you trust.

### Release

Usually run on a `v*` tag push.

#### dist

Runs `mage release`: builds the Linux packages (`.deb`, `.rpm`), the standalone Linux and Windows binaries and `SHA256SUMS`, and publishes a GitHub release. The frontend is built once and shared by all of them. To run it locally, install [nfpm](https://nfpm.goreleaser.com) and set `APP_VERSION`.

#### docker

Builds and pushes the Docker Hub image. Also runs on every push to `main`.

Skipped while the `DOCKER_IMAGE` variable is unset, so forks don't fail. To enable, set under Settings → Secrets and variables → Actions:

- variable `DOCKER_IMAGE`: Docker Hub repository, e.g. `chiririll/savvy-go`
- secret `DOCKERHUB_USERNAME`: Docker Hub username
- secret `DOCKERHUB_TOKEN`: Docker Hub API token
