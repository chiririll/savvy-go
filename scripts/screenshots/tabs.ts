import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import type { Locator, Page as PwPage } from 'playwright'
import { captureControls } from './controls.ts'
import { log, slug, state, type Job } from './shared.ts'

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
export async function fresh(job: Job, dialog: Locator) {
    const key = await stateKey(dialog)
    job.seen ??= new Set()
    if (job.seen.has(key)) return false
    job.seen.add(key)
    return true
}

/** A shot of every choice of every tab group in scope, which is a page or an open dialog. */
export async function captureTabs(page: PwPage, job: Job, scope: Locator, base: string, folder: string, kind: 'page' | 'dialog', local = true) {
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
                state.saved++
                log(job, `  tab ${name}`)
                // A tab of a page has charts and filters of its own: their choices are taken in it.
                if (local && kind === 'page' && state.config.controls && (await group.getAttribute('role')) === 'tablist') {
                    await captureControls(page, job, scope, name, folder, 'local')
                }
            }
            // Back to the choice that was selected, for whatever comes next.
            if (start >= 0) await items.nth(start).click({ timeout: 3000 }).catch(() => {})
        }
    } catch (e) {
        state.failures.push(`${job.label} ${job.viewport} ${base} tabs: ${(e as Error).message.split('\n')[0]}`)
    }
}
