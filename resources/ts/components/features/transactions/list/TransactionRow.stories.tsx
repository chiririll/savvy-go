import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, line, payment, pendingTransaction, transaction } from '@/test/fixtures'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionRow } from './TransactionRow'

const meta = {
    title: 'features/transactions/list/TransactionRow',
    component: TransactionRow,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        transaction: transaction(),
        onEdit: fn(),
        onDelete: fn(),
        onDuplicate: fn(),
        onConfirm: fn(),
        onSkip: fn(),
    },
} satisfies Meta<typeof TransactionRow>

export default meta
type Story = StoryObj<typeof meta>

// An expense paid from one account: the lines are what was spent, the payment what left the account.
const spent = (account: (typeof accounts)[number], amount: number, overrides = {}) =>
    transaction({
        payments: [payment({ account, accountId: account.id, amount: -amount })],
        lines: [line({ amount, currencyId: account.currency?.id })],
        ...overrides,
    })

const openMenu = (canvasElement: HTMLElement) =>
    userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))

export const Expense: Story = {}
export const Income: Story = {
    args: {
        transaction: transaction({
            type: 'income',
            description: 'Salary',
            payments: [payment({ amount: 3200 })],
            lines: [line({ amount: 3200, category: categories[1], categoryId: categories[1].id })],
        }),
    },
}
export const Transfer: Story = {
    args: {
        transaction: transaction({
            type: 'transfer',
            description: null,
            payments: [
                payment({ amount: -100 }),
                payment({ account: accounts[1], accountId: accounts[1].id, amount: 100 }),
            ],
            lines: [],
        }),
    },
}

// A transfer between currencies shows what arrived, in the other currency's own precision.
export const TransferBetweenCurrencies: Story = {
    args: {
        transaction: transaction({
            type: 'transfer',
            description: null,
            payments: [
                payment({ amount: -100 }),
                payment({ account: accounts[2], accountId: accounts[2].id, amount: 15000 }),
            ],
            lines: [],
        }),
    },
}
export const DebtPayment: Story = {
    args: {
        transaction: transaction({
            type: 'debt_payment',
            description: null,
            payments: [
                payment({ amount: -50 }),
                payment({ account: { ...accounts[0], id: 9, name: 'Alex', type: 'debt' }, accountId: 9, amount: 50 }),
            ],
            lines: [],
        }),
    },
}
export const NoCategory: Story = {
    args: { transaction: transaction({ lines: [line({ category: null, categoryId: null })], description: null }) },
}
export const ThreeDecimals: Story = { args: { transaction: spent(accounts[3], 12.345) } }
export const ZeroDecimals: Story = { args: { transaction: spent(accounts[2], 1500) } }

// Several categories: two emojis in the icon, a count in the subtitle, and no notes in the row.
export const SeveralCategories: Story = {
    args: {
        transaction: transaction({
            description: 'Market trip',
            lines: [
                line({ amount: 30, note: '- Bread\n- Cheese' }),
                line({ seq: 1, amount: 12.5, category: categories[2], categoryId: categories[2].id }),
            ],
            payments: [payment({ amount: -42.5 })],
        }),
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/2 categories/)).toBeVisible()
        await expect(canvas.queryByText(/Bread/)).toBeNull()
    },
}

// Several currencies: one value in the first, with an asterisk; the exact totals appear on hover.
export const SeveralCurrencies: Story = {
    args: {
        transaction: transaction({
            description: 'Trip',
            payments: [
                payment({ amount: -30 }),
                payment({ account: accounts[1], accountId: accounts[1].id, amount: -10 }),
            ],
            lines: [
                line({ amount: 30, currencyId: currencies[0].id }),
                line({ seq: 1, amount: 10, currencyId: currencies[1].id, category: categories[2], categoryId: categories[2].id }),
            ],
        }),
    },
    play: async ({ canvasElement }) => {
        const amount = within(canvasElement).getByTestId('converted-amount')
        await expect(amount).toHaveTextContent(/\*$/)
        await userEvent.hover(amount)
        await waitFor(() => expect(screen.getAllByText(/€/).length).toBeGreaterThan(0))
    },
}

