// The states of a page, in the order they depend on each other:
//
//   the page as it is
//   └─ a choice of what the whole page shows (the period of a report: "global-" controls)
//      └─ a tab
//         └─ a choice in the tab (type or grouping of a chart: the other controls)
//
// A choice of a level is shot under every choice of the level above, so the period is switched
// first and the tabs are gone through under each period. What is in a tab is shot only under the
// page as it is, or the number of files would be the product of all the levels.
import type { Locator, Page as PwPage } from 'playwright'
import { captureControls } from './controls.ts'
import { state, type Job } from './shared.ts'
import { captureTabs } from './tabs.ts'

export async function captureViews(page: PwPage, job: Job, scope: Locator, base: string, folder: string) {
    const tabs = async (name: string, local: boolean) => {
        if (state.config.tabs) await captureTabs(page, job, scope, name, folder, 'page', local)
    }
    // The page as it is: its tabs, and the choices of the tab it opens on.
    await tabs(base, true)
    if (!state.config.controls) return
    await captureControls(page, job, scope, base, folder, 'local')
    // Each other choice of the whole page, with the tabs under it.
    await captureControls(page, job, scope, base, folder, 'global', (name) => tabs(name, false))
}
