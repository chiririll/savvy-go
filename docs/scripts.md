# Scripts

Helper scripts live in [`scripts/`](../scripts) with their own [`package.json`](../scripts/package.json), so they stay out of the app and the Docker build. Install once:

```bash
npm --prefix scripts install
```

Root `npm run <name>` commands are aliases for `npm --prefix scripts run <name>`.

## Docker or local

`npm run screenshots` and `npm run stories` ([development](development.md#frontend-tests)) run in Docker by default, so you need nothing but Node and Docker. Pass `--local` to run on this machine, or `--docker` to say it outright:

```bash
npm run screenshots -- --local full
npm run stories -- --local Foo.stories.tsx
```

The flag wins over `SAVVY_RUNNER=docker|local`, which wins over the default (docker). Every other argument passes through unchanged.

Docker mode calls plain `docker`, so it uses your active context (`docker context use ...` or `DOCKER_HOST`) and never picks one itself. Nothing is bind-mounted: sources go up as the build context and results are copied back, so any daemon works, local or remote. Expect the repo upload on every run.

`logos` and `screenshots:diff` run locally only.

Tests of the scripts themselves (the flag, the run plans): `npm run test:scripts`. They are not part of `npm test` or CI, which install only the root packages.

## Logos

```bash
npm run logos
```

Regenerates every logo from the `Logo` component ([`Logo.tsx`](../resources/ts/components/shared/appearance/Logo.tsx)): favicons and app icons in `public/`, README logos in `docs/images/`. Edit the component, then run it. Needs the root `npm install` too.

## Screenshots

```bash
npm run screenshots              # defaults: English, light theme
npm run screenshots -- full      # use configs/full.json
npm run screenshots -- --concurrency=8   # browser contexts at a time
npm run screenshots -- --local           # on this machine
```

Screenshots every page for each demo user, space and viewport (desktop and mobile).

In Docker (default), [`docker-compose.screenshots.yml`](../docker-compose.screenshots.yml) builds the app, seeds demo data into an empty volume and runs Playwright against it. Config files are passed as environment variables, and the result is copied back with `docker compose cp`. When the run ends (even on failure or Ctrl+C) containers and volume are removed, so every run starts fresh.

With `--local` ([`local.mts`](../scripts/screenshots/local.mts)), the same happens on this machine in a temporary directory: the frontend is built with `APP_ENV=screenshots`, the app with `go build`, and the app is seeded on empty data on a free port. `public/build` and `dist/` are left alone, and the directory is removed afterwards. Needs Go and the browser of `scripts/`: `npm --prefix scripts ci && npx --prefix scripts playwright install chromium`.

Pages and scenarios run at the same time and share the browser contexts (half each; all to whichever runs alone). The number is the CPU count of the Docker host (or this machine with `--local`) unless `--concurrency=N` is given. Pages all hit one seeded app, so a very high number can slow it down instead of the run.

Output goes to `screenshots/<start time>[_<config>]/` (git-ignored). Set `RUN_NAME` to name a run, e.g. `RUN_NAME=before-redesign`.

```
<language>-<theme>/<user>/<space>/<desktop|mobile>/<page>.png
<language>-<theme>/public/       # pages without a space (sign-in, invitations)
```

### Comparing runs

```bash
npm run screenshots:diff -- before-redesign latest   # <old> <new>; "latest" is the newest run
```

Writes `screenshots/<old>_vs_<new>/`: changed screenshots from the new run with changes boxed in red, plus `report.txt` (changed areas per file, files only in one run). Small color differences and specks under 12 px are ignored. Compare runs of the same day and config, or dates and amounts show up as changes (the script warns).

### Configs

A config is a JSON file in [`scripts/screenshots/configs/`](../scripts/screenshots/configs): `full` (English and Russian, both themes), `mobile`, `reports`. Without a name, defaults from [`config.ts`](../scripts/screenshots/config.ts) apply.

```json
{
    "users": ["alex", "jordan"],
    "spaces": ["Demo"],
    "viewports": ["mobile"],
    "languages": ["en", "ru"],
    "themes": ["light", "dark"],
    "pages": ["/transactions", "/settings"],
    "dialogs": true,
    "tabs": true
}
```

| Setting | Meaning | If omitted |
|---------|---------|------------|
| `users` | Seeded users: `alex`, `jordan`, `sam`, `demo` | all |
| `spaces` | Space names, e.g. `Demo`, `Family Budget` | all |
| `viewports` | `desktop`, `mobile` | both |
| `languages` | Folders in `resources/ts/locales` | `en` |
| `themes` | `light`, `dark` | `light` |
| `pages` | Paths from [`pages.ts`](../resources/ts/app/pages.ts), with subpages (`/settings` = all settings pages) | all |
| `sidebar` | Sidebar drawer on mobile | `true` |
| `dialogs` | Dialogs of each page | `true` |
| `tabs` | Tabs of each page and dialog | `true` |
| `controls` | Chart types, grouping and report periods | `false` |
| `capture` | Pages of the seeded app (needs its manifest) | `true` |
| `scenarios` | The [scenarios](#scenarios) | `true` |
| `scenarioNames` | Scenarios by name or prefix (`"onboarding/"`) | all |
| `scenarioLanguages`, `scenarioThemes`, `scenarioViewports` | What scenarios vary over | `languages`, `themes`, `viewports` |
| `detail` | `key`: essential shot of each step; `full`: also error states and variants | `full` |

The number of screenshots is the product of the lists (two languages × two themes = four times the files). Invalid values stop the run.

Any setting can also be passed on the command line, on top of a config or without one: `--name=a,b` for a list, `--name=true|false` for a flag.

```bash
npm run screenshots -- scenarios --scenarioLanguages=en,ru --scenarioThemes=light,dark
npm run screenshots -- --capture=false --scenarioNames=onboarding/ --detail=key
```

### Dialogs, tabs, controls

The script finds what to click by `data-testid` marks, added only in builds with `APP_ENV=screenshots`. Helpers: [`test-id.ts`](../resources/ts/lib/test-id.ts).

- **Dialogs**: put `{...testId('some-id')}` on the element that opens one. Shots go to `dialogs/`. Escape closes each, so nothing is confirmed. A dialog reachable several ways is shot once per run.
- **Tabs**: Radix tabs are found automatically; `SegmentedChoice` carries its own marks. Each unselected tab is clicked and shot.
- **Controls** (`controls: true`): mark a group of choices with `testIdControls` and `testIdControl`, a select with `testIdSelect`. A `global-` name applies to the whole page (e.g. report period), anything else to a tab. Level dependencies: [`views.ts`](../scripts/screenshots/capture/views.ts).

### Scenarios

Flows you cannot reach by opening a URL on the seeded app (such as first-run setup, which needs a server with no users) are driven by scenarios in [`scenarios/flows/`](../scripts/screenshots/scenarios/flows). They run after the pages in every run and double as tests: a failed assertion fails the run (`FAILED.png` and the app's log tail are saved).

- **Own server.** Each scenario, per language, theme and viewport, starts the app binary on an empty data directory and free port, then discards it. The binary is `SAVVY_BIN`, else the Docker image's, else one from `go build`. A binary from `go build` serves `public/build` (or `SAVVY_WEB_DIR/public/build`; `--local` sets both), so run `APP_ENV=screenshots npm run build` first. Shots go to `scenarios/<name>/<language>-<theme>/<viewport>/NN-<shot>.png`.
- **Detail.** `detail` sets how much is shot; steps and assertions always run. A shot declared `{ name: 'wrong-password', detail: true }` is taken only at `full`. The scenario axes can differ from the pages': for example pages in English only, onboarding in both languages and themes.
- **Writing one.** `scenario({ name, covers, shots }, async ({ page, server, shot, run }) => { ... })`, with `expect` from `playwright/test`. Find elements by `testId` marks or `name`/`id` attributes, never by text, since scenarios run in every language. `shots` lists every shot in order; an unlisted or untaken shot fails the run.
- **Composing.** `await run(otherScenario)` runs another scenario first on the same server and page. Its shots are not repeated (`{ shots: true }` repeats them). State what it leaves behind in `leaves`.
- **Outside the browser.** `auth/oauth-app` connects an AI app over OAuth with the repo's `cmd/oauth-probe` (what `go tool mage probe:oauth` runs). `scenarios/probe.ts` starts it with `-no-browser`, reads the consent address and opens it in the page. The binary is `OAUTH_PROBE_BIN`, else the Docker image's, else one built with `go build` on first use.
- **Every page has one.** `covers` lists the page paths (from `pages.ts`) the scenario visits. The run fails for a page no scenario covers unless [`pending.ts`](../scripts/screenshots/scenarios/pending.ts) lists it with a reason, or for a page in both. A new page needs its scenario; `pending.ts` should only shrink.

Only scenarios: `npm run screenshots -- scenarios` (`configs/scenarios.json` has `"capture": false`), with `--local` to skip Docker.

### Good to know

- **Pages** come from [`pages.ts`](../resources/ts/app/pages.ts), so new ones are picked up. Pages with a `:param` need a source in [`routes.ts`](../scripts/screenshots/capture/routes.ts); skipped ones are logged.
- **Seed date.** Periods follow the current day, so the compose file seeds for today. Setting `date` in the seed config gives empty totals (the script warns).
- **Runs on different days differ**: on-screen dates follow the current day, and exchange rates are fetched live.
- **Animations** are off, so shots are still frames.
- **Playwright version**: the Docker image tag must match the `playwright` package version, or the build fails.
- **Linux**: output files of a Docker run belong to root.
