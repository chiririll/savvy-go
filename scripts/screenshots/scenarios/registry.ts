// Finds the scenarios: every file under flows/ exports its scenarios, and all of them are used. A
// new file needs no entry anywhere. Then checks them against the pages of the app.
import { readdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { pages } from '../../../resources/ts/app/pages.ts'
import { pending } from './pending.ts'
import { shotName, type Scenario } from './types.ts'

const flows = path.join(path.dirname(fileURLToPath(import.meta.url)), 'flows')

const isScenario = (v: unknown): v is Scenario =>
    typeof v === 'object' && v !== null && typeof (v as Scenario).name === 'string' && typeof (v as Scenario).run === 'function'

export async function loadScenarios(): Promise<Scenario[]> {
    const files = readdirSync(flows, { recursive: true, withFileTypes: true })
        .filter((e) => e.isFile() && e.name.endsWith('.ts') && !e.name.endsWith('.test.ts'))
        .map((e) => path.join(e.parentPath, e.name))
        .sort()
    const found = new Map<string, Scenario>()
    for (const file of files) {
        const mod: Record<string, unknown> = await import(pathToFileURL(file).href)
        for (const value of Object.values(mod)) {
            if (!isScenario(value)) continue
            // A file that imports another's scenario exports its own only, but the same object twice is fine.
            if (found.has(value.name) && found.get(value.name) !== value) throw new Error(`two scenarios are called "${value.name}"`)
            found.set(value.name, value)
        }
    }
    return [...found.values()].sort((a, b) => a.name.localeCompare(b.name))
}

/**
 * What is wrong between the scenarios and the app's pages; empty when all is well. Every page is
 * either covered by a scenario or listed in pending.ts with the reason it is not (yet), so a
 * new page cannot go by unnoticed.
 */
export function coverageProblems(scenarios: Scenario[]): string[] {
    const paths = new Set<string>(Object.values(pages).map((p) => p.path))
    const covered = new Set(scenarios.flatMap((s) => s.covers))
    const problems: string[] = []
    for (const s of scenarios) {
        for (const c of s.covers) if (!paths.has(c)) problems.push(`${s.name} covers "${c}", which is not a page of pages.ts`)
        const names = s.shots.map(shotName)
        const dup = names.find((n, i) => names.indexOf(n) !== i)
        if (dup) problems.push(`${s.name} declares the shot "${dup}" twice`)
    }
    for (const p of paths) {
        if (!covered.has(p) && !(p in pending)) problems.push(`page ${p} has no scenario: add one to scenarios/flows/, or list the page in scenarios/pending.ts with the reason`)
    }
    for (const p of Object.keys(pending)) {
        if (!paths.has(p)) problems.push(`pending.ts lists "${p}", which is not a page of pages.ts`)
        else if (covered.has(p)) problems.push(`page ${p} has a scenario now: remove it from pending.ts`)
    }
    return problems
}
