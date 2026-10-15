# Scripts

Helper scripts live in [`scripts/`](../scripts) with their own [`package.json`](../scripts/package.json), so they stay out of the app and the Docker build. Install them once:

```bash
npm --prefix scripts install
```

Root `npm run <name>` commands are aliases for `npm --prefix scripts run <name>`.

## Logos

```bash
npm run logos
```

Regenerates all logo files from the `Logo` component ([`Logo.tsx`](../resources/ts/components/shared/Logo.tsx)): favicons and app icons in `public/`, README logos in `docs/images/`. Edit the component, then run the command. Needs the root `npm install` too.

## Screenshots

```bash
npm run screenshots              # defaults: English, light theme
npm run screenshots -- full      # use configs/full.json
```

Takes a screenshot of every page for each demo user, space and viewport (desktop and mobile). Only Docker is needed.

How it works: [`docker-compose.screenshots.yml`](../docker-compose.screenshots.yml) builds the app, seeds demo data into an empty volume and runs Playwright against it. When the run ends (even on failure or Ctrl+C), the containers and volume are removed, so every run starts with a fresh seed.

Output goes to `screenshots/<start time>[_<config>]/` (git-ignored):

```
<language>-<theme>/<user>/<space>/<desktop|mobile>/<page>.png
<language>-<theme>/public/       # pages without a space (sign-in, invitations)
```

Set `RUN_NAME` to name the run yourself, e.g. `RUN_NAME=before-redesign`.

### Configs

A config is a JSON file in [`scripts/screenshots/configs/`](../scripts/screenshots/configs). Included: `full` (English and Russian, both themes), `mobile`, `reports`. Without a name the defaults from [`config.ts`](../scripts/screenshots/config.ts) are used.

```json
{
    "users": ["alex", "demo"],
    "spaces": ["Demo"],
    "viewports": ["mobile"],
    "languages": ["en", "ru"],
    "themes": ["light", "dark"],
    "pages": ["/transactions", "/settings"],
    "dialogs": true,
    "tabs": true
}
```

