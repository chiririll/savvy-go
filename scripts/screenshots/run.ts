// Screenshots every page of a running app, per seeded user, space and viewport.
//
// The app must be seeded with SEED_DEMO=true and SEED_MANIFEST (see
// internal/seed/manifest.go); the manifest says who can sign in, which spaces
// exist and which ids the parametrised pages need. See docs/scripts.md.
//
// What is taken is set in a config of configs/ (users, spaces, viewports, languages, themes, pages, ...).
// Run it with docker compose (docker-compose.screenshots.yml).
// Env:   BASE_URL (http://localhost:8080), MANIFEST (<repo>/screenshots/manifest.json),
//        OUT (<repo>/screenshots), RUN_NAME (the start time; the files go to OUT/RUN_NAME),
//        CONFIG (the name of a file of configs/; none: the defaults), CONCURRENCY (4), TZ (UTC)
import { mkdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium, devices, type Browser, type BrowserContextOptions } from 'playwright'
import { loadConfig } from './config.ts'
import { captureControls } from './controls.ts'
import { captureDialogs } from './dialogs.ts'
import type { Manifest } from './manifest.ts'
import { plan } from './plan.ts'
import { routes } from './routes.ts'
import { log, state, type Job } from './shared.ts'
import { captureTabs } from './tabs.ts'

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
const timezoneId = env('TZ', 'UTC')

const viewports: Record<string, BrowserContextOptions> = {
    desktop: { viewport: { width: 1440, height: 900 } },
    mobile: { ...devices['Pixel 7'] },
}

// Animations are switched off rather than waited for.
const noMotion = `*, *::before, *::after {
    animation: none !important;
    transition: none !important;
    caret-color: transparent !important;
}`

async function main() {
    const manifest: Manifest = JSON.parse(await readFile(manifestPath, 'utf8'))
    state.config = await loadConfig(configName, manifest, {
        viewports: Object.keys(viewports),
        pages: routes.map((r) => r.path),
    })
    // The app works out some periods ("last 30 days") from the server's own clock,
    // so data seeded for another day only looks right if the server runs on that
    // day. The browser's clock is moved to the seeded day, but the server's is not.
    const seeded = new Date(manifest.now)
    const day = (d: Date) => new Intl.DateTimeFormat('en-CA', { timeZone: timezoneId }).format(d)
    const now = day(seeded) === day(new Date()) ? null : seeded
    if (now) console.warn(`Data was seeded for ${day(seeded)}, not today (${day(new Date())}): server-side periods will be empty.`)
    const browser = await chromium.launch()

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
    await browser.close()

    console.log(`${state.saved} screenshots in ${outDir}`)
    if (state.failures.length) {
        console.error(`${state.failures.length} failed:\n  ${state.failures.join('\n  ')}`)
        process.exit(1)
    }
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
    const context = await browser.newContext({
        ...viewports[job.viewport], baseURL, timezoneId, storageState: job.storageState,
        // What the browser itself says; the app also keeps its own choice in storage (below).
        locale: job.language, colorScheme: job.theme as 'light' | 'dark',
        reducedMotion: 'reduce',
    })
    // install() keeps the time running: charts animate off the clock and stall on a frozen one.
    if (now) await context.clock.install({ time: now })
    await context.addInitScript(
        ({ css, spaceId, language, theme }) => {
            if (spaceId !== undefined) {
                localStorage.setItem('savvy-space', JSON.stringify({ state: { currentId: spaceId }, version: 0 }))
            }
            // The keys of the app's language and theme choice (lib/i18n.ts, hooks/use-theme.ts).
            localStorage.setItem('savvy.locale', language)
            localStorage.setItem('theme', theme)
            document.addEventListener('DOMContentLoaded', () => {
                const style = document.createElement('style')
                style.textContent = css
                document.head.append(style)
            })
        },
        { css: noMotion, spaceId: job.spaceId, language: job.language, theme: job.theme },
    )
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
                    if (state.config.tabs) await captureTabs(page, job, page.locator('body'), p.name, dir, 'page')
                    if (state.config.controls) await captureControls(page, job, page.locator('body'), p.name, dir, true)
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
