// Transactions made through the form, the way a person makes them: one paid from two accounts in
// two currencies with lines of its own, an income received in parts, a transfer with a fee. Then
// the detail dialog, and the same dialog for a viewer, who can only look. The seeded space has
// such transactions already; this goes through the form that makes them.
import { expect, type Locator, type Page } from 'playwright/test'
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { api, spaceId } from '../../api.ts'
import { scenario } from '../../types.ts'
import { visit } from '../../visit.ts'
import { createSpace } from '../onboarding/create-space.ts'

const field = (page: Page, name: string) => page.locator(`[name="${name}"]`)
const mark = (page: Page, id: string) => page.getByTestId(id)

/** Picks an entry of a drop-down by what it starts with. Names of accounts are the data of the space, so they read the same in every language. */
async function choose(page: Page, trigger: Locator, name: RegExp) {
    await trigger.click()
    await page.getByRole('option', { name }).first().click()
}

/** Opens the form of a new transaction of a type, from the button in the header. */
async function openForm(page: Page, type: 'expense' | 'income' | 'transfer') {
    await page.locator('[data-testid-menu="layout-transaction"]').click()
    await mark(page, `layout-transaction-${type}`).click()
    await expect(page.getByRole('dialog')).toBeVisible()
}

async function save(page: Page) {
    await mark(page, 'form-submit').click()
    await expect(page.getByRole('dialog')).toBeHidden()
}

export const complexTransactions = scenario(
    {
        name: 'transactions/create-complex',
        covers: [],
        shots: [
            'expense-form',
            { name: 'expense-line-note', detail: true },
            'income-form',
            'transfer-form',
            { name: 'transfer-fee', detail: true },
            'list',
            'detail',
        ],
        leaves: 'the admin in "Family Budget", on its transactions, with an expense in two currencies, an income in two parts and a transfer with a fee',
    },
    async ({ page, run, shot }) => {
        await run(createSpace)
        await visit(page, pages.transactions.path)

        // An expense paid from two accounts in two currencies, in two lines: one in each.
        await openForm(page, 'expense')
        await mark(page, 'tx-description').fill('Market trip')
        await choose(page, mark(page, 'payments.0.account_id'), /Wallet/)
        await field(page, 'payments.0.amount').fill('30')
        await mark(page, 'tx-add-payments').click()
        await choose(page, mark(page, 'payments.1.account_id'), /Euro card/)
        await field(page, 'payments.1.amount').fill('20')

        await mark(page, 'tx-add-line').click()
        await mark(page, 'tx-add-line').click()
        await choose(page, mark(page, 'lines.0.category_id'), /./)
        await field(page, 'lines.0.amount').fill('30')
        await choose(page, mark(page, 'lines.1.category_id'), /./)
        await choose(page, mark(page, 'lines.1.currency_id'), /^EUR/)
        await field(page, 'lines.1.amount').fill('20')
        await shot('expense-form')

        // From two lines on a line has a note, in markdown.
        await mark(page, 'lines.1.note').fill('- Tram day pass ×2\n- Castle tickets')
        await shot('expense-line-note')
        await save(page)
        await expect(page.getByText('Market trip').first()).toBeVisible()

        // An income that came in two parts.
        await openForm(page, 'income')
        await mark(page, 'tx-description').fill('Bonus')
        await choose(page, mark(page, 'payments.0.account_id'), /Wallet/)
        await field(page, 'payments.0.amount').fill('800')
        await mark(page, 'tx-add-payments').click()
        await choose(page, mark(page, 'payments.1.account_id'), /Euro card/)
        await field(page, 'payments.1.amount').fill('200')
        await shot('income-form')
        await save(page)

        // A transfer between currencies, with the bank's fee on top.
        await openForm(page, 'transfer')
        await mark(page, 'tx-description').fill('Top up the euro card')
        await choose(page, mark(page, 'payments.0.account_id'), /Wallet/)
        await field(page, 'payments.0.amount').fill('110')
        await choose(page, mark(page, 'to_account_id'), /Euro card/)
        const received = field(page, 'to_amount')
        if (await received.isEditable()) await received.fill('100')
        await shot('transfer-form')
        await mark(page, 'tx-add-fee').click()
        await choose(page, mark(page, 'fee.category_id'), /./)
        await field(page, 'fee.amount').fill('2')
        await shot('transfer-fee')
        await save(page)

        for (const text of ['Market trip', 'Bonus', 'Top up the euro card']) await expect(page.getByText(text).first()).toBeVisible()
        await shot('list')

        // A row opens the transaction: every payment and line, notes in markdown.
        await page.getByText('Market trip').first().click()
        await expect(page.getByRole('dialog')).toBeVisible()
        await expect(page.getByRole('dialog').getByText('Castle tickets')).toBeVisible()
        await shot('detail')
    },
)