export const SeveralAccounts: Story = {
    args: {
        transaction: transaction({
            payments: [
                payment({ amount: -30 }),
                payment({ account: accounts[1], accountId: accounts[1].id, amount: -12.5 }),
            ],
            lines: [line({ amount: 42.5 })],
        }),
    },
}

export const Pending: Story = { args: { transaction: pendingTransaction() } }

// The pending occurrence of a rule with an estimate: its amount is a guess until it is confirmed.
export const PendingWithAnEstimate: Story = {
    args: { transaction: pendingTransaction(), approximate: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/≈/)).toBeVisible()
    },
}
export const Skipped: Story = { args: { transaction: transaction({ status: 'skipped' }) } }
export const SkippedTransfer: Story = {
    args: {
        transaction: transaction({
            status: 'skipped',
            type: 'transfer',
            payments: [
                payment({ amount: -100 }),
                payment({ account: accounts[2], accountId: accounts[2].id, amount: 15000 }),
            ],
            lines: [],
        }),
    },
}
export const WithDate: Story = { args: { showDate: true } }
export const LongDescription: Story = {
    args: { transaction: transaction({ description: 'A description so long that it can never fit on the single line it is given' }) },
}

// A viewer may still open a transaction to look at it, but cannot confirm, skip, duplicate or delete it.
export const ReadOnly: Story = {
    tags: ['desktop-only'],
    args: {
        isReadOnly: true,
        transaction: pendingTransaction({
            actions: { edit: true, editDetails: true, duplicate: true, delete: true, confirm: true, skip: true },
        }),
    },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await screen.findByRole('menuitem', { name: /edit/i })
        await expect(screen.queryByRole('menuitem', { name: /confirm|skip|duplicate|delete/i })).toBeNull()
    },
}

export const ActionsHidden: Story = {
    args: { showActions: false },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button', { name: /more actions/i })).toBeNull()
    },
}

// What the server allows decides what the menu offers.
export const NothingAllowed: Story = {
    args: {
        transaction: transaction({ actions: { edit: false, editDetails: false, duplicate: false, delete: false, confirm: false, skip: false } }),
    },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

// A row without actions keeps its amount in the column of the rows that have the dots.
export const NothingAllowedKeepsDotsColumn: Story = {
    tags: ['desktop-only'],
    args: {
        transaction: transaction({ actions: { edit: false, editDetails: false, duplicate: false, delete: false, confirm: false, skip: false } }),
    },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByTestId('feed-row-actions-spacer')).toBeInTheDocument()
    },
}

// A viewer has no dots on any row, so there is no column to keep.
export const ReadOnlyHasNoDotsColumn: Story = {
    tags: ['desktop-only'],
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByTestId('feed-row-actions-spacer')).toBeNull()
    },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Weekly shop'))
        await expect(args.onEdit).toHaveBeenCalledWith(args.transaction)
    },
}

export const Duplicated: Story = {
    tags: ['desktop-only'],
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /duplicate/i }))
        await expect(args.onDuplicate).toHaveBeenCalledWith(1)
    },
}

export const Deleted: Story = {
    tags: ['desktop-only'],
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1)
    },
}

export const Confirmed: Story = {
    tags: ['desktop-only'],
    args: { transaction: pendingTransaction() },
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /confirm/i }))
        await expect(args.onConfirm).toHaveBeenCalledWith(args.transaction)
    },
}

// Skipping asks first, because it cannot be taken back.
export const SkipAsksFirst: Story = {
    tags: ['desktop-only'],
    args: { transaction: pendingTransaction() },
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /skip/i }))
        const dialog = await screen.findByRole('alertdialog')
        await expect(args.onSkip).not.toHaveBeenCalled()
        await userEvent.click(within(dialog).getByRole('button', { name: /^skip$/i }))
        await waitFor(() => expect(args.onSkip).toHaveBeenCalledWith(1))
    },
}

export const SkipCancelled: Story = {
    tags: ['desktop-only'],
    args: { transaction: pendingTransaction() },
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /skip/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /cancel/i }))
        await expect(args.onSkip).not.toHaveBeenCalled()
    },
}
