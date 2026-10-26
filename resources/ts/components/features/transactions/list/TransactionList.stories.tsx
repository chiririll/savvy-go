import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { currencies, pendingTransaction, transaction } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionList } from './TransactionList'

const transactions = [
    transaction({ id: 1, description: 'Coffee', date: '2020-05-04', amount: 3.5 }),
    transaction({ id: 2, description: 'Bus ticket', date: '2020-05-04', amount: 2 }),
    transaction({ id: 3, description: 'Rent', date: '2020-05-01', amount: 1100 }),
    transaction({ id: 4, description: 'Gift', date: null, amount: 30 }),
]

const meta = {
    title: 'features/transactions/list/TransactionList',
    component: TransactionList,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        transactions,
        emptyTitle: 'No transactions',
        emptyDescription: 'Add one to see it here.',
        onDelete: fn(),
        onDuplicate: fn(),
        onEdit: fn(),
    },
} satisfies Meta<typeof TransactionList>

export default meta
type Story = StoryObj<typeof meta>

export const GroupedByDay: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getAllByRole('heading', { level: 2 }).length).toBeGreaterThanOrEqual(2)
    },
}

// Without groups every row carries its own date.
export const Ungrouped: Story = { args: { grouped: false } }
export const Loading: Story = { args: { isLoading: true } }
export const Empty: Story = { args: { transactions: [], onCreate: fn(), createLabel: 'Add transaction' } }
export const EmptyReadOnly: Story = {
    args: { transactions: [], onCreate: fn(), createLabel: 'Add transaction', isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}
export const PendingList: Story = {
    args: {
        transactions: [pendingTransaction({ id: 7 }), pendingTransaction({ id: 8, date: '2099-07-01' })],
        onConfirm: fn(),
        onSkip: fn(),
    },
}
export const ReadOnly: Story = { args: { isReadOnly: true } }

// Coffee and Bus ticket share a day: the first can only go down, the second only up.
export const MovedWithMenu: Story = {
    tags: ['desktop-only'],
    args: { onReorder: fn() },
    play: async ({ canvasElement, args }) => {
        const menus = within(canvasElement).getAllByRole('button', { name: /more actions/i })
        await userEvent.click(menus[0])
        await expect(screen.queryByRole('menuitem', { name: /move up/i })).toBeNull()
        await userEvent.click(await screen.findByRole('menuitem', { name: /move down/i }))
        await expect(args.onReorder).toHaveBeenCalledWith('2020-05-04', [2, 1])
    },
}

export const NoMoveWhenReadOnly: Story = {
    tags: ['desktop-only'],
    args: { onReorder: fn(), isReadOnly: true },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /more actions/i })[0])
        await expect(screen.queryByRole('menuitem', { name: /move/i })).toBeNull()
        await userEvent.keyboard('{Escape}')
    },
}
