import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import type { MoneyFlowData, NetWorthData, OverviewMetrics, TransactionsByCategoryData } from '@/api/reports'
import { unmountedTabsA11y } from '@/test/a11y'
import { fail, reply } from '@/test/api-mocks'
import { accounts, categories, currencies, tags } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import ReportsPage from './index'

const overview = (currency: string | null, scale = 1): OverviewMetrics => ({
    income: { value: 3000 * scale, previous: 2500 * scale, sparkline: [1, 2, 3] },
    expenses: { value: 1800.5 * scale, previous: 2000 * scale, sparkline: [3, 2, 1] },
    netCashFlow: { value: 1199.5 * scale, previous: 500 * scale, sparkline: [1, 2, 4] },
    savingsRate: { value: 40, previous: 20, sparkline: [1, 2, 4] },
    currency,
})

const flow = (currency: string | null): MoneyFlowData => ({
    nodes: [{ name: 'Salary', itemStyle: { color: '#3b82f6' } }, { name: 'Rent', itemStyle: { color: '#f97316' } }],
    links: [{ source: 'Salary', target: 'Rent', value: 500 }],
    totals: { income: 1000, expenses: 500, savings: 500 },
    currency,
})

const byCategory = (currency: string | null): TransactionsByCategoryData => ({
    items: [
        { id: 1, name: 'Groceries', icon: '🛒', color: '#16a34a', value: 450.25, percentage: 33 },
        { id: 2, name: 'Rent', icon: '🏠', color: '#f97316', value: 900, percentage: 67 },
    ],
    total: 1350.25,
    currency,
})

const worth = (currency: string | null, scale = 1): NetWorthData => ({
    current: 1500.5 * scale,
    previous: 1000 * scale,
    change: 500.5 * scale,
    changePercent: 50.05,
    accounts: [{ id: 1, name: 'Cash', type: 'cash', balance: 1500.5 * scale, percentage: 100 }],
    currency,
})

const reports = (currency: string | null, scale = 1) => [
    reply('get', '/reports/overview', overview(currency, scale)),
    reply('get', '/reports/money-flow', flow(currency)),
    reply('get', '/reports/expense-pace', { months: [], currency }),
    reply('get', '/reports/expenses-by-category', {
        categories: [{ id: 7, name: 'Groceries', icon: '🛒', color: '#16a34a', current: 450.25 * scale, previous: 300 * scale }],
        currency,
    }),
    reply('get', '/reports/cash-flow-over-time', {
        items: [{ date: '2030-01-01', income: 1000, expenses: 400, balance: 600, prevIncome: 0, prevExpenses: 0, prevBalance: 0 }],
        currency,
    }),
    reply('get', '/reports/activity-heatmap', { items: [{ date: '2030-01-01', value: 12.5, count: 1 }], max: 12.5, currency }),
    reply('get', '/reports/transactions/summary', {
        total: 3100.5 * scale,
        previous: 2800 * scale,
        avgPerDay: 103.35,
        avgPerWeek: 723.45,
        prevAvgPerDay: 93.33,
        prevAvgPerWeek: 653.33,
        daysInPeriod: 30,
        currency,
    }),
    reply('get', '/reports/transactions/by-category', byCategory(currency)),
    reply('get', '/reports/transactions/dynamics', {
        dates: ['2030-01-01', '2030-01-02'],
        datasets: [{ id: 0, name: 'Total', color: '#3b82f6', data: [100, 200] }],
        currency,
    }),
    reply('get', '/reports/transactions/top', { currency, items: [] }),
    reply('get', '/reports/net-worth', worth(currency, scale)),
    reply('get', '/reports/net-worth-history', { dates: ['2030-01-01', '2030-01-02'], values: [1000, 1500.5], currency }),
]

const meta = {
    title: 'pages/reports/ReportsPage',
    component: ReportsPage,
    parameters: { a11y: unmountedTabsA11y },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/accounts', accounts),
            reply('get', '/categories', categories),
            reply('get', '/tags', tags),
            ...reports('USD'),
        )
    },
} satisfies Meta<typeof ReportsPage>

export default meta
type Story = StoryObj<typeof meta>

const openTab = async (canvasElement: HTMLElement, name: RegExp) =>
    userEvent.click(await within(canvasElement).findByRole('tab', { name }))

export const Overview: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('heading', { name: 'Reports' })).toBeVisible()
        await expect(await canvas.findByText('Money flow')).toBeVisible()
        await expect(await canvas.findByText('Groceries')).toBeVisible()
    },
}

export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }

export const OverviewFailed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/reports/overview')),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/failed to load/i)).toBeVisible()
    },
}

export const CashFlowTab: Story = {
    play: async ({ canvasElement }) => {
        await openTab(canvasElement, /cash flow/i)
        await expect(await within(canvasElement).findByText('Cash flow over time')).toBeVisible()
    },
}

export const ExpensesTab: Story = {
    play: async ({ canvasElement }) => {
        await openTab(canvasElement, /^expenses/i)
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Expenses structure')).toBeVisible()
        await expect(await canvas.findByText('Expenses dynamics')).toBeVisible()
        await expect(await canvas.findByText('Total expenses')).toBeVisible()
    },
}

export const IncomeTab: Story = {
    play: async ({ canvasElement }) => {
        await openTab(canvasElement, /^income/i)
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Income dynamics')).toBeVisible()
        await expect(await canvas.findByText('Total income')).toBeVisible()
    },
}

export const NetWorthTab: Story = {
    play: async ({ canvasElement }) => {
        await openTab(canvasElement, /net worth/i)
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Current net worth')).toBeVisible()
        await expect((await canvas.findAllByText('1,500.50 $'))[0]).toBeVisible()
        await expect(canvas.getByText('+500.50 $')).toBeVisible()
    },
}

// A zero-decimal and a three-decimal currency in the headline amount.
export const NetWorthInYen: Story = {
    beforeEach: ({ msw }) => msw.use(...reports('JPY', 1000)),
    play: async ({ canvasElement }) => {
        await openTab(canvasElement, /net worth/i)
        await expect(await within(canvasElement).findByText('1,500,500 ¥')).toBeVisible()
    },
}

export const NetWorthInDinar: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/net-worth', worth('KWD'))),
    play: async ({ canvasElement }) => {
        await openTab(canvasElement, /net worth/i)
        await waitFor(() => expect(within(canvasElement).getByText(/1,500\.500/)).toBeVisible())
    },
}
