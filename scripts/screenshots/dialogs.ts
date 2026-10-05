import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import type { Locator, Page as PwPage } from 'playwright'
import { captureTabs, fresh } from './tabs.ts'
import { log, state, type Job } from './shared.ts'

// Dialogs are found by the marks the app puts on what opens them (resources/ts/lib/test-id.ts),
// which exist only in a build made with APP_ENV=screenshots. Another build has none and gets no shots.
//   data-testid="id"       a button or menu item that opens a dialog
//   data-testid-menu="id"  the button of a menu with such items; the first rows' menus are opened
// An id starting with "layout-" belongs to the page frame and is taken once per job.
const MENUS_PER_PAGE = 5

// The sidebar is a dialog too, but it is not what is being looked for.
const dialogSelector = '[role="dialog"]:not([data-sidebar]), [role="alertdialog"]'

export async function captureDialogs(page: PwPage, job: Job, pageName: string, dir: string, frameDone: Set<string>) {
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
        state.saved++
        take(id)
        log(job, `  dialog ${name}`)
        if (state.config.tabs) await captureTabs(page, job, page.locator(dialogSelector).first(), name, dialogs, 'dialog')
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
        state.failures.push(`${job.label} ${job.viewport} ${pageName} dialogs: ${(e as Error).message.split('\n')[0]}`)
    }
    await closeAll()
    await closeSidebar()
}
