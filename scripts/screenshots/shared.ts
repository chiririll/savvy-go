// What the parts of the screenshot script share: the unit of work, the counters and the log.
import type { BrowserContextOptions } from 'playwright'
import type { Config } from './config.ts'

export const slug = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

export interface Job {
    label: string
    viewport: string
    language: string
    theme: string
    dir: string
    // The session, space and pages to capture.
    storageState?: BrowserContextOptions['storageState']
    spaceId?: number
    pages: { path: string; name: string; sidebar?: boolean }[]
    /** What was shot in this run, by what was on the screen (see stateKey in tabs.ts). */
    seen?: Set<string>
}

export const state = {
    config: undefined as unknown as Config,
    failures: [] as string[],
    saved: 0,
    done: 0,
    total: 0,
}

// Runs go side by side, so every line says which one it is from.
export const log = (job: Job, what: string) => console.log(`[${job.label} ${job.viewport}] ${what}`)
