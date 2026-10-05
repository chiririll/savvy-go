import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import type { Locator, Page as PwPage } from 'playwright'
import { log, slug, state, type Job } from './shared.ts'

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
    state.saved++
    log(job, `  control ${name}`)
}

/** Takes the page with each choice of each marked group in scope; `top` is the page as it is, not a tab of it. */
export async function captureControls(page: PwPage, job: Job, scope: Locator, base: string, folder: string, top: boolean) {
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
        state.failures.push(`${job.label} ${job.viewport} ${base} controls: ${(e as Error).message.split('\n')[0]}`)
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
