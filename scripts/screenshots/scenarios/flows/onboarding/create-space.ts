// Flow 3 (docs/todo-onboarding/07-create-space-modal.md): the modal that builds a filled space in
// steps. The first space has no other to copy from; the second one does (create-space-copy).
import { expect, type Locator, type Page } from 'playwright/test'
import { scenario } from '../../types.ts'
import { setup2faSkipped } from '../auth/setup-2fa.ts'

/** Picks an entry of a drop-down (Radix select) by what it starts with; codes such as "USD" read the same in every language. */
async function choose(page: Page, trigger: Locator, startsWith: RegExp) {
    await trigger.click()
    await page.getByRole('option', { name: startsWith }).click()
}

export const createSpace = scenario(
    {
        name: 'onboarding/create-space',
        covers: [],
        shots: ['name', 'entities', { name: 'entities-invalid', detail: true }, 'entities-filled', 'confirm', 'done'],
        leaves: 'a space "Family Budget" (USD, EUR, two accounts) is the current one; on its dashboard',
    },
    async ({ page, run, shot }) => {
        await run(setup2faSkipped)

        const name = page.locator('#space-name')
        await expect(name).toBeVisible()
        await shot('name')
        await name.fill('Family Budget')
        await page.getByTestId('create-space-next').click()

        // The currency list comes from the server's data service: may take a while.
        const base = page.getByTestId('create-space-base')
        await expect(base).toBeEnabled({ timeout: 20_000 })
        await choose(page, base, /^USD/)
        await shot('entities')

        // An account with no name cannot be created: the way on stays shut.
        await page.getByTestId('create-space-add-account').click()
        await expect(page.getByTestId('create-space-next')).toBeDisabled()
        await shot('entities-invalid')

        await page.getByTestId('create-space-account-name').fill('Wallet')
        await page.getByTestId('create-space-account-balance').fill('100')
        await choose(page, page.getByTestId('create-space-add-currency'), /^EUR/)
        await page.getByTestId('create-space-add-account').click()
        await page.getByTestId('create-space-account-name').last().fill('Euro card')
        await choose(page, page.getByTestId('create-space-account-currency').last(), /^EUR/)
        await page.getByTestId('create-space-account-balance').last().fill('250')
        await expect(page.getByTestId('create-space-next')).toBeEnabled()
        await shot('entities-filled')

        await page.getByTestId('create-space-next').click()
        await expect(page.getByTestId('create-space-submit')).toBeVisible()
        await shot('confirm')
        await page.getByTestId('create-space-submit').click()
        await expect(page.getByTestId('create-space-done')).toBeVisible()
        await shot('done')

        // Done switches to the new space.
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-done')).toBeHidden()
        await expect(page.locator('[data-sidebar="trigger"]').first()).toBeAttached()
    },
)

/** A second space: the modal now offers to copy from the first. */
export const createSpaceCopy = scenario(
    {
        name: 'onboarding/create-space-copy',
        covers: [],
        shots: ['copy-source', 'copy-kinds', 'entities-copied', 'confirm'],
        leaves: 'two spaces; the second ("Holiday") is the current one',
    },
    async ({ page, viewport, run, shot }) => {
        await run(createSpace)

        // On a phone the sidebar, and the space switcher in it, is a drawer.
        if (viewport === 'mobile') await page.locator('[data-sidebar="trigger"]').first().click()
        await page.locator('[data-testid-menu="layout-space"]').click()
        await page.getByTestId('layout-space-create').click()

        await page.locator('#space-name').fill('Holiday')
        await page.getByTestId('create-space-next').click()

        // With another space to copy from, there is a step for that before the entities.
        const source = page.getByRole('combobox')
        await expect(source).toBeVisible()
        await shot('copy-source')
        await source.click()
        await page.getByRole('option', { name: 'Family Budget' }).click()
        // Nothing is copied unless ticked: take the currencies and the accounts (without their balances).
        const kinds = page.getByRole('checkbox')
        await expect(kinds).toHaveCount(4)
        await kinds.nth(0).click()
        await kinds.nth(1).click()
        await expect(kinds.nth(1)).toBeChecked()
        await shot('copy-kinds')

        await page.getByTestId('create-space-next').click()
        // The accounts of the first space come along, with no balance; its base currency is the base here too.
        await expect(page.getByTestId('create-space-account-name').first()).toHaveValue('Wallet', { timeout: 20_000 })
        await expect(page.getByTestId('create-space-account-balance').first()).toHaveValue('')
        await expect(page.getByTestId('create-space-account-name')).toHaveCount(2)
        await shot('entities-copied')

        await page.getByTestId('create-space-next').click()
        await shot('confirm')
        await page.getByTestId('create-space-submit').click()
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-done')).toBeHidden()
    },
)
