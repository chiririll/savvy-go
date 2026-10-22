import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { currencies, pendingTransaction, transaction } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { RecentTransactionsCard } from './RecentTransactionsCard'

const meta = {
    title: 'features/dashboard/RecentTransactionsCard',
    component: RecentTransactionsCard,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        onEdit: fn(),
        transactions: [
            transaction({ id: 1, description: 'Coffee', amount: 3.5 }),
            transaction({ id: 2, type: 'income', description: 'Salary', amount: 3200 }),
            pendingTransaction({ id: 3, description: 'Rent', amount: 1100 }),
        ],
    },
} satisfies Meta<typeof RecentTransactionsCard>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {}
export const Empty: Story = { args: { transactions: [] } }
export const Loading: Story = { args: { transactions: undefined } }
export const InRussian: Story = { globals: { locale: 'ru' } }

// The rows here only open; they have no menu of their own.
export const NoRowMenus: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button', { name: /more actions/i })).toBeNull()
    },
}

export const OpensOnClick: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Coffee'))
        await expect(args.onEdit).toHaveBeenCalledOnce()
    },
}
