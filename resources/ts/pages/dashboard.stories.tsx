import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { accounts, budget, categories, currencies, debt, transaction } from '@/test/fixtures'
import { hang, raw, reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'
import { CreateTransactionProvider } from '@/components/features/transactions/form/CreateTransactionProvider'

import DashboardPage from './dashboard'

const debtSummary = { totalIOwe: 400, totalOwedToMe: 0, netDebt: -400, debtsCount: 1, currency: 'USD', decimals: 2 }

const metric = (value: number, previous: number) => ({ value, previous, sparkline: [1, 2, 3] })

const meta = {
    title: 'pages/Dashboard',
    component: DashboardPage,
    parameters: { a11y: overlayA11y },
    // The page opens the transaction dialog the layout provides.
    render: () => (
        <CreateTransactionProvider>
            <DashboardPage />
        </CreateTransactionProvider>
    ),
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            raw('get', '/accounts', {
                data: accounts,
                summary: { totalBalance: 1500.5, currency: 'USD', decimals: 2, accountsCount: accounts.length },
            }),
            raw('get', '/accounts-balance-comparison', { current: 1500.5, previous: 1000, currency: 'USD', decimals: 2 }),
            raw('get', '/accounts-balance-history', { dates: ['2030-01-01', '2030-01-02'], series: [], currency: 'USD', decimals: 2 }),
            raw('get', '/transactions', { data: [transaction({ description: 'Weekly shop' })] }),
            reply('get', '/transactions-pending-summary', { income: 0, expense: 0, balance: 0, currency: 'USD', decimals: 2, count: 0 }),
            raw('get', '/categories-summary', { data: [{ ...categories[0], total: 450.25 }], total: 450.25, currency: 'USD' }),
            reply('get', '/budgets', [budget()]),
            raw('get', '/debts', { data: [debt()], summary: debtSummary }),
            reply('get', '/reports/overview', {
                income: metric(3000, 2500),
                expenses: metric(1800.5, 2000),
                netCashFlow: metric(1199.5, 500),
                savingsRate: metric(40, 20),
                currency: 'USD',
            }),
            reply('get', '/currencies', currencies),
            reply('get', '/categories', categories),
        )
    },
} satisfies Meta<typeof DashboardPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('heading', { level: 1 })).toBeVisible()
        await expect(await canvas.findByText('Weekly shop')).toBeVisible()
        await expect(await canvas.findByText('Loan from Alex')).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

export const Loading: Story = {
    beforeEach: ({ msw }) => msw.use(hang('get', '/accounts'), hang('get', '/transactions'), hang('get', '/debts')),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('heading', { level: 1 })).toBeVisible()
    },
}

export const NothingYet: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            raw('get', '/accounts', { data: [], summary: { totalBalance: 0, currency: 'USD', decimals: 2, accountsCount: 0 } }),
            raw('get', '/transactions', { data: [] }),
            reply('get', '/budgets', []),
            raw('get', '/debts', { data: [], summary: { ...debtSummary, totalIOwe: 0, netDebt: 0, debtsCount: 0 } }),
        ),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('heading', { level: 1 })).toBeVisible()
    },
}
