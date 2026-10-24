import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, pendingTransaction, transaction } from '@/test/fixtures'
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

const items = [
    { id: 1, name: 'Bread', quantity: 2, pricePerUnit: 2.5, totalPrice: 5 },
    { id: 2, name: 'Cheese', quantity: 1, pricePerUnit: 12.5, totalPrice: 12.5 },
]

const openMenu = (canvasElement: HTMLElement) =>
    userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))

export const Expense: Story = {}
export const Income: Story = {
    args: { transaction: transaction({ type: 'income', amount: 3200, description: 'Salary', category: categories[1] }) },
}
export const Transfer: Story = {
    args: {
        transaction: transaction({ type: 'transfer', description: null, category: null, toAccount: accounts[0], account: accounts[0] }),
    },
}

// A transfer between currencies shows what arrived, in the other currency's own precision.
export const TransferBetweenCurrencies: Story = {
    args: {
        transaction: transaction({
            type: 'transfer', description: null, category: null, amount: 100, account: accounts[0],
            toAccount: accounts[2], toAmount: 15000,
        }),
    },
}
export const DebtPayment: Story = { args: { transaction: transaction({ type: 'debt_payment', description: null, category: null }) } }
export const DebtCollection: Story = { args: { transaction: transaction({ type: 'debt_collection', description: null, category: null }) } }
export const NoCategory: Story = { args: { transaction: transaction({ category: null, description: null }) } }
export const Estimated: Story = { args: { transaction: transaction({ isEstimated: true }) } }
export const ThreeDecimals: Story = { args: { transaction: transaction({ amount: 12.345, account: accounts[3] }) } }
export const ZeroDecimals: Story = { args: { transaction: transaction({ amount: 1500, account: accounts[2] }) } }

export const Pending: Story = { args: { transaction: pendingTransaction() } }
export const Skipped: Story = { args: { transaction: transaction({ status: 'skipped' }) } }
export const SkippedTransfer: Story = {
    args: {
        transaction: transaction({
            status: 'skipped', type: 'transfer', category: null, toAccount: accounts[2], toAmount: 15000, amount: 100,
        }),
    },
}
export const WithDate: Story = { args: { showDate: true } }
export const LongDescription: Story = {
    args: { transaction: transaction({ description: 'A description so long that it can never fit on the single line it is given' }) },
}

export const WithItems: Story = {
    args: { transaction: transaction({ items }) },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const toggle = canvas.getByRole('button', { name: /^\(2/ })
        await expect(toggle).toHaveAttribute('aria-expanded', 'false')
        await userEvent.click(toggle)
        await expect(toggle).toHaveAttribute('aria-expanded', 'true')
        await expect(canvas.getByText('Cheese')).toBeVisible()
    },
}

export const SingleItemHasNoToggle: Story = {
    args: { transaction: transaction({ items: [items[0]] }) },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button', { name: /^\(1/ })).toBeNull()
    },
}

// A viewer may still open a transaction to look at it, but cannot confirm, skip, duplicate or delete it.
export const ReadOnly: Story = {
    tags: ['desktop-only'],
    args: { isReadOnly: true, transaction: pendingTransaction({ actions: { edit: true, duplicate: true, delete: true, confirm: true, skip: true } }) },
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
        transaction: transaction({ actions: { edit: false, duplicate: false, delete: false, confirm: false, skip: false } }),
    },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
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
