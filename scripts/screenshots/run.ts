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
import { chromium, devices, type Browser, type BrowserContextOptions, type Locator, type Page as PwPage } from 'playwright'
import { pages, type Page } from '../../resources/ts/app/pages.ts'
import { loadConfig, pageWanted, type Config } from './config.ts'
import type { Manifest, ManifestSpace } from './manifest.ts'
import { paramValue, routes, skip } from './routes.ts'

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

const slug = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
const pageName = (p: string) => slug(p.replace(/^\//, '').replace(/\//g, '-')) || 'dashboard'

interface Job {
    label: string
    viewport: string
    language: string
    theme: string
    dir: string
    // The page's "today" and the session, space and pages to capture.
    storageState?: BrowserContextOptions['storageState']
    spaceId?: number
    pages: { path: string; name: string; sidebar?: boolean }[]
    /** What was shot in this run, by what was on the screen (see stateKey). */
    seen?: Set<string>
}

let config: Config
const failures: string[] = []
let saved = 0
let done = 0
let total = 0

// Runs go side by side, so every line says which one it is from.
const log = (job: Job, what: string) => console.log(`[${job.label} ${job.viewport}] ${what}`)

async function main() {
    const manifest: Manifest = JSON.parse(await readFile(manifestPath, 'utf8'))
    config = await loadConfig(configName, manifest, {
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
        return pageWanted(config, page.path) ? [{ path: p, name: pageName(page.path.replace(/\/:\w+/g, '')) }] : []
    }
    const pagesOf = (scope: Page['scope'], space?: ManifestSpace): Job['pages'] =>
        routes.filter((r) => r.scope === scope).flatMap((r) => fill(r, space))

    // Everything is taken once for each language and theme, in a directory of its own.
    const variants = config.languages.flatMap((language) => config.themes.map((theme) => ({ language, theme, name: `${language}-${theme}` })))

    const jobs: Job[] = []
    for (const v of variants) {
        for (const viewport of config.viewports) {
            const list = pagesOf('public')
            if (list.length) jobs.push({ label: `${v.name} public`, viewport, language: v.language, theme: v.theme, dir: path.join(v.name, 'public'), pages: list })
        }
    }

    for (const user of manifest.users) {
        if (config.users.length && !config.users.includes(user.key)) continue
        const storageState = await signIn(browser, user.email, user.password)
        const mine = manifest.spaces
            .filter((s) => user.key in s.members)
            .filter((s) => !config.spaces.length || config.spaces.some((n) => n.toLowerCase() === s.name.toLowerCase()))
        for (const v of variants) {
            for (const viewport of config.viewports) {
                mine.forEach((sp, i) => {
                    const list = pagesOf('space', sp)
                    // On a phone the sidebar is a drawer behind a button, so it is not in any page's shot.
                    if (viewport === 'mobile' && config.sidebar) {
                        list.unshift({ path: pages.dashboard.path, name: 'sidebar', sidebar: true })
                    }
                    // Pages of the user and the server do not depend on the space: once is enough.
                    if (i === 0) {
                        list.push(...pagesOf('user'))
                        if (user.role === 'admin') list.push(...pagesOf('admin'))
                    }
                    if (!list.length) return
                    jobs.push({
                        label: `${v.name} ${user.key}/${slug(sp.name)}`, viewport, language: v.language, theme: v.theme,
                        storageState, spaceId: sp.id, pages: list,
                        dir: path.join(v.name, user.key, slug(sp.name)),
                    })
                })
            }
        }
    }
    // Some spaces have no value (no automation rules): only a page that has none anywhere is worth a warning.
    const never = [...unresolved].filter((p) => !resolved.has(p))
    if (never.length) console.warn(`Left out, no value for their parameters in the manifest: ${never.join(', ')}`)

    total = jobs.reduce((n, j) => n + j.pages.length, 0)
    console.log(
        `${baseURL}, config ${configName || 'default'}: ${total} pages in ${jobs.length} runs; languages ${config.languages.join(', ')}, themes ${config.themes.join(', ')}, ` +
            `viewports ${config.viewports.join(', ')}; ${concurrency} at a time`,
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

    console.log(`${saved} screenshots in ${outDir}`)
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
                saved++
                ok = true
                log(job, `${done + 1}/${total} ${p.path}${p.sidebar ? ' (sidebar)' : ''}`)
                if (!p.sidebar) {
                    if (config.tabs) await captureTabs(page, job, page.locator('body'), p.name, dir, 'page')
                    if (config.controls) await captureControls(page, job, page.locator('body'), p.name, dir, true)
                    if (config.dialogs) await captureDialogs(page, job, p.name, dir, frameDone)
                }
                break
            } catch (e) {
                const why = (e as Error).message.split('\n')[0]
                if (attempt < 2) {
                    log(job, `retrying ${p.path}: ${why}`)
                    continue
                }
                failures.push(`${job.label} ${job.viewport} ${p.path}: ${why}`)
                break
            }
        }
        done++
        if (!ok) log(job, `${done}/${total} ${p.path} FAILED`)
    }
    await context.close()
}

// Tabs: a group of choices of which one is selected. Radix tabs are found by their role; the
// app's SegmentedChoice has no role and carries test ids instead (resources/ts/lib/test-id.ts).
const tabGroup = '[role="tablist"], [data-testid-tabs]'
const tabItem = '[role="tab"], [data-testid-tab]'

/**
 * What an open dialog shows: its text and the selected choice of each of its tab groups. Different
 * ways to the same dialog (a menu item, or a tab of another dialog) give the same key.
 */
const stateKey = (dialog: Locator) =>
    dialog.evaluate((el) => {
        const text = (el as HTMLElement).innerText.replace(/\s+/g, ' ').trim()
        const selected = [...el.querySelectorAll<HTMLElement>('[role="tab"], [data-testid-tab]')]
            .filter((t) => t.getAttribute('aria-selected') === 'true' || t.getAttribute('data-testid-active') === 'true')
            .map((t) => t.getAttribute('data-testid-tab') ?? t.id)
        return `${text}|${selected.join(',')}`
    })

/** True the first time this run sees what the dialog shows. */
async function fresh(job: Job, dialog: Locator) {
    const key = await stateKey(dialog)
    job.seen ??= new Set()
    if (job.seen.has(key)) return false
    job.seen.add(key)
    return true
}

/** A shot of every choice of every tab group in scope, which is a page or an open dialog. */
async function captureTabs(page: PwPage, job: Job, scope: Locator, base: string, folder: string, kind: 'page' | 'dialog') {
    try {
        const groups = scope.locator(tabGroup)
        const count = await groups.count()
        for (let g = 0; g < count; g++) {
            const group = groups.nth(g)
            if (!(await group.isVisible())) continue
            const choices = await group.evaluate((el, itemSelector) => {
                return [...el.querySelectorAll<HTMLElement>(itemSelector)].map((item, i) => ({
                    // A Radix trigger's id ends with its value; SegmentedChoice says its own.
                    id: item.getAttribute('data-testid-tab') ?? item.id.match(/-trigger-(.+)$/)?.[1] ?? (item.textContent?.trim() || String(i)),
                    active: item.getAttribute('aria-selected') === 'true' || item.getAttribute('data-testid-active') === 'true',
                }))
            }, tabItem)
            const items = group.locator(tabItem)
            const start = choices.findIndex((c) => c.active)
            for (const [i, choice] of choices.entries()) {
                if (choice.active || (await items.nth(i).isDisabled())) continue
                try {
                    await items.nth(i).click({ timeout: 3000 })
                    // The page may load what the tab shows.
                    await page.waitForLoadState('networkidle', { timeout: 3000 }).catch(() => {})
                } catch (e) {
                    log(job, `  could not open tab "${choice.id}" of ${base}: ${(e as Error).message.split('\n')[0]}`)
                    continue
                }
                const name = `${base}-${count > 1 ? `${g + 1}-` : ''}${slug(choice.id)}`
                if (kind === 'dialog' && !(await fresh(job, scope))) {
                    log(job, `  tab ${name}: already shot`)
                    continue
                }
                await mkdir(folder, { recursive: true })
                await page.screenshot({ path: path.join(folder, `${name}.png`), fullPage: kind === 'page', animations: 'disabled' })
                saved++
                log(job, `  tab ${name}`)
                // A tab of a page has charts and filters of its own: their choices are taken in it.
                if (kind === 'page' && config.controls && (await group.getAttribute('role')) === 'tablist') {
                    await captureControls(page, job, scope, name, folder, false)
                }
            }
            // Back to the choice that was selected, for whatever comes next.
            if (start >= 0) await items.nth(start).click({ timeout: 3000 }).catch(() => {})
        }
    } catch (e) {
        failures.push(`${job.label} ${job.viewport} ${base} tabs: ${(e as Error).message.split('\n')[0]}`)
    }
}

// Controls: what changes the way a page shows its data, such as the type of a chart, its grouping or
// the period of a report. A group of choices is marked data-testid-controls and a select
// data-testid-select (resources/ts/lib/test-id.ts); each choice is taken like a tab, and what comes
// with it: choosing a month brings a select of months. A name starting with "global-" is for what is
// on the whole page, so it is taken once, not again in every tab.
const MAX_SELECT_OPTIONS = 3

const controlLabel = (name: string) => name.replace(/^global-/, '')

async function snap(page: PwPage, job: Job, folder: string, name: string) {
    await mkdir(folder, { recursive: true })
    await page.screenshot({ path: path.join(folder, `${name}.png`), fullPage: true, animations: 'disabled' })
    saved++
    log(job, `  control ${name}`)
}

/** Takes the page with each choice of each marked group in scope; `top` is the page as it is, not a tab of it. */
async function captureControls(page: PwPage, job: Job, scope: Locator, base: string, folder: string, top: boolean) {
    try {
        if (top) await captureSelects(page, job, scope, base, folder, true)
        const groups = scope.locator('[data-testid-controls]')
        const count = await groups.count()
        for (let g = 0; g < count; g++) {
            const group = groups.nth(g)
            if (!(await group.isVisible())) continue
            const name = (await group.getAttribute('data-testid-controls')) ?? ''
            if (!top && name.startsWith('global-')) continue
            const choices = await group.evaluate((el) =>
                [...el.querySelectorAll<HTMLElement>('[data-testid-control]')].map((item) => ({
                    id: item.getAttribute('data-testid-control') ?? '',
                    active: item.getAttribute('data-testid-active') === 'true',
                })),
            )
            const items = group.locator('[data-testid-control]')
            const start = choices.findIndex((c) => c.active)
            for (const [i, choice] of choices.entries()) {
                if (choice.active) continue
                const shot = `${base}-${controlLabel(name)}-${slug(choice.id)}`
                try {
                    await items.nth(i).click({ timeout: 3000 })
                    await page.waitForLoadState('networkidle', { timeout: 3000 }).catch(() => {})
                } catch (e) {
                    log(job, `  could not choose "${choice.id}" of ${name} on ${base}: ${(e as Error).message.split('\n')[0]}`)
                    continue
                }
                await snap(page, job, folder, shot)
                await captureSelects(page, job, scope, shot, folder, false)
            }
            // Back to the choice that was selected, for whatever comes next.
            if (start >= 0) await items.nth(start).click({ timeout: 3000 }).catch(() => {})
        }
    } catch (e) {
        failures.push(`${job.label} ${job.viewport} ${base} controls: ${(e as Error).message.split('\n')[0]}`)
    }
}

/** Takes the page with the first few other options of each marked select in scope chosen. */
async function captureSelects(page: PwPage, job: Job, scope: Locator, base: string, folder: string, top: boolean) {
    const selects = scope.locator('[data-testid-select]')
    const count = await selects.count()
    for (let s = 0; s < count; s++) {
        const select = selects.nth(s)
        if (!(await select.isVisible())) continue
        const name = (await select.getAttribute('data-testid-select')) ?? ''
        if (!top && name.startsWith('global-')) continue
        // The options of an open select are in a portal of their own.
        const options = page.locator('[role="option"]')
        try {
            await select.click({ timeout: 3000 })
            await page.locator('[role="listbox"]').waitFor({ state: 'visible', timeout: 3000 })
        } catch {
            await page.keyboard.press('Escape')
            continue
        }
        const list = await options.evaluateAll((els) =>
            els.map((el) => ({ text: (el.textContent ?? '').trim(), selected: el.getAttribute('data-state') === 'checked' })),
        )
        await page.keyboard.press('Escape')
        const current = list.findIndex((o) => o.selected)
        let taken = 0
        for (const [i, option] of list.entries()) {
            if (i === current || taken >= MAX_SELECT_OPTIONS) continue
            try {
                await select.click({ timeout: 3000 })
                await options.nth(i).click({ timeout: 3000 })
                await page.waitForLoadState('networkidle', { timeout: 3000 }).catch(() => {})
            } catch (e) {
                log(job, `  could not choose "${option.text}" of ${name} on ${base}: ${(e as Error).message.split('\n')[0]}`)
                await page.keyboard.press('Escape')
                continue
            }
            taken++
            await snap(page, job, folder, `${base}-${controlLabel(name)}-${slug(option.text)}`)
        }
        // Back to the option that was selected.
        if (current >= 0) {
            try {
                await select.click({ timeout: 3000 })
                await options.nth(current).click({ timeout: 3000 })
            } catch {
                await page.keyboard.press('Escape')
            }
        }
    }
}

// Dialogs are found by the marks the app puts on what opens them (resources/ts/lib/test-id.ts),
// which exist only in a build made with APP_ENV=screenshots. Another build has none and gets no shots.
//   data-testid="id"       a button or menu item that opens a dialog
//   data-testid-menu="id"  the button of a menu with such items; the first rows' menus are opened
// An id starting with "layout-" belongs to the page frame and is taken once per job.
const MENUS_PER_PAGE = 5

// The sidebar is a dialog too, but it is not what is being looked for.
const dialogSelector = '[role="dialog"]:not([data-sidebar]), [role="alertdialog"]'

async function captureDialogs(page: PwPage, job: Job, pageName: string, dir: string, frameDone: Set<string>) {
    const dialogs = path.join(dir, 'dialogs')
    const taken = new Set<string>()
    // A second mark with the same id on a page gets a number.
    const named = new Map<string, number>()
    const nameOf = (id: string) => {
        const n = named.get(id) ?? 0
        named.set(id, n + 1)
        const base = id.startsWith('layout-') ? id : `${pageName}-${id}`
        return n ? `${base}-${n}` : base
    }
    const wanted = (id: string) => !taken.has(id) && !(id.startsWith('layout-') && frameDone.has(id))
    const take = (id: string) => {
        taken.add(id)
        if (id.startsWith('layout-')) frameDone.add(id)
    }

    const closeAll = async () => {
        for (let i = 0; i < 3; i++) {
            if (!(await page.locator(`${dialogSelector}, [role="menu"]`).first().isVisible().catch(() => false))) break
            await page.keyboard.press('Escape')
            await page.waitForTimeout(100)
        }
    }

    // Clicks what opens a dialog and, when one appears, shoots it.
    const shoot = async (opener: Locator, id: string) => {
        try {
            await opener.click({ timeout: 3000 })
            await page.locator(dialogSelector).first().waitFor({ state: 'visible', timeout: 3000 })
        } catch (e) {
            log(job, `  no dialog from "${id}" on ${pageName}: ${(e as Error).message.split('\n')[0]}`)
            await closeAll()
            return
        }
        if (!(await fresh(job, page.locator(dialogSelector).first()))) {
            log(job, `  dialog from "${id}" on ${pageName}: already shot`)
            take(id)
            await closeAll()
            return
        }
        await mkdir(dialogs, { recursive: true })
        const name = nameOf(id)
        // A drawer or dialog is fixed to the screen: no full-page shot.
        await page.screenshot({ path: path.join(dialogs, `${name}.png`), animations: 'disabled' })
        saved++
        take(id)
        log(job, `  dialog ${name}`)
        if (config.tabs) await captureTabs(page, job, page.locator(dialogSelector).first(), name, dialogs, 'dialog')
        await closeAll()
    }

    // On a phone the sidebar is a drawer: what is in it is reached by opening it.
    const revealed = async (locator: Locator) => {
        if (await locator.isVisible()) return true
        if (job.viewport !== 'mobile') return false
        await page.locator('[data-sidebar="trigger"]').first().click()
        await page.locator('[data-sidebar="sidebar"][data-mobile="true"]').waitFor({ timeout: 3000 }).catch(() => {})
        return locator.isVisible()
    }

    const closeSidebar = async () => {
        if (await page.locator('[data-sidebar="sidebar"][data-mobile="true"]').isVisible().catch(() => false)) {
            await page.keyboard.press('Escape')
            await page.waitForTimeout(100)
        }
    }
    const attrs = (locator: Locator, name: string) =>
        locator.evaluateAll((els, n) => els.map((el) => el.getAttribute(n) ?? ''), name)

    try {
        // Buttons that open a dialog by themselves. Two with one id are two dialogs.
        const direct = await attrs(page.locator('[data-testid]'), 'data-testid')
        const nth = new Map<string, number>()
        for (const id of direct) {
            const k = nth.get(id) ?? 0
            nth.set(id, k + 1)
            if (id.startsWith('layout-') && frameDone.has(id)) continue
            const opener = page.locator(`[data-testid="${id}"]`).nth(k)
            if (await opener.isVisible()) await shoot(opener, id)
        }

        // Menus: each marked item is clicked in a freshly opened menu.
        const menuIds = await attrs(page.locator('[data-testid-menu]'), 'data-testid-menu')
        const seen = new Map<string, number>()
        for (const menuId of menuIds) {
            const k = seen.get(menuId) ?? 0
            seen.set(menuId, k + 1)
            if (k >= (menuId.startsWith('layout-') ? 1 : MENUS_PER_PAGE)) continue
            // An open sidebar drawer covers the rest of the page.
            await closeSidebar()
            const trigger = page.locator(`[data-testid-menu="${menuId}"]`).nth(k)
            if (!(await revealed(trigger))) continue
            const menu = page.locator('[role="menu"]').last()
            const open = async () => {
                await trigger.click({ timeout: 3000 })
                await menu.waitFor({ state: 'visible', timeout: 3000 })
            }
            let items: string[]
            try {
                await open()
                items = await attrs(menu.locator('[data-testid]'), 'data-testid')
            } catch {
                await closeAll()
                continue
            }
            await closeAll()
            for (const id of items.filter(wanted)) {
                try {
                    await open()
                } catch {
                    await closeAll()
                    continue
                }
                await shoot(menu.locator(`[data-testid="${id}"]`).first(), id)
            }
            await closeAll()
        }
    } catch (e) {
        failures.push(`${job.label} ${job.viewport} ${pageName} dialogs: ${(e as Error).message.split('\n')[0]}`)
    }
    await closeAll()
    await closeSidebar()
}

main().catch((e) => {
    console.error(e)
    process.exit(1)
})
