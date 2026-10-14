// Runs the scenarios: each one, for each language, theme and viewport of the config, on an app and
// a browser context of its own, so nothing one leaves behind reaches another.
import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { type Browser, type Page } from 'playwright'
import type { Config, Detail } from '../config.ts'
import { newContext } from '../context.ts'
import { slug } from '../shared.ts'
import { coverageProblems, loadScenarios } from './registry.ts'
import { binary, startServer, type Server } from './server.ts'
import { isDetail, shotName, type Ctx, type Scenario } from './types.ts'

export interface Options {
    config: Config
    /** The run's output directory; the shots go to <outDir>/scenarios. */
    outDir: string
    concurrency: number
    /** The app's binary (SAVVY_BIN); empty: see server.ts. */
    binary: string
}

interface Variant {
    language: string
    theme: string
    viewport: string
}

/** How many shots were taken. */
export const shotsTaken = { shots: 0 }

/** Returns what failed, one line each; empty when all went well. */
export async function runScenarios(browser: Browser, options: Options): Promise<string[]> {
    const all = await loadScenarios()
    // The check is on all of them, whatever the config picks, so a filtered run cannot hide a gap.
    const problems = coverageProblems(all)
    if (problems.length) return problems.map((p) => `coverage: ${p}`)

    const names = options.config.scenarioNames
    const wanted = all.filter((s) => !names.length || names.some((n) => s.name === n || s.name.startsWith(n)))
    if (names.length && !wanted.length) return [`no scenario matches ${names.join(', ')}; there are ${all.map((s) => s.name).join(', ')}`]

    // The scenarios have their own lists when the config gives them, so pages and flows can differ in how many variants they take.
    const { config } = options
    const languages = config.scenarioLanguages.length ? config.scenarioLanguages : config.languages
    const themes = config.scenarioThemes.length ? config.scenarioThemes : config.themes
    const viewports = config.scenarioViewports.length ? config.scenarioViewports : config.viewports
    const variants: Variant[] = languages.flatMap((language) => themes.flatMap((theme) => viewports.map((viewport) => ({ language, theme, viewport }))))
    const jobs = wanted.flatMap((sc) => variants.map((v) => ({ sc, v })))
    if (!jobs.length) return []

    const bin = binary(options.binary)
    console.log(`scenarios: ${wanted.length} scenarios, ${jobs.length} runs; ${options.concurrency} at a time`)
    const failures: string[] = []
    const queue = [...jobs]
    await Promise.all(
        Array.from({ length: options.concurrency }, async () => {
            for (let job = queue.shift(); job; job = queue.shift()) {
                const label = `${job.sc.name} ${job.v.language}-${job.v.theme} ${job.v.viewport}`
                try {
                    await runOne(browser, bin, options, job.sc, job.v)
                    console.log(`[scenario] ${label} ok`)
                } catch (e) {
                    console.error(`[scenario] ${label} FAILED`)
                    failures.push(`${label}: ${(e as Error).message}`)
                }
            }
        }),
    )
    return failures
}

async function runOne(browser: Browser, bin: string, options: Options, sc: Scenario, v: Variant) {
    const shotsRoot = path.join(options.outDir, 'scenarios')
    const dir = shotDir(shotsRoot, sc, v)
    const server = await startServer(bin, slug(`${sc.name}-${v.language}-${v.theme}-${v.viewport}`))
    const context = await newContext(browser, { baseURL: server.url, ...v })
    const page = await context.newPage()
    try {
        const ctx = makeCtx(page, server, v, shotsRoot, sc, true, new Set(), options.config.detail)
        await sc.run(ctx)
        const missing = sc.shots.map(shotName).filter((s) => !(ctx as WithTaken).taken.has(s))
        if (missing.length) throw new Error(`declared but not taken: ${missing.join(', ')}`)
    } catch (e) {
        await mkdir(dir, { recursive: true })
        await page.screenshot({ path: path.join(dir, 'FAILED.png'), fullPage: true }).catch(() => {})
        const log = server.log().trim().split('\n').slice(-15).join('\n')
        throw new Error(`${(e as Error).message.split('\n').slice(0, 12).join('\n')}\n  at ${page.url()}\n  app log:\n${log}`)
    } finally {
        await context.close()
        await server.stop()
    }
}

type WithTaken = Ctx & { taken: Set<string> }

const shotDir = (shotsRoot: string, sc: Scenario, v: Variant) => path.join(shotsRoot, sc.name, `${v.language}-${v.theme}`, v.viewport)

function makeCtx(page: Page, server: Server, v: Variant, shotsRoot: string, sc: Scenario, record: boolean, taken: Set<string>, detail: Detail): WithTaken {
    const dir = shotDir(shotsRoot, sc, v)
    return {
        page, server, ...v, taken,
        async shot(name, o) {
            if (!record) return
            const at = sc.shots.findIndex((d) => shotName(d) === name)
            if (at < 0) throw new Error(`${sc.name}: the shot "${name}" is not declared in its shots`)
            // Left out at this detail level; counted as taken, since it was not wanted.
            if (detail === 'key' && isDetail(sc.shots[at])) {
                taken.add(name)
                return
            }
            await mkdir(dir, { recursive: true })
            // Let the data arrive, but do not wait for a page that never goes quiet.
            await page.waitForLoadState('networkidle', { timeout: 3000 }).catch(() => {})
            // A full-page shot resizes the window, which closes an open dialog: dialogs are shot as seen.
            const dialog = (await page.locator('[role="dialog"]:visible').count()) > 0
            await page.screenshot({
                path: path.join(dir, `${String(at + 1).padStart(2, '0')}-${name}.png`),
                fullPage: !dialog, animations: 'disabled', mask: o?.mask, maskColor: '#9ca3af',
                style: o?.toasts ? undefined : '[data-sonner-toaster] { display: none !important }',
            })
            taken.add(name)
            shotsTaken.shots++
        },
        async run(other, o) {
            const shots = o?.shots ?? false
            const childTaken = new Set<string>()
            await other.run(makeCtx(page, server, v, shotsRoot, other, shots, childTaken, detail))
            if (shots) {
                const missing = other.shots.map(shotName).filter((s) => !childTaken.has(s))
                if (missing.length) throw new Error(`${other.name}: declared but not taken: ${missing.join(', ')}`)
            }
        },
    }
}
