import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import type { OverviewMetrics } from '@/api/reports'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { OverviewMetricCards, type OverviewMetricSpec } from './OverviewMetricCards'

const metrics: OverviewMetricSpec[] = [
    { key: 'income', title: 'Income', type: 'income' },
    { key: 'expenses', title: 'Expenses', type: 'expense' },
    { key: 'netCashFlow', title: 'Net cash flow', type: 'net' },
    { key: 'savingsRate', title: 'Savings rate', type: 'percent' },
]

const data = (currency: string | null): OverviewMetrics => ({
    income: { value: 3000, previous: 2500, sparkline: [1, 2, 3] },
    expenses: { value: 1800.5, previous: 2000, sparkline: [3, 2, 1] },
    netCashFlow: { value: 1199.5, previous: 500, sparkline: [1, 2, 4] },
    savingsRate: { value: 40, previous: 20, sparkline: [1, 2, 4] },
    currency,
})

const meta = {
    title: 'pages/reports/OverviewMetricCards',
    component: OverviewMetricCards,
    parameters: { layout: 'padded' },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        metrics,
        data: data('USD'),
        isLoading: false,
        compareWith: 'previous_period',
        className: 'sm:grid-cols-2 lg:grid-cols-4',
    },
} satisfies Meta<typeof OverviewMetricCards>

export default meta
type Story = StoryObj<typeof meta>

export const Loaded: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/3,000\.00 \$/)).toBeVisible()
        await expect(canvas.getByText(/1,800\.50 \$/)).toBeVisible()
        await expect(canvas.getByText(/1,199\.50 \$/)).toBeVisible()
        // The rate is a percentage, never money.
        await expect(canvas.getByText('40.0%')).toBeVisible()
    },
}

export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }

export const InYen: Story = {
    args: { data: data('JPY') },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/3,000 ¥/)).toBeVisible()
        // Rounded to the yen, never shown with a fraction.
        await expect(canvas.getByText(/1,801 ¥/)).toBeVisible()
    },
}

export const Loading: Story = {
    args: { isLoading: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText('Income')).toBeNull()
    },
}

export const NoData: Story = { args: { data: undefined } }
