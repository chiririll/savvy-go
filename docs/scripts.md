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

The number of screenshots is the product of the lists: two languages and two themes make four times the files. Invalid values stop the run with a message.

### Dialogs, tabs, controls

The script finds what to click by `data-testid` marks that the app adds only in builds with `APP_ENV=screenshots`. Helpers are in [`test-id.ts`](../resources/ts/lib/test-id.ts).

- **Dialogs**: put `{...testId('some-id')}` on the element that opens a dialog. Shots go to `dialogs/`. Escape closes each one, so nothing is confirmed.
- **Tabs**: Radix tabs are found automatically; `SegmentedChoice` carries its own marks. Each unselected tab is clicked and shot.
- **Controls** (`controls: true`): mark a group of choices with `testIdControls` and `testIdControl`, a select with `testIdSelect`. A name starting with `global-` applies to the whole page (e.g. report period); anything else to a tab. Dependencies between levels are described in [`views.ts`](../scripts/screenshots/views.ts).

A dialog reachable in several ways is shot once per run.

### Good to know

- **Pages** come from [`pages.ts`](../resources/ts/app/pages.ts), so new pages are picked up automatically. Pages with a `:param` need a source in [`routes.ts`](../scripts/screenshots/routes.ts); skipped ones are listed in the log.
- **Seed date**: the app computes periods from the current day, so the compose file seeds for today. Setting `SEED_DATE` gives empty totals (the script warns).
- **Animations** are off, so shots are still frames.
- **Runs on different days differ** because dates on screen follow the current day. Exchange rates are fetched live, so those amounts can differ too.
- **Playwright version**: the Docker image tag must match the `playwright` package version; the build fails otherwise.
- **Linux**: the output files belong to root.
