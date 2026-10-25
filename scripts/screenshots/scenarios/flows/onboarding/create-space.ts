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
        shots: ['name', 'currencies', { name: 'currencies-no-account', detail: true }, { name: 'currencies-invalid', detail: true }, 'currencies-filled', 'categories', 'tags', 'confirm', 'done', 'members'],
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
        // The list searches by name, not only by code, and is capped.
        await base.click()
        await page.getByTestId('currency-search').fill('yen')
        await expect(page.getByRole('option', { name: /^JPY/ })).toBeVisible()
        await expect(page.getByRole('option')).toHaveCount(1)
        await page.getByTestId('currency-search').fill('')
        expect(await page.getByRole('option').count()).toBeLessThanOrEqual(50)
        await page.keyboard.press('Escape')
        await choose(page, base, /^USD/)
        await shot('currencies')

        // A space needs an account: with none, the way on stays shut (there is no message about it).
        await expect(page.getByTestId('create-space-next')).toBeDisabled()
        await expect(page.getByTestId('create-space-add-account')).toBeVisible()
        await shot('currencies-no-account')

        // An account with no name cannot be created: the way on stays shut.
        await page.getByTestId('create-space-add-account').click()
        await expect(page.getByTestId('create-space-next')).toBeDisabled()
        await shot('currencies-invalid')

        await page.getByTestId('create-space-account-name').fill('Wallet')
        await page.getByTestId('create-space-account-balance').fill('100')
        await choose(page, page.getByTestId('create-space-add-currency'), /^EUR/)
        await page.getByTestId('create-space-add-account').click()
        await page.getByTestId('create-space-account-name').last().fill('Euro card')
        await choose(page, page.getByTestId('create-space-account-currency').last(), /^EUR/)
        await page.getByTestId('create-space-account-balance').last().fill('250')
        await expect(page.getByTestId('create-space-next')).toBeEnabled()
        await shot('currencies-filled')

        // Categories and tags come next; neither blocks the way on.
        await page.getByTestId('create-space-next').click()
        await expect(page.getByRole('textbox', { name: /new category|новая категория/i })).toBeVisible()
        await shot('categories')
        await page.getByTestId('create-space-next').click()
        await expect(page.getByRole('textbox', { name: /new tag|новый тег/i })).toBeVisible()
        await shot('tags')
        await page.getByTestId('create-space-next').click()
        await expect(page.getByTestId('create-space-submit')).toBeVisible()
        await shot('confirm')
        await page.getByTestId('create-space-submit').click()
        await expect(page.getByTestId('create-space-done')).toBeVisible()
        await shot('done')

        // Inviting is optional and not a step: the button opens the list. Continue with it empty sends nothing.
        // This app has no mail, so the form offers a link.
        await page.getByTestId('create-space-invite').click()
        await expect(page.getByTestId('invite-add')).toBeVisible()
        await shot('members')

        // Continue switches to the new space.
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-done')).toBeHidden()
        await expect(page.locator('[data-sidebar="trigger"]').first()).toBeAttached()
    },
)

/** From the currencies step through the categories and tags steps (nothing to change there) to the summary. */
async function skipToSummary(page: Page) {
    for (let i = 0; i < 3; i++) await page.getByTestId('create-space-next').click()
}

/** Adds an account to the draft, as every space needs one. */
async function addAccount(page: Page, name: string) {
    await page.getByTestId('create-space-add-account').click()
    await page.getByTestId('create-space-account-name').last().fill(name)
}

/** A second space: the modal now offers to copy from the first. */
export const createSpaceCopy = scenario(
    {
        name: 'onboarding/create-space-copy',
        covers: [],
        shots: ['copy-source', 'copy-kinds', 'currencies-copied', 'confirm'],
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
        await shot('currencies-copied')

        await skipToSummary(page)
        await expect(page.getByTestId('create-space-submit')).toBeVisible()
        await shot('confirm')
        await page.getByTestId('create-space-submit').click()
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-done')).toBeHidden()
    },
)

/** What the server says about how invitations may be made; the form differs for each. */
interface Methods {
    email: boolean
    link: boolean
    emailReason?: string
}

/**
 * Makes the app answer GET /invitation-methods with `methods` and reloads, which is when the form asks
 * again. The server of a scenario has no mail, so the combination is staged here; what the server
 * allows is checked by its own tests.
 */
async function serverAllows(page: Page, methods: Methods) {
    await page.unroute('**/api/invitation-methods')
    await page.route('**/api/invitation-methods', (route) => route.fulfill({ json: { data: methods } }))
    await page.reload()
    await expect(page.locator('[data-sidebar="trigger"]').first()).toBeAttached()
}

/**
 * Makes one more space through the modal, as the one who is signed in and has a space already. The
 * catalog is priced in the current space's base currency and leaves it out, so `currency` must differ.
 */