| Setting     | Meaning                                                             | If omitted      |
|-------------|---------------------------------------------------------------------|-----------------|
| `users`     | Seeded users: `alex`, `jordan`, `sam`, `demo`                       | all             |
| `spaces`    | Space names, e.g. `Demo`, `Family Budget`                           | all             |
| `viewports` | `desktop`, `mobile`                                                 | both            |
| `languages` | Folders in `resources/ts/locales`                                   | `en`            |
| `themes`    | `light`, `dark`                                                     | `light`         |
| `pages`     | Paths from [`pages.ts`](../resources/ts/app/pages.ts); includes subpages (`/settings` = all settings pages) | all |
| `sidebar`   | Sidebar drawer on mobile                                            | `true`          |
| `dialogs`   | Dialogs of each page                                                | `true`          |
| `tabs`      | Tabs of each page and dialog                                        | `true`          |
| `controls`  | Chart types, grouping and report periods                            | `false`         |
| `capture`   | The pages of the seeded app (needs its manifest)                    | `true`          |
| `scenarios` | The [scenarios](#scenarios)                                         | `true`          |
| `scenarioNames` | Scenarios by name or start of it (`"onboarding/"`)              | all             |
| `scenarioLanguages`, `scenarioThemes`, `scenarioViewports` | What the scenarios vary over, apart from the pages | `languages`, `themes`, `viewports` |
| `detail`    | `key`: the essential shot of each step; `full`: also error states and variants | `full` |

The number of screenshots is the product of the lists: two languages and two themes make four times the files. Invalid values stop the run with a message.

Any setting can also be given on the command line, on top of the config (or without one): `--name=a,b` for a list, `--name=true|false` for a flag.

```bash
npm run screenshots -- scenarios --scenarioLanguages=en,ru --scenarioThemes=light,dark
npm run screenshots -- --capture=false --scenarioNames=onboarding/ --detail=key
```

### Dialogs, tabs, controls

The script finds what to click by `data-testid` marks that the app adds only in builds with `APP_ENV=screenshots`. Helpers are in [`test-id.ts`](../resources/ts/lib/test-id.ts).

- **Dialogs**: put `{...testId('some-id')}` on the element that opens a dialog. Shots go to `dialogs/`. Escape closes each one, so nothing is confirmed.
- **Tabs**: Radix tabs are found automatically; `SegmentedChoice` carries its own marks. Each unselected tab is clicked and shot.
- **Controls** (`controls: true`): mark a group of choices with `testIdControls` and `testIdControl`, a select with `testIdSelect`. A name starting with `global-` applies to the whole page (e.g. report period); anything else to a tab. Dependencies between levels are described in [`views.ts`](../scripts/screenshots/capture/views.ts).

A dialog reachable in several ways is shot once per run.

### Scenarios

Flows that cannot be reached by opening a URL on the seeded app (the first-run setup, which needs a server with no users) are driven by scenarios in [`scenarios/flows/`](../scripts/screenshots/scenarios/flows). They run after the pages, in every run, and are also tests: a failed assertion fails the run (`FAILED.png` and the app's log tail are saved).

- **A server of its own.** Each scenario, in each language, theme and viewport of the config, starts the app's binary on an empty data directory and a free port, and throws it away afterwards. The binary is `SAVVY_BIN`, else the one in the Docker image, else one built with `go build` (it then serves `public/build`, so build that first with `APP_ENV=screenshots npm run build`). Shots go to `scenarios/<name>/<language>-<theme>/<viewport>/NN-<shot>.png`.
- **How precise.** `detail` (config or `--detail=`) picks how much of each scenario is shot; the steps and assertions always run. A shot declared as `{ name: 'wrong-password', detail: true }` is taken only at `full`; the others at both. The scenario axes (`scenarioLanguages`, `scenarioThemes`, `scenarioViewports`) can differ from those of the pages: for example every page in English only, but the onboarding in both languages and themes.
- **Writing one.** `scenario({ name, covers, shots }, async ({ page, server, shot, run }) => { ... })`. Use `expect` from `playwright/test`. Find elements by `testId` marks or `name`/`id` attributes, never by text: the same scenario runs in every language. `shots` lists every shot in order; a shot that is not listed, or listed and not taken, fails the run.
- **Composing.** `await run(otherScenario)` runs another scenario first on the same server and page, as the start of this one. Its shots are not taken again (`{ shots: true }` takes them). Say in `leaves` what state it leaves behind.
- **An app outside the browser.** `auth/oauth-app` connects an AI app over OAuth with the repository's own probe, `cmd/oauth-probe` (what `go tool mage probe:oauth` runs). `scenarios/probe.ts` starts it with `-no-browser`, reads the consent address it prints and opens that in the scenario's page. The binary is `OAUTH_PROBE_BIN`, else the one in the Docker image (built in its own stage), else one built with `go build` at the first use.
- **Every page has one.** `covers` lists the page paths (from `pages.ts`) the scenario goes through. The run fails for a page that no scenario covers unless it is listed with a reason in [`pending.ts`](../scripts/screenshots/scenarios/pending.ts), and for a page that is in both. Adding a page therefore means adding its scenario; `pending.ts` should only get shorter.

Only scenarios, without Docker:

```bash
APP_ENV=screenshots npm run build
cd scripts && CONFIG=scenarios npx tsx screenshots/run.ts   # configs/scenarios.json has "capture": false
```

### Good to know

- **Pages** come from [`pages.ts`](../resources/ts/app/pages.ts), so new pages are picked up automatically. Pages with a `:param` need a source in [`routes.ts`](../scripts/screenshots/capture/routes.ts); skipped ones are listed in the log.
- **Seed date**: the app computes periods from the current day, so the compose file seeds for today. Setting `date` in the seed config gives empty totals (the script warns).
- **Animations** are off, so shots are still frames.
- **Runs on different days differ** because dates on screen follow the current day. Exchange rates are fetched live, so those amounts can differ too.
- **Playwright version**: the Docker image tag must match the `playwright` package version; the build fails otherwise.
- **Linux**: the output files belong to root.
