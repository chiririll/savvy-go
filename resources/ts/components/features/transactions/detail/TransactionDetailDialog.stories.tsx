import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { accounts, categories, currencies, line, payment, pendingTransaction, tags, transaction } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionDetailDialog } from './TransactionDetailDialog'

const meta = {
    title: 'features/transactions/detail/TransactionDetailDialog',
    component: TransactionDetailDialog,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        transaction: transaction(),
        open: true,
        onOpenChange: fn(),
        onEdit: fn(),
        onConfirm: fn(),
        onSkip: fn(),
        onDelete: fn(),
    },
} satisfies Meta<typeof TransactionDetailDialog>

export default meta
type Story = StoryObj<typeof meta>

const dialog = async () => within(await screen.findByRole('dialog'))

export const Expense: Story = {
    play: async () => {
        const view = await dialog()
        await expect(view.getByText('Weekly shop')).toBeVisible()
        await expect(view.getByText('Payments')).toBeVisible()
        await expect(view.getByRole('button', { name: 'Edit' })).toBeVisible()
    },
}

export const LinesWithNotesAndChange: Story = {
    args: {
        transaction: transaction({
            payments: [payment({ amount: -50 }), payment({ account: accounts[1], accountId: accounts[1].id, amount: 7.5 })],
            lines: [
                line({ seq: 0, amount: 30, note: '- milk\n- **eggs**' }),
                line({ seq: 1, categoryId: categories[1].id, category: categories[1], amount: 12.5 }),
            ],
            tags: [tags[0]],
        }),
    },
    play: async () => {
        const view = await dialog()
        await expect(view.getByText('Change')).toBeVisible()
        await expect(await view.findByText('milk')).toBeVisible()
        await expect(view.getByText(`#${tags[0].name}`)).toBeVisible()
    },
}

export const Pending: Story = {
    args: { transaction: pendingTransaction({ recurringTransactionId: 3 }) },
    play: async ({ args }) => {
        const view = await dialog()
        await expect(view.getByText('Recurring')).toBeVisible()
        await userEvent.click(view.getByRole('button', { name: 'Confirm' }))
        await expect(args.onConfirm).toHaveBeenCalledOnce()
    },
}

export const Skipped: Story = {
    args: {
        transaction: transaction({
            status: 'skipped',
            actions: { edit: false, editDetails: false, duplicate: false, delete: false, confirm: false, skip: false },
        }),
    },
    play: async () => {
        const view = await dialog()
        await expect(view.getByText('Skipped')).toBeVisible()
        await expect(view.queryByRole('button', { name: 'Edit' })).toBeNull()
    },
}

export const Restricted: Story = {
    args: {
        transaction: transaction({
            difference: 2.6,
            restrictions: [{ reason: 'debt', message: 'debt' }],
            actions: { edit: false, editDetails: true, duplicate: false, delete: false, confirm: false, skip: false },
        }),
    },
    play: async () => {
        const view = await dialog()
        await expect(view.getByText(/payment on a debt/)).toBeVisible()
        await expect(view.getByText(/differ by/)).toBeVisible()
    },
}

export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async () => {
        const view = await dialog()
        await expect(view.queryByRole('button', { name: 'Edit' })).toBeNull()
        await expect(view.queryByRole('button', { name: 'Delete' })).toBeNull()
    },
}

// On a phone Delete is in the "more" menu, not a footer button.
export const DeleteAsks: Story = {
    tags: ['desktop-only'],
    play: async ({ args }) => {
        const view = await dialog()
        await userEvent.click(view.getByRole('button', { name: 'Delete' }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Delete' }))
        await expect(args.onDelete).toHaveBeenCalledOnce()
    },
}
