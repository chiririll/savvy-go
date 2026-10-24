import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import type { CashFlowDataPoint, CashFlowOverTimeData } from '@/api/reports'
import { fail, hang, reply } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { expectChartDrawn } from '../story-fixtures'
import { CashFlowChart } from './CashFlowChart'

const point = (day: number, income: number, expenses: number): CashFlowDataPoint => ({
    date: `2030-01-${String(day).padStart(2, '0')}`,
    income,
    expenses,
    balance: income - expenses,
    prevIncome: income / 2,
    prevExpenses: expenses / 2,
    prevBalance: (income - expenses) / 2,
})

const flow = (currency: string | null, scale = 1, income = 1000, expenses = 400): CashFlowOverTimeData => ({
    items: [point(1, income * scale, expenses * scale), point(2, 0, 120.5 * scale), point(3, 300 * scale, 80 * scale)],
    currency,
})

const meta = {
    title: 'pages/reports/CashFlowChart',
    component: CashFlowChart,
    parameters: { layout: 'padded' },
    args: { filters: DEFAULT_FILTERS },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/reports/cash-flow-over-time', flow('USD')))
    },
} satisfies Meta<typeof CashFlowChart>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Cash flow over time')).toBeVisible()
        await expectChartDrawn(canvasElement)
    },
}

export const WithoutComparison: Story = {
    args: { filters: { ...DEFAULT_FILTERS, compareWith: 'none' } },
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const Monthly: Story = {
    args: { filters: { ...DEFAULT_FILTERS, periodType: 'year', selectedYear: '2030' } },
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/cash-flow-over-time', flow('JPY', 1000))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/cash-flow-over-time', flow('KWD', 0.125))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InEuroWithBalanceBelowZero: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/cash-flow-over-time', flow('EUR', 1, 100, 400))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

// A chart of one-sided flow says nothing, so the card says why it draws none.
export const NoIncome: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/cash-flow-over-time', { items: [point(1, 0, 50), point(2, 0, 20)], currency: 'USD' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No income data for selected period')).toBeVisible()
    },
}

export const NoExpenses: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/cash-flow-over-time', { items: [point(1, 50, 0), point(2, 20, 0)], currency: 'USD' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No expenses data for selected period')).toBeVisible()
    },
}

export const NoIncomeOrExpenses: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/cash-flow-over-time', { items: [point(1, 0, 0)], currency: 'USD' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No income or expenses for selected period')).toBeVisible()
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/cash-flow-over-time', { items: [], currency: 'USD' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no data/i)).toBeVisible()
    },
}

export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/reports/cash-flow-over-time')),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Failed to load cash flow data')).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/cash-flow-over-time')) }
