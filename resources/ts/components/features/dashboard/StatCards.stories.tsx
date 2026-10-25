import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DashboardStats } from './StatCards'


const meta = {
    title: 'features/dashboard/DashboardStats',
    component: DashboardStats,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        currency: 'USD',
        totalBalance: 12345.67,
        periodIncome: 3200,
        periodExpense: 1840.5,
        balanceChange: 4.2,
        incomeChange: 10,
        expenseChange: -3.5,
        pendingSummary: { income: 0, expense: 0, balance: -250, transactionsCount: 3, currency: 'USD' },
    },
} satisfies Meta<typeof DashboardStats>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}
export const NoComparison: Story = { args: { balanceChange: null, incomeChange: null, expenseChange: null } }
export const Worse: Story = { args: { balanceChange: -8, incomeChange: -12.5, expenseChange: 20 } }
export const NothingPending: Story = {
    args: { pendingSummary: { income: 0, expense: 0, balance: 0, transactionsCount: 0, currency: 'USD' } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('link')).toHaveAttribute('href', '/transactions?status=pending')
    },
}
export const PendingIncome: Story = {
    args: { pendingSummary: { income: 500, expense: 0, balance: 500, transactionsCount: 1, currency: 'USD' } },
}
export const NoPendingSummary: Story = { args: { pendingSummary: undefined } }
export const ZeroEverywhere: Story = {
    args: { totalBalance: 0, periodIncome: 0, periodExpense: 0, balanceChange: 0, incomeChange: 0, expenseChange: 0 },
}
export const NegativeBalance: Story = { args: { totalBalance: -420.1 } }
export const VeryLargeAmounts: Story = {
    args: { totalBalance: 987654321.99, periodIncome: 123456789.01, periodExpense: 23456789.5 },
}

// The figures follow the precision of the currency they are in.
export const InYen: Story = { args: { currency: 'JPY', totalBalance: 1500000, periodIncome: 320000, periodExpense: 184050 } }
export const InDinars: Story = { args: { currency: 'KWD', totalBalance: 1234.567, periodIncome: 320.125, periodExpense: 84.005 } }
