// Screenshots every page of a running app, per seeded user, space and viewport.
//
// The app must be seeded with SEED_DEMO=true and SEED_MANIFEST (see
// internal/seed/manifest.go); the manifest says who can sign in, which spaces
// exist and which ids the parametrised pages need. See docs/scripts.md.
//
// Usage: npm --prefix scripts run screenshots -- [--only=<text in the page path>]
// Env:   BASE_URL (http://localhost:8080), MANIFEST (<repo>/screenshots/manifest.json),
//        OUT (<repo>/screenshots), CONCURRENCY (4), TZ (UTC), LOCALE (en-US)
import { mkdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium, devices, type Browser, type BrowserContextOptions } from 'playwright'
import type { Page } from '../../resources/ts/app/pages.ts'
import type { Manifest, ManifestSpace } from './manifest.ts'
import { paramValue, routes, skip } from './routes.ts'

const env = (name: string, fallback: string) => process.env[name]?.trim() || fallback
const baseURL = env('BASE_URL', 'http://localhost:8080').replace(/\/$/, '')
// Relative to the repository, not the working directory (npm --prefix runs from scripts/).
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')
const outDir = path.resolve(env('OUT', path.join(root, 'screenshots')))
const manifestPath = path.resolve(env('MANIFEST', path.join(outDir, 'manifest.json')))
const concurrency = Number(env('CONCURRENCY', '4'))
const timezoneId = env('TZ', 'UTC')
const locale = env('LOCALE', 'en-US')
const only = process.argv.find((a) => a.startsWith('--only='))?.slice('--only='.length)

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

const slug = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
const pageName = (p: string) => slug(p.replace(/^\//, '').replace(/\//g, '-')) || 'dashboard'

interface Job {
    label: string
    viewport: string
    dir: string
    // The page's "today" and the session, space and pages to capture.
    storageState?: BrowserContextOptions['storageState']
    spaceId?: number
    pages: { path: string; name: string }[]
}

const failures: string[] = []

async function main() {
    const manifest: Manifest = JSON.parse(await readFile(manifestPath, 'utf8'))
    // The app works out some periods ("last 30 days") from the server's own clock,
    // so data seeded for another day only looks right if the server runs on that
    // day. The browser's clock is moved to the seeded day, but the server's is not.
    const seeded = new Date(manifest.now)
    const day = (d: Date) => new Intl.DateTimeFormat('en-CA', { timeZone: timezoneId }).format(d)
    const now = day(seeded) === day(new Date()) ? null : seeded
    if (now) console.warn(`Data was seeded for ${day(seeded)}, not today (${day(new Date())}): server-side periods will be empty.`)
    const browser = await chromium.launch()

    // A page with a ":param" is captured only where the manifest has a value for it.
    const resolved = new Set<string>()
    const unresolved = new Set<string>()
    const fill = (page: Page, space?: ManifestSpace) => {
        if (skip.has(page.path)) return []
        let missing = false
        const p = page.path.replace(/:\w+/g, () => {
            const value = paramValue[page.path]?.(manifest, space)
            if (value === undefined) missing = true
            return String(value)
        })
        if (missing) {
            unresolved.add(page.path)
            return []
        }
        resolved.add(page.path)
        // The name leaves out the ids, which differ from run to run.
        return only && !p.includes(only) ? [] : [{ path: p, name: pageName(page.path.replace(/\/:\w+/g, '')) }]
    }
    const pagesOf = (scope: Page['scope'], space?: ManifestSpace) =>
        routes.filter((r) => r.scope === scope).flatMap((r) => fill(r, space))

    const jobs: Job[] = []
    for (const viewport of Object.keys(viewports)) {
        jobs.push({ label: 'public', viewport, dir: 'public', pages: pagesOf('public') })
    }

    for (const user of manifest.users) {
        const storageState = await signIn(browser, user.email, user.password)
        const mine = manifest.spaces.filter((s) => user.key in s.members)
        for (const viewport of Object.keys(viewports)) {
            mine.forEach((sp, i) => {
                const list = pagesOf('space', sp)
                // Pages of the user and the server do not depend on the space: once is enough.
                if (i === 0) {
                    list.push(...pagesOf('user'))
                    if (user.role === 'admin') list.push(...pagesOf('admin'))
                }
                jobs.push({
                    label: `${user.key}/${slug(sp.name)}`, viewport, storageState, spaceId: sp.id, pages: list,
                    dir: path.join(user.key, slug(sp.name)),
                })
            })
        }
    }
    // Some spaces have no value (no automation rules): only a page that has none anywhere is worth a warning.
    const never = [...unresolved].filter((p) => !resolved.has(p))
    if (never.length) console.warn(`Left out, no value for their parameters in the manifest: ${never.join(', ')}`)

    let done = 0
    const total = jobs.reduce((n, j) => n + j.pages.length, 0)
    const queue = [...jobs]
    await Promise.all(
        Array.from({ length: concurrency }, async () => {
            for (let job = queue.shift(); job; job = queue.shift()) {
                await run(browser, job, now, () => {
                    done++
                    if (process.stdout.isTTY) process.stdout.write(`\r${done}/${total}`)
                })
            }
        }),
    )
    await browser.close()

    console.log(`\n${done - failures.length} screenshots in ${outDir}`)
    if (failures.length) {
        console.error(`${failures.length} failed:\n  ${failures.join('\n  ')}`)
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

async function run(browser: Browser, job: Job, now: Date | null, tick: () => void) {
    const context = await browser.newContext({
        ...viewports[job.viewport], baseURL, locale, timezoneId, storageState: job.storageState,
        reducedMotion: 'reduce',
    })
    // install() keeps the time running: charts animate off the clock and stall on a frozen one.
    if (now) await context.clock.install({ time: now })
    await context.addInitScript(
        ({ css, spaceId }) => {
            if (spaceId !== undefined) {
                localStorage.setItem('savvy-space', JSON.stringify({ state: { currentId: spaceId }, version: 0 }))
            }
            document.addEventListener('DOMContentLoaded', () => {
                const style = document.createElement('style')
                style.textContent = css
                document.head.append(style)
            })
        },
        { css: noMotion, spaceId: job.spaceId },
    )
    const page = await context.newPage()
    const dir = path.join(outDir, job.dir, job.viewport)
    await mkdir(dir, { recursive: true })
    for (const p of job.pages) {
        const file = path.join(dir, `${p.name}.png`)
        try {
            await page.goto(p.path, { waitUntil: 'networkidle' })
            await page.screenshot({ path: file, fullPage: true, animations: 'disabled' })
        } catch (e) {
            failures.push(`${job.label} ${job.viewport} ${p.path}: ${(e as Error).message.split('\n')[0]}`)
        }
        tick()
    }
    await context.close()
}

main().catch((e) => {
    console.error(e)
    process.exit(1)
})
