# Development

## Backend (Go)

```bash
go generate ./internal/db   # sqlc: internal/db/queries → internal/db/sqlc
go test ./...
go run .
```

`go run .` listens on `localhost:8080`. Data and the generated `config.toml` go to `./savvy-data`, see [Data and backups](data-and-backups.md) and [Configuration](deployment.md#configuration). A plain `go build` or `go run` serves the frontend from `public/` on disk; release builds embed it, see [Build tasks](#build-tasks).

### Code shape checks

`internal/archtest` fails `go test ./...` on code that is clearly out of hand: a function over 200 lines, nesting deeper than 8, a name over 8 words, a line over 160 characters and similar. A justified exception goes in the doc comment: `//archtest:ignore funcLines <reason>`.

`mage test:audit` (or `go test -tags audit ./internal/archtest -run Audit -v`, `npm run audit`) prints everything above the softer limits, worst first. It never fails; use it as a cleanup worklist.

The frontend has the same two layers in `resources/ts/lib/code-shape` (`npm test` runs the hard one). Limits differ where the language does: a component may be 500 lines, and test titles count words. Opt out with `// archtest:ignore funcLines <reason>` above the declaration.

### Demo data

With `--seed-config <file>` (any TOML file, even empty), the first boot creates demo users, three spaces with ~12 months of data, linked spaces and open invitations:

| Email              | Password   | Server role |
|--------------------|------------|-------------|
| `admin@savvy.app`  | `password` | admin       |
| `editor@savvy.app` | `password` | user        |
| `guest@savvy.app`  | `password` | guest       |
| `demo@demo.com`    | `demo`     | user        |

Roles differ per space, so each user sees a different view. Every space has its own emoji and color. Family Budget and Demo are in one sync group, each with a category and a tag the other lacked. Studio GmbH is outside it and has a set of categories and tags of its own (clients, retainers, VAT) instead of the presets. About a dozen transfers cross the links between the spaces, some in the last few days and some between currencies. All addresses are verified except the guest's, which shows as unverified once mail is set up.

Example `seed.toml`:

```toml
# All keys are optional; an empty file seeds with the defaults.

# Set to false to keep the file but skip seeding. Default: true.
enabled = true

# Place the data relative to this day (YYYY-MM-DD) instead of today.
date = "2026-06-05"

# Write the seeded users, spaces and invitation tokens to this file.
manifest = "./savvy-data/manifest.json"
```

```bash
go run . --seed-config seed.toml
```

`enabled` (default `true`) turns seeding off without removing the flag. `date` places the data relative to that day instead of today. `manifest` is a file the seed writes users (with whether their address is verified), spaces (emoji, color, sync group) and invitation tokens to; [Scripts](scripts.md) uses it. Unknown keys are an error. Seeding only happens on an empty database: if the server already has a user, or was seeded before, the flag does nothing, so start from a fresh data directory.

## Architecture

One Go process serves the API, the React frontend, the scheduler and workers. Server data is in `server.sqlite`, each space has its own SQLite file. Business logic uses the storage interface in `internal/store` and asks the repositories in `internal/db/repo/<area>` for rows, so it holds no SQL; SQLite code stays in `internal/store/sqlite`, and operations spanning two spaces use `store.InSpaces`. Migrations run on startup.

The HTTP layer in `internal/httpserver` is split by role: `api/<resource>` holds one thin handler package per resource (each with its own `routes.go` and `openapi.yaml`), `auth/` the sign-in, two-factor, OAuth and SSO flows, `admin/` what only server admins use (server backups, signing keys, spaces overview, users, mail, SSO providers, checks, monitoring) and `httpx/` what they share (JSON replies, request and scope helpers, `dto`, `guard`, `routing`). Handlers only adapt HTTP; the logic that scheduled jobs also need lives in the top-level `internal/` packages. A new handler package is linked in from `httpserver/modules*.go`.

`config.toml` is owned by `internal/config`, the only package that knows the file. The builder (`internal/app/builder.go`) calls `config.Init(path)`, asks for each section with `config.Of(section)`, then calls `config.Finish()`, which reports unknown keys and writes the file. A package declares its sections next to the code that uses them, and can have several. A section is a `config.Section[Table, Value]`: `Table` is the TOML struct and `Decode` turns it into the value the package works with (units, trimming, checks). When `Table` implements `Defaults()`, the defaults fill the keys the file lacks and the section is always written. Without them, it is written only when the file has it, and a table that only holds subsections is never written on its own. `config.OfOptional` is for one of several alternatives, such as `[mail.smtp]`. A `*config.Ref` gives the decoded value with `Get`. `Subscribe` is there for the rare package that must react to a change. Tests use `config.Fixed(value)`.

## Frontend

React, Vite, react-query, ShadCN/UI. Source in `resources/ts`; run from the repo root:

```bash
npm install
npm run dev     # Vite dev server
npx tsc --noEmit
npm run build   # into public/build
```

### Frontend tests

```bash
npm test                  # unit tests (Vitest, no browser)
npm run test:stories      # every Storybook story, as a test in Chromium (English, light theme)
npm run test:stories:full # the same stories again in every other language and in the dark theme
npm run storybook         # the stories in the browser, with a language and theme switch
```

Components are tested through Storybook (`*.stories.tsx` next to the component). A story is one state of a component: loading, empty, error, read-only, a currency with 0 or 3 decimals. `npm run test:stories` runs each story twice, at a desktop size (1280×800, project `stories-desktop`) and a phone size (390×844, project `stories-mobile`), so a responsive break fails the same way a logic bug does. Story tags narrow that: `tags: ['mobile-only']` skips the desktop run, `tags: ['desktop-only']` (on `meta`, for a component with no responsive classes or `use-mobile` in its tree, or on a story that uses a control the phone layout lacks, like a row's "more actions" button) skips the phone run. Untagged stories run at both sizes. Tag names and the projects live in `.storybook/tags.ts` and `vite.config.ts`. A story with a `play` function also clicks and types, and asserts. Every story is checked for accessibility (axe) as well, and a violation fails it.

The first run needs a browser: `npx playwright install chromium`.

#### Language and theme matrix

The language and the theme are Storybook globals (`locale`, `theme`; the toolbar switches them in `npm run storybook`), not separate stories. Do not write an `InRussian` or `InDarkTheme` story: the matrix runs every story in them.

| Run | Projects | Language, theme | Size |
| --- | --- | --- | --- |
| `npm run test:stories` (local, job `frontend` in `ci-frontend.yaml`) | `stories-desktop`, `stories-mobile` | English, light | desktop, phone |
| `npm run test:stories:full` (`ci-frontend.yaml`, job `stories-matrix`) | the two above, plus `stories-mobile-<locale>` for every other language and `stories-desktop-dark` | each project sets its own | phone for a language (longest text, narrowest screen), desktop for dark |

A project picks its globals from environment variables set in `vite.config.ts` (`STORYBOOK_LOCALE`, `STORYBOOK_THEME`), which `initialGlobals` in `.storybook/preview.tsx` reads. A story can still set `globals: { locale: 'ru' }` itself, for a check that is about Russian (number format, a Russian label), like `ApplicationInRussian` in `Layout.stories.tsx`.

In a project that is not English and light, `.storybook/matrix-checks.ts` fails a story that:

- shows a translation key (`forms:save`) or calls `t()` with a key the language lacks (English would fill it in silently),
- scrolls the page sideways (`documentElement.scrollWidth > clientWidth`; the message names the widest elements).

The language projects leave out the `play` function (it finds elements by English text); the English run covers it. The dark project runs everything, play included.

To add a language: add the folder `resources/ts/locales/<code>/` with the same files as `en`, and add the code to `SUPPORTED_LOCALES` and `LOCALE_LABELS` in `resources/ts/lib/i18n.ts`. The matrix projects come from the folder names, so there is no story and no test config to change. `lib/i18n-keys.test.ts` checks that the files match `en`.

The browser size is not taken from the project: addon-vitest resets it before each story to the story's `viewport` global, so `vite.config.ts` also sets `STORYBOOK_VIEWPORT` and the preview has a `desktop` and a `mobile` viewport option.

Coverage of the stories: `npx vitest run --project stories-desktop --project stories-mobile --coverage.enabled --coverage.provider=v8` (needs `@vitest/coverage-v8`, not a dependency). Which files it counts is set under `coverage` in `vite.config.ts`.

- **Data.** Components read the API through the real client; a story answers it with MSW (`resources/ts/test/api-mocks.ts`: `reply`, `fail`, `hang` for a request that never ends). A request a story did not mock fails the story. Sample data is in `resources/ts/test/fixtures.ts`.
- **Forms.** Components built on the ShadCN form parts need a form around them: `decorators: [withForm({ label: 'Currency' })]` from `resources/ts/test/form-decorator.tsx`.
- **Menus and selects.** Radix hides the page behind an open menu, which axe reports. Add `parameters: { a11y: overlayA11y }` from `resources/ts/test/a11y.ts`.
- **Colour contrast is not checked yet** (`resources/ts/test/a11y.ts` has the reason and a TODO). The palette misses WCAG AA in a few places; the check goes back on once it passes. To see the misses in the dark theme: `STORYBOOK_CONTRAST=on npx vitest run --project stories-desktop-dark`.
- **Status colours are tokens.** Use `text-destructive`, `text-success`, `text-warning`, `text-info` and tints like `bg-success/10` (defined per theme in `resources/ts/index.css`, checked against WCAG AA), not `text-red-600` plus a `dark:` variant. Do not dim text with `opacity-*`: use `text-muted-foreground`, or `aria-disabled` for a disabled control.
- **Language and theme.** See "Language and theme matrix" above.
- **Animations are off**, so a check never sees an element half faded in.

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
- `ci-frontend`: frontend type checking, unit tests and story tests.

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
