// What a scenario is. A scenario drives one flow of the app on a server of its own, checks what
// it shows and takes the shots it declares. See docs/scripts.md.
import type { Locator, Page } from 'playwright'
import type { Server } from './server.ts'

export interface Ctx {
    page: Page
    /** The app under test: started for this scenario, empty (no users) unless the scenario made it otherwise. */
    server: Server
    language: string
    theme: string
    viewport: string
    /**
     * Takes the shot called name, which the scenario must have declared in `shots`. `mask` hides
     * what differs from run to run (a generated password) behind a box. Toasts are hidden (they
     * linger with animations off); `toasts: true` keeps them, for a shot that is about one.
     */
    shot(name: string, options?: { mask?: Locator[]; toasts?: boolean }): Promise<void>
    /**
     * Runs another scenario first, on this server and page: its steps become the start of this one.
     * Its shots are not taken again (it takes them when it runs on its own); `shots: true` takes them.
     */
    run(other: Scenario, options?: { shots?: boolean }): Promise<void>
}

/**
 * A shot a scenario takes: its name, or `{ name, detail: true }` for one that only the `full`
 * detail level takes (an error state, a variant). At `key` it is left out, but the steps
 * around it still run and assert.
 */
export type ShotDecl = string | { name: string; detail: true }

export const shotName = (s: ShotDecl) => (typeof s === 'string' ? s : s.name)
export const isDetail = (s: ShotDecl) => typeof s !== 'string'

export interface Scenario {
    /** Unique, with the area first: "onboarding/admin-setup". Also the directory of its shots. */
    name: string
    /** The pages of resources/ts/app/pages.ts (their `path`) that the scenario goes through. */
    covers: string[]
    /** Every shot it takes, in order (the number in the file name is the place in this list, whatever the detail level). A shot that is not here, or one that is here and never taken, fails the run. */
    shots: ShotDecl[]
    /** What the scenario leaves behind, for the ones that run it first: "admin signed in, no spaces". */
    leaves?: string
    run(ctx: Ctx): Promise<void>
}

export type ScenarioDef = Omit<Scenario, 'run'>

/** Declares a scenario; `run` asserts as it goes (a failed assertion fails the run). */
export function scenario(def: ScenarioDef, run: Scenario['run']): Scenario {
    return { ...def, run }
}
