# Development

## Backend (Go)

```bash
go generate ./internal/db   # sqlc: internal/db/queries → internal/db/sqlc
go test ./...
go run .
```

### Code shape checks

`internal/archtest` fails `go test ./...` on code that is clearly out of hand: a function over 500 lines, nesting deeper than 8, a name over 10 words, a line over 300 characters and similar. A justified exception goes in the doc comment: `//archtest:ignore funcLines <reason>`.

`mage test:audit` (or `go test -tags audit ./internal/archtest -run Audit -v` and `npm run audit`) prints everything above the softer limits, worst first. It never fails; use it as a worklist when cleaning up.

The frontend has the same two layers in `resources/ts/lib/code-shape` (`npm test` runs the hard one). Limits differ where the language does: a component may be 500 lines before it fails, and a test title is counted in words instead of an identifier. Opt out with `// archtest:ignore funcLines <reason>` above the declaration.

Listens on `localhost:8080`. Data and the generated `config.toml` go to `./savvy-data`, see [Data and backups](data-and-backups.md) and [Configuration](deployment.md#configuration). A plain `go build` or `go run` serves the frontend from `public/` on disk; release builds have it built into the binary, see [Build tasks](#build-tasks).

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

One Go process serves the API, the React frontend, the scheduler and workers. Server data is in `server.sqlite`, each space has its own SQLite file. Business logic uses the storage interface in `internal/store` and asks the repositories in `internal/db/repo/<area>` for rows, so it holds no SQL; SQLite code stays in `internal/store/sqlite`, and operations spanning two spaces use `store.InSpaces`. Migrations run on startup.

The HTTP layer in `internal/httpserver` is split by role: `api/<resource>` holds one thin handler package per resource (each with its own `routes.go` and `openapi.yaml`), `auth/` the sign-in, two-factor, OAuth and SSO flows, `admin/` what only server admins use (server backups, signing keys, spaces overview, users, mail, SSO providers, checks, monitoring) and `httpx/` what they share (JSON replies, request and scope helpers, `dto`, `guard`, `routing`). Handlers only adapt HTTP; the logic that scheduled jobs also need lives in the top-level `internal/` packages. A new handler package is linked in from `httpserver/modules.go`.

`config.toml` is owned by `internal/config`, the only package that knows the file. The builder (`internal/app/builder.go`) calls `config.Init(path)`, asks for each section with `config.Of(section)`, then calls `config.Finish()`, which reports unknown keys and writes the file. A package declares its sections next to the code that uses them, and can have several. A section is a `config.Section[Table, Value]`: `Table` is the TOML struct and `Decode` turns it into the value the package works with (units, trimming, checks). When `Table` implements `Defaults()`, the defaults fill the keys the file lacks and the section is always written. Without them, it is written only when the file has it, and a table that only holds subsections is never written on its own. `config.OfOptional` is for one of several alternatives, such as `[mail.smtp]`. A `*config.Ref` gives the decoded value with `Get`. `Subscribe` is there for the rare package that must react to a change. Tests use `config.Fixed(value)`.

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
go tool mage probe:oauth     # OAuth sign-in against a running server (OAUTH_SERVER, see mcp.md)
go tool mage generate        # sqlc
go tool mage release         # everything a release ships, into dist/ (needs APP_VERSION and nfpm)
```

`build:release` honors `APP_VERSION`, `APP_ENV` (default `production`), `TARGET_GOOS` and `TARGET_GOARCH`. The target falls back to `GOOS` and `GOARCH`, but those also change how Mage itself is compiled, so cross-building from Windows or macOS needs `TARGET_*`. It copies `public/` to `internal/httpserver/ui/dist` (git-ignored) and builds with the `embed` tag; without the tag the Go build never touches the frontend, so `go test ./...` needs no frontend build.

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