async function makeSpace(page: Page, viewport: string, name: string, currency: 'USD' | 'EUR') {
    // On a phone the sidebar, and the space switcher in it, is a drawer.
    if (viewport === 'mobile') await page.locator('[data-sidebar="trigger"]').first().click()
    await page.locator('[data-testid-menu="layout-space"]').click()
    await page.getByTestId('layout-space-create').click()
    await page.locator('#space-name').fill(name)
    await page.getByTestId('create-space-next').click()
    // Another space to copy from: the step is passed with the defaults.
    await page.getByTestId('create-space-next').click()
    const base = page.getByTestId('create-space-base')
    await expect(base).toBeEnabled({ timeout: 20_000 })
    // The list can fill in while the select opens and close it again: open it again until the entry is there.
    await expect(async () => {
        if (!(await page.getByRole('listbox').isVisible())) await base.click()
        await page.getByRole('option', { name: new RegExp(`^${currency}`) }).click({ timeout: 3_000 })
    }).toPass({ timeout: 30_000 })
    await addAccount(page, 'Wallet')
    await skipToSummary(page)
    await page.getByTestId('create-space-submit').click()
    await expect(page.getByTestId('create-space-invite')).toBeVisible()
}

/**
 * The optional invitations on the screen after the space is made. They go out when the screen is left
 * with Continue; a link has to be read then, so the screen stays with the outcomes. The form asks the
 * server how invitations may be made once per page load, and the list does not outlive a reload, so
 * each combination gets a space of its own after a reload.
 */
export const createSpaceMembers = scenario(
    {
        name: 'onboarding/create-space-members',
        covers: [],
        shots: [
            'confirm',
            'done',
            'link',
            'link-added',
            'results',
            'email',
            { name: 'email-invalid', detail: true },
            'email-added',
            'both-email',
            { name: 'both-link', detail: true },
            { name: 'unavailable', detail: true },
        ],
        leaves: 'four spaces; the last ("Neither") is the current one; invitations were tried in the first (a link, made) and the second (an address, refused: no mail)',
    },
    async ({ page, viewport, run, shot }) => {
        await run(setup2faSkipped)

        await page.locator('#space-name').fill('Family Budget')
        await page.getByTestId('create-space-next').click()
        const base = page.getByTestId('create-space-base')
        await expect(base).toBeEnabled({ timeout: 20_000 })
        await choose(page, base, /^USD/)
        await addAccount(page, 'Wallet')
        await skipToSummary(page)
        await expect(page.getByTestId('create-space-submit')).toBeVisible()
        await shot('confirm')
        await page.getByTestId('create-space-submit').click()

        // No mail on this server: a note, optional, where the address would be.
        await expect(page.getByTestId('create-space-invite')).toBeVisible()
        await expect(page.getByTestId('invite-add')).toHaveCount(0)
        await shot('done')
        await page.getByTestId('create-space-invite').click()
        await expect(page.getByTestId('invite-note')).toBeVisible()
        await expect(page.getByTestId('invite-email')).toHaveCount(0)
        await shot('link')
        await page.getByTestId('invite-note').fill('Kim from work')
        await page.getByTestId('invite-add').click()
        await expect(page.getByTestId('invite-note')).toHaveValue('')
        await expect(page.getByText('Kim from work')).toBeVisible()
        await shot('link-added')

        // Continue sends it. The link has to be copied, so the screen stays with the outcome.
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-invite')).toHaveCount(0)
        await expect(page.getByText('Kim from work')).toBeVisible()
        await shot('results')
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-done')).toBeHidden()

        // Mail on, links not allowed (addresses must be confirmed): the address is the only way.
        await serverAllows(page, { email: true, link: false })
        await makeSpace(page, viewport, 'By email', 'EUR')
        await page.getByTestId('create-space-invite').click()
        await expect(page.getByTestId('invite-email')).toBeVisible()
        await expect(page.getByTestId('invite-note')).toHaveCount(0)
        await expect(page.getByTestId('invite-method-link')).toHaveCount(0)
        await shot('email')
        await page.getByTestId('invite-email').fill('not-an-address')
        await page.getByTestId('invite-add').click()
        await expect(page.getByTestId('invite-email')).toHaveAttribute('aria-invalid', 'true')
        await shot('email-invalid')
        await page.getByTestId('invite-email').fill('lee@example.com')
        await page.getByTestId('invite-add').click()
        await expect(page.getByText('lee@example.com')).toBeVisible()
        await shot('email-added')
        // The address cannot be mailed here (no mail): the refusal is shown with the outcomes.
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-invite')).toHaveCount(0)
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-done')).toBeHidden()

        // Both allowed: the choice between them.
        await serverAllows(page, { email: true, link: true })
        await makeSpace(page, viewport, 'Both', 'USD')
        await page.getByTestId('create-space-invite').click()
        await expect(page.getByTestId('invite-method-email')).toHaveAttribute('aria-pressed', 'true')
        await shot('both-email')
        await page.getByTestId('invite-method-link').click()
        await expect(page.getByTestId('invite-note')).toBeVisible()
        await shot('both-link')
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-done')).toBeHidden()

        // Neither: nothing to add with.
        await serverAllows(page, { email: false, link: false })
        await makeSpace(page, viewport, 'Neither', 'EUR')
        await page.getByTestId('create-space-invite').click()
        await expect(page.getByTestId('invite-add')).toHaveCount(0)
        await shot('unavailable')
        await page.getByTestId('create-space-done').click()
        await expect(page.getByTestId('create-space-done')).toBeHidden()
    },
)
