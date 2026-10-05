# Scripts

## `npm run logos`

Regenerates every static logo file from the `Logo` React component in [`resources/ts/components/shared/Logo.tsx`](../resources/ts/components/shared/Logo.tsx), which is the single source of truth for the logo. The script ([`scripts/update-logos.ts`](../scripts/update-logos.ts)) renders the component to SVG and writes `public/favicon.svg` (with a dark-theme colour), the PNG icons (`favicon-96x96`, `web-app-manifest-192x192`/`512x512`, `apple-touch-icon`) and a multi-size `favicon.ico` into `public/`, plus the light and dark README logos into `docs/images/`. Edit the component, then run the command. The scripts have their own [`package.json`](../scripts/package.json) (`tsx`, `sharp`, `playwright`), so they never end up in the app's dependencies or the Docker build; install them once with `npm --prefix scripts install`. The commands in the root `package.json` (`npm run logos`, `npm run screenshots`) are aliases for `npm --prefix scripts run <name>`. The script also imports `react`, so the root `npm install` is needed too.

## Screenshots of every page

Takes a screenshot of every page of the app for each demo user, each space they belong to, and a desktop and a mobile viewport (Playwright, Chromium). It is for looking over the whole interface at once.

```bash
npm run screenshots
```

It is an alias for `npm --prefix scripts run screenshots`, which runs [`scripts/screenshots/compose.mjs`](../scripts/screenshots/compose.mjs): these two commands, the second one even if the first fails:

```bash
docker compose -f docker-compose.screenshots.yml up --build --abort-on-container-exit --exit-code-from shots
docker compose -f docker-compose.screenshots.yml down -v
```

[`docker-compose.screenshots.yml`](../docker-compose.screenshots.yml) builds the app, seeds it with the demo data on an empty volume and, once it is healthy, runs the screenshot script in the official Playwright image against it over the compose network. Each run writes to a directory of its own in `screenshots/` in the repository root (ignored by git), named by when it started, such as `screenshots/2026-10-05_14-30-12/`, so older runs stay and nothing has to be cleaned; the path is printed at the end; Ctrl+C stops the containers and still runs the `down -v` (a second Ctrl+C gives up on it); `down -v` removes the app's database with the volume. Nothing needs installing besides Docker: the browsers come with the image ([`scripts/screenshots/Dockerfile`](../scripts/screenshots/Dockerfile)) and the `playwright` package is a dependency of [`scripts/package.json`](../scripts/package.json), which CI does not install. The image tag and the package must be the same Playwright version; the build stops with a message if they are not.

