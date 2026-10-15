// Screenshots every page of a running app, per seeded user, space and viewport, then runs the
// scenarios (scenarios/): flows driven on servers of their own.
//
// The app must be seeded with `--seed-config` with a `manifest` (see
// internal/seed/manifest.go); the manifest says who can sign in, which spaces
// exist and which ids the parametrised pages need. See docs/scripts.md.
//
// What is taken is set in a config of configs/ (users, spaces, viewports, languages, themes, pages, ...).
// Run it with docker compose (docker-compose.screenshots.yml).
// Env:   BASE_URL (http://localhost:8080), MANIFEST (<repo>/screenshots/manifest.json),
//        OUT (<repo>/screenshots), RUN_NAME (the start time; the files go to OUT/RUN_NAME),
//        CONFIG (the name of a file of configs/; none: the defaults), CONCURRENCY (4), TZ (UTC),
//        SAVVY_BIN (the app's binary that scenarios start; see scenarios/server.ts)
import { mkdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium, type Browser } from 'playwright'
import { loadConfig } from './config.ts'
import { newContext, timezoneId, viewports } from './context.ts'
import { captureDialogs } from './capture/dialogs.ts'
import type { Manifest } from './capture/manifest.ts'
import { plan } from './capture/plan.ts'
import { routes } from './capture/routes.ts'
import { runScenarios, shotsTaken } from './scenarios/runner.ts'
import { log, state, type Job } from './shared.ts'
import { captureViews } from './capture/views.ts'

const env = (name: string, fallback: string) => process.env[name]?.trim() || fallback
const baseURL = env('BASE_URL', 'http://localhost:8080').replace(/\/$/, '')
// Relative to the repository, not the working directory (npm --prefix runs from scripts/).
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')
const outRoot = path.resolve(env('OUT', path.join(root, 'screenshots')))
const configName = env('CONFIG', '')
// Every run has a directory of its own, named by when it started (and by its config), so nothing
// has to be cleaned between runs.
const stamp = new Date().toISOString().slice(0, 19).replace('T', '_').replace(/:/g, '-')
const runName = env('RUN_NAME', configName ? `${stamp}_${configName}` : stamp)
const outDir = path.join(outRoot, runName)
const manifestPath = path.resolve(env('MANIFEST', path.join(outRoot, 'manifest.json')))
const concurrency = Number(env('CONCURRENCY', '4'))

async function main() {
    const manifest = await readManifest()
    state.config = await loadConfig(configName, manifest ?? emptyManifest, {
        viewports: Object.keys(viewports),
        pages: routes.map((r) => r.path),
    })
    if (state.config.capture && !manifest) throw new Error(`no manifest at ${manifestPath}: seed the app with --seed-config, or set "capture": false`)
    const browser = await chromium.launch()

    if (state.config.capture && manifest) await capturePages(browser, manifest)
    if (state.config.scenarios) {
        const failures = await runScenarios(browser, {
            config: state.config, outDir, concurrency, binary: env('SAVVY_BIN', ''),
        })
        state.saved += shotsTaken.shots
        state.failures.push(...failures.map((f) => `scenario ${f}`))
    }
    await browser.close()

    console.log(`${state.saved} screenshots in ${outDir}`)
    if (state.failures.length) {
        console.error(`${state.failures.length} failed:\n  ${state.failures.join('\n  ')}`)
        process.exit(1)
    }
}

const emptyManifest: Manifest = { now: new Date().toISOString(), users: [], spaces: [], invitations: [] }

/** The manifest of the seeded app; undefined when there is none (scenarios start their own servers). */
async function readManifest(): Promise<Manifest | undefined> {
    try {
        return JSON.parse(await readFile(manifestPath, 'utf8'))
    } catch (e) {
        if ((e as NodeJS.ErrnoException).code === 'ENOENT') return undefined
        throw e
    }
}

async function capturePages(browser: Browser, manifest: Manifest) {
    // The app works out some periods ("last 30 days") from the server's own clock,
    // so data seeded for another day only looks right if the server runs on that
    // day. The browser's clock is moved to the seeded day, but the server's is not.
    const seeded = new Date(manifest.now)
    const day = (d: Date) => new Intl.DateTimeFormat('en-CA', { timeZone: timezoneId }).format(d)
    const now = day(seeded) === day(new Date()) ? null : seeded
    if (now) console.warn(`Data was seeded for ${day(seeded)}, not today (${day(new Date())}): server-side periods will be empty.`)

    const jobs = await plan(manifest, state.config, (email, password) => signIn(browser, email, password))

    state.total = jobs.reduce((n, j) => n + j.pages.length, 0)
    console.log(
        `${baseURL}, config ${configName || 'default'}: ${state.total} pages in ${jobs.length} runs; languages ${state.config.languages.join(', ')}, themes ${state.config.themes.join(', ')}, ` +
            `viewports ${state.config.viewports.join(', ')}; ${concurrency} at a time`,
    )
    const queue = [...jobs]
    await Promise.all(
        Array.from({ length: concurrency }, async () => {
            for (let job = queue.shift(); job; job = queue.shift()) {
                await run(browser, job, now)
            }
        }),
    )
}

/** Signs in through the API and returns the resulting cookies. */
async function signIn(browser: Browser, email: string, password: string) {
    const context = await browser.newContext({ baseURL })
    const res = await context.request.post('/api/auth/login', { data: { email, password } })
    if (!res.ok()) throw new Error(`sign in as ${email}: ${res.status()} ${await res.text()}`)
    const state = await context.storageState()
    await context.close()
    return state
}

async function run(browser: Browser, job: Job, now: Date | null) {
    const context = await newContext(browser, { ...job, baseURL, now })
    const page = await context.newPage()
    const dir = path.join(outDir, job.dir, job.viewport)
    await mkdir(dir, { recursive: true })
    // Marks of the page frame (header, sidebar) are the same on every page: once per job is enough.
    const frameDone = new Set<string>()
    for (const p of job.pages) {
        const file = path.join(dir, `${p.name}.png`)
        // A slow machine now and then stalls a capture; one more try is enough.
        let ok = false
        for (let attempt = 1; ; attempt++) {
            try {
                await page.goto(p.path)
                // Wait for the data to arrive, but not for a page that never goes quiet.
                await page.waitForLoadState('networkidle', { timeout: 3000 }).catch(() => {})
                if (p.sidebar) {
                    await page.locator('[data-sidebar="trigger"]').first().click()
                    await page.locator('[data-sidebar="sidebar"][data-mobile="true"]').waitFor()
                }
                // The open drawer is fixed to the screen: a full-page shot would stretch it over the page.
                await page.screenshot({ path: file, fullPage: !p.sidebar, animations: 'disabled' })
                state.saved++
                ok = true
                log(job, `${state.done + 1}/${state.total} ${p.path}${p.sidebar ? ' (sidebar)' : ''}`)
                if (!p.sidebar) {
                    await captureViews(page, job, page.locator('body'), p.name, dir)
                    if (state.config.dialogs) await captureDialogs(page, job, p.name, dir, frameDone)
                }
                break
            } catch (e) {
                const why = (e as Error).message.split('\n')[0]
                if (attempt < 2) {
                    log(job, `retrying ${p.path}: ${why}`)
                    continue
                }
                state.failures.push(`${job.label} ${job.viewport} ${p.path}: ${why}`)
                break
            }
        }
        state.done++
        if (!ok) log(job, `${state.done}/${state.total} ${p.path} FAILED`)
    }
    await context.close()
}

main().catch((e) => {
    console.error(e)
    process.exit(1)
})