export const viewerTransactions = scenario(
    {
        name: 'transactions/viewer',
        covers: [],
        shots: ['list', 'detail'],
        leaves: 'a viewer of "Family Budget", on its transactions, with a transaction open',
    },
    async ({ page, run, shot }) => {
        await run(complexTransactions)

        // A viewer joins by a link and may read the space, nothing more.
        const space = await spaceId(page)
        const invitation = await api<{ token: string }>(page, 'POST', `/spaces/${space}/invitations`, { role: 'viewer', note: 'Kim' })
        await api(page, 'POST', '/auth/logout')
        await page.goto(`/invite/${invitation.token}`)
        await page.locator('#name').fill('Kim Lee')
        await page.locator('#email').fill('kim@example.com')
        await page.locator('#password').fill('correct-horse-battery')
        await page.locator('form button[type="submit"]').click()
        await expect(page).not.toHaveURL(/\/invite\//)

        await visit(page, pages.transactions.path)
        await expect(page.getByText('Market trip').first()).toBeVisible()
        // No button to make a transaction, no menu on a row.
        await expect(page.locator('[data-testid-menu="layout-transaction"]')).toHaveCount(0)
        await shot('list')

        await page.getByText('Market trip').first().click()
        const dialog = page.getByRole('dialog')
        await expect(dialog).toBeVisible()
        await expect(dialog.getByText('Castle tickets')).toBeVisible()
        // Nothing to change in it: no Edit, Skip, Delete or Confirm.
        await expect(dialog.getByRole('button', { name: /edit|delete|skip|confirm|измен|удал|пропуст|подтверд/i })).toHaveCount(0)
        await shot('detail')
    },
)

/**
 * "Transfers needing attention" on a space's settings: a transfer between two linked spaces waits
 * for a decision once one of them is restored from a backup and differs from the other. Made here
 * the way it happens: a transfer, a backup, then a change in the other space and a new transfer,
 * then the restore.
 */
export const transfersInReview = scenario(
    {
        name: 'transactions/transfers-in-review',
        covers: [],
        shots: ['review', 'accepted'],
        leaves: 'the admin in "Family Budget", on its settings, with one transfer left to decide',
    },
    async ({ page, run, shot }) => {
        await run(createSpace)
        const home = await spaceId(page)
        const other = await api<{ id: number }>(page, 'POST', '/spaces', {
            name: 'Pets',
            baseCurrency: 'USD',
            accounts: [{ name: 'Cash', type: 'cash', currency: 'USD' }],
        })
        const accountOf = async (space: number) => (await api<{ id: number }[]>(page, 'GET', `/spaces/${space}/accounts`))[0].id
        const [from, to] = [await accountOf(home), await accountOf(other.id)]
        const send = (amount: string, description: string) =>
            api<{ uuid: string }>(page, 'POST', `/spaces/${home}/transfers`, {
                to_space_id: other.id, from_account_id: from, to_account_id: to,
                from_amount: amount, to_amount: amount, date: new Date().toISOString().slice(0, 10), description,
            })

        await api(page, 'POST', `/spaces/${home}/links`, { target_space_id: other.id })
        const changed = await send('20', 'Vet visit')
        const backup = await api<{ filename: string }>(page, 'POST', `/spaces/${home}/backups`, {})

        // After the backup: one transfer is changed from the other side and one is new. The restore
        // brings "Family Budget" back to the backup, so both differences wait for a decision.
        await api(page, 'PATCH', `/spaces/${other.id}/transfers/${changed.uuid}`, { from_amount: '25', to_amount: '25' })
        await send('40', 'Dog food')
        await api(page, 'POST', `/spaces/${home}/backups/${backup.filename}/restore`)

        await visit(page, pages.spaceSettings.path)
        const items = page.locator('main li').filter({ has: page.locator('table') })
        await expect(items).toHaveCount(2)
        await shot('review')

        // Accepting the first takes the other space's version; one is left.
        await items.first().getByRole('button').first().click()
        await expect(items).toHaveCount(1)
        await shot('accepted')
    },
)