The script ([`scripts/screenshots/run.ts`](../scripts/screenshots/run.ts), the entry point; `plan.ts` works out what to take, and `tabs.ts`, `controls.ts` and `dialogs.ts` take the tabs, controls and dialogs of a page) learns who can sign in, which spaces exist and which ids the parametrised pages need from a manifest the seed writes when `SEED_MANIFEST` is set. Inside a run's directory the files land in `<language>-<theme>/<user>/<space>/<desktop|mobile>/<page>.png` (for example `en-light/alex/demo/desktop/transactions.png`); pages without a space (the sign-in and invitation pages) are in `<language>-<theme>/public/`. On a phone the sidebar is a drawer behind a button, so each space also gets a `mobile/sidebar.png` with it open. The pages of the user's own settings and of administration are taken once per user, in their first space. What is taken is set in a config (below); its environment is set in the compose file: `BASE_URL`, `MANIFEST`, `OUT`, `CONFIG` (the config's name), `CONCURRENCY` (4) and `TZ` (UTC). The directory's name is the host's local time, which `npm run screenshots` passes in as `RUN_NAME` (the container knows only UTC); set `RUN_NAME` yourself to name a run, say `RUN_NAME=before-redesign`. On Linux the files in `screenshots/` belong to root, since the container runs as root.

### What to take

What to take is a config, a JSON file in [`scripts/screenshots/configs/`](../scripts/screenshots/configs). Name one after the command to use it, and keep as many as are useful (`full`, which is English and Russian in both themes, `mobile`, and `reports`, the reports page for one user in English in the light theme, are there to start from):

```bash
npm run screenshots                  # no name: the defaults, everything in English and the light theme
npm run screenshots -- full          # configs/full.json
npm run screenshots -- mobile        # configs/mobile.json
npm run screenshots -- reports       # configs/reports.json
```

Without a name no file is needed: the defaults are in [`config.ts`](../scripts/screenshots/config.ts). With one, the files go into the image with the script, so a new or changed config is picked up on the next run. The name also ends the run's directory (`screenshots/2026-10-05_14-30-12_full/`), so runs with different configs are easy to tell apart. An unknown name stops before anything is built and lists the configs there are. A config looks like this:

```json
{
    "users": ["alex", "demo"],
    "spaces": ["Demo"],
    "viewports": ["mobile"],
    "languages": ["en", "ru"],
    "themes": ["light", "dark"],
    "pages": ["/transactions", "/settings"],
    "sidebar": false,
    "dialogs": true,
    "tabs": true
}
```

| Setting      | Meaning | When left out or empty |
|--------------|---------|------------------------|
| `users`      | Keys of the seeded users: `alex`, `jordan`, `sam`, `demo`. | all of them |
| `spaces`     | Names of spaces, such as `Demo` or `Family Budget`; each user is taken in the ones they belong to. | all |
| `viewports`  | `desktop`, `mobile`. | both |
| `languages`  | The languages of the app, one directory of `resources/ts/locales` each (`en`, `ru`). | `en` |
| `themes`     | `light`, `dark`. | `light` |
| `pages`      | Paths as in [`resources/ts/app/pages.ts`](../resources/ts/app/pages.ts); a path also takes the pages under it, so `/settings` is all the settings pages. | all pages |
| `sidebar`    | The sidebar drawer on a phone. | `true` |
| `dialogs`    | The dialogs of each page. | `true` |
| `tabs`       | The tabs of each page and dialog. | `true` |
| `controls`   | The type of a chart, its grouping and the period of a report, each choice in each tab (see below). | `false` |

Every language and theme is taken in full, each in a directory of its own, so the number of screenshots is the product of the lists: two languages and two themes make four times the files. An unknown value or a path that matches no page stops the run with a message saying what is wrong and what there is. A language and a theme are set the way a user sets them, through the app's own settings in the browser storage, and the browser is told the same (`locale`, `colorScheme`).

### Dialogs

After each page the script also opens the dialogs on it and takes a screenshot of each, into `<user>/<space>/<desktop|mobile>/dialogs/`: `<page>-<id>.png` for a page's own, such as `accounts-create.png` or `categories-edit.png`, and `layout-<id>.png` for those of the page frame (the new-transaction menu, creating a space), which are taken once per space. A phone gets them too (the sidebar is opened when the button is in it); a dialog is shot as it is on the screen, not full-page.

The script finds them by marks the app puts on what opens a dialog: `data-testid="<id>"` on a button or a menu item, and `data-testid-menu="<id>"` on the button of a menu with such items (the menus of the first five rows are opened). They are added with `testId()` and `testIdMenu()` from [`resources/ts/lib/test-id.ts`](../resources/ts/lib/test-id.ts), mostly in the shared `PageHeader` (the create button) and `RowActions` (edit, delete), and are present only in a build made with `APP_ENV=screenshots`, which the compose file sets: the value is a compile-time constant, so the marks never reach another build. That environment also hides the development banner. To get a new dialog into the screenshots, put `{...testId('some-id')}` on the element that opens it. Escape closes each dialog, so nothing is confirmed.

### Tabs

On a page and in each open dialog the script also goes through the tabs: for every group of choices it clicks each one that is not selected, takes a screenshot, and puts the selected one back. A page's go next to its own shot (`transactions-pending.png`, `categories-income.png`), a dialog's into `dialogs/` (`layout-transaction-expense-transfer.png`); a page or dialog with several groups gets the group's number in the name (`reports-2-...`). Two kinds are found: Radix tabs by their role, with nothing to mark, and the app's `SegmentedChoice` (the type switch in the transaction, recurring, category and debt forms), which has no role and carries `testIdTabs()` and `testIdTab()` instead, set in that one component.

The same dialog can be reached in several ways (a menu item, or a tab of another dialog), so it is shot once per run: a dialog is identified by its text and by which choice is selected in each of its tab groups, not by the path that led to it, and a repeat is logged as `already shot`.

### Controls

With `controls` on, the script also takes what the app lets you change about the way it shows its data: the type of a chart (line or bar, donut, bar or treemap), its grouping (day, week, month) and the period of a report. Each choice is clicked, shot and put back, like a tab, and so are the choices that come with it: picking *Month* brings a select of months, and the first three other options of a select are taken (`MAX_SELECT_OPTIONS` in `controls.ts`). The reports page has them in each of its tabs, so they are reached through the tabs and `tabs` has to stay on. The files are named after the page (or its tab), the control and the choice: `reports-period-month.png`, `reports-period-month-period-value-august-2026.png`, `reports-cashflow-group-by-week.png`, `reports-expenses-view-treemap.png`, `reports-compare-no-comparison.png`.

The script finds them by marks, like dialogs: `testIdControls(name)` on a group of choices with `testIdControl(id, active)` on each, and `testIdSelect(name)` on a select (the helpers are in [`resources/ts/lib/test-id.ts`](../resources/ts/lib/test-id.ts)). A name starting with `global-` is for a control on the whole page, such as the period filters of the reports, so it is taken once and not again in every tab. A control worth taking is one more mark on it.

### Comparing runs

```bash
npm run screenshots:diff -- 2026-10-05_14-30-12 2026-10-05_16-02-40    # old, new; "latest" is the newest run
```

[`scripts/screenshots/diff.ts`](../scripts/screenshots/diff.ts) matches the files of two runs by their path inside the run, compares each pair pixel by pixel and writes the new screenshot with every changed area boxed in red to `screenshots/<old>_vs_<new>/`, next to a `report.txt` (how many files changed, which are only in one of the runs, and the share of changed pixels in each file). Only the changed files are written. It needs no Docker, only `npm --prefix scripts install`.

A difference of up to 24 in a colour channel is ignored (font smoothing), and changes close to each other are one box. A screenshot that became taller or wider has the added part boxed too. Compare runs of one day and one config: across days the dates and periods differ everywhere (the script warns), and a block that merely shifted down marks everything below it as changed.

Things to know:

- **Log.** The script prints a line for each page (`[alex/demo desktop] 12/274 /transactions`, with the user, the space and the viewport it is from, since runs go side by side), an indented line for each dialog it shot, a `no dialog from "<id>" on <page>` line when clicking a mark opened nothing, and `retrying` or `FAILED` lines for pages that did not load. It ends with the number of screenshots and the failures, and exits with an error if there were any. The compose run shows it as it goes.

- **Which pages.** They come from [`resources/ts/app/pages.ts`](../resources/ts/app/pages.ts), the list the router is built from, so a new page is captured without touching the script. A page with a `:param` is captured only when [`scripts/screenshots/routes.ts`](../scripts/screenshots/routes.ts) says where its value comes from; the script lists the ones it left out. A test (`resources/ts/app/pages.test.ts`) keeps the router and that list in step.
- **Seeded for today.** The app works out periods such as "last 30 days" from the server's clock, so the compose file seeds for the current day (`SEED_DATE` unset). Pinning another day with `SEED_DATE=YYYY-MM-DD` would show empty totals; the script warns when that happens.
- **No animations.** The script turns them off (and asks for reduced motion, which the charts honour) instead of waiting for them, so a capture is a still frame.
- **Not repeatable across days.** The dates on screen follow the current day, so two runs on different days differ; the demo data itself is the same for the same date.
- **Currency rates** are fetched from the network in the background, so exchange-rate amounts can differ between runs.
