import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import type { NetWorthData, TransactionsByCategoryData } from '@/api/reports'
import { fail, hang, reply } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { chartText, expectChartDrawn } from '../story-fixtures'
import { AccountsStructureChart } from './AccountsStructureChart'
import { TransactionStructureChart } from './TransactionStructureChart'

const byCategory = (currency: string | null, values: number[]): TransactionsByCategoryData => {
    const total = values.reduce((sum, value) => sum + value, 0)
    return {
        items: values.map((value, i) => ({
            id: i + 1,
            name: ['Groceries', 'Rent', 'Fun'][i] ?? `Other ${i}`,
            icon: '🛒',
            color: ['#16a34a', '#f97316', '#8b5cf6'][i] ?? '#64748b',
            value,
            percentage: Math.round((value / total) * 100),
        })),
        total,
        currency,
    }
}

const worth = (currency: string | null, balances: Array<[string, number]>): NetWorthData => ({
    current: balances.reduce((sum, [, balance]) => sum + balance, 0),
    previous: null,
    change: 0,
    changePercent: 0,
    accounts: balances.map(([name, balance], i) => ({ id: i + 1, name, type: 'cash', balance, percentage: 0 })),
    currency,
})

const meta = {
    title: 'pages/reports/StructureChart',
    component: TransactionStructureChart,
    parameters: { layout: 'padded' },
    args: { filters: DEFAULT_FILTERS, type: 'expense' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/reports/transactions/by-category', byCategory('USD', [450.25, 900, 100.1])))
    },
} satisfies Meta<typeof TransactionStructureChart>

export default meta
type Story = StoryObj<typeof meta>

export const Donut: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Expenses structure')).toBeVisible()
        await expectChartDrawn(canvasElement)
        // The total in the middle is the sum of the slices, to the cent.
        await waitFor(() => expect(chartText(canvasElement)).toContain('1,450.35 $'))
    },
}

export const Income: Story = {
    args: { type: 'income' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Income structure')).toBeVisible()
        await expectChartDrawn(canvasElement)
    },
}

// The views are buttons: the keyboard reaches them.
export const SwitchesTheView: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const donut = await canvas.findByRole('button', { name: 'Donut' })
        await expect(donut).toHaveAttribute('aria-pressed', 'true')
        await expectChartDrawn(canvasElement)

        await userEvent.click(canvas.getByRole('button', { name: 'Bar' }))
        await expect(canvas.getByRole('button', { name: 'Bar' })).toHaveAttribute('aria-pressed', 'true')
        await expect(donut).toHaveAttribute('aria-pressed', 'false')
        await expectChartDrawn(canvasElement)

        const treemap = canvas.getByRole('button', { name: 'Treemap' })
        treemap.focus()
        await userEvent.keyboard('{Enter}')
        await expect(treemap).toHaveAttribute('aria-pressed', 'true')
        await expectChartDrawn(canvasElement)
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/transactions/by-category', byCategory('JPY', [12_000, 3_500]))),
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(chartText(canvasElement)).toContain('15,500 ¥'))
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/transactions/by-category', byCategory('KWD', [10.125, 2.5]))),
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(chartText(canvasElement)).toMatch(/12\.625/))
    },
}

export const InEuro: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/transactions/by-category', byCategory('EUR', [0.1, 0.2]))),
    play: async ({ canvasElement }) => {
        // 0.1 + 0.2 is 0.30, not 0.30000000000000004.
        await waitFor(() => expect(chartText(canvasElement)).toContain('0.30 €'))
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/transactions/by-category', byCategory('USD', []))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no expenses for selected period/i)).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/transactions/by-category')) }

export const AccountsByBalance: Story = {
    render: (args) => <AccountsStructureChart filters={args.filters} />,
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/net-worth', worth('EUR', [['Cash', 500.5], ['Card', 250], ['Loan', -300]]))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Accounts breakdown')).toBeVisible()
        // The total counts the assets only; an account in debt takes no share.
        await waitFor(() => expect(chartText(canvasElement)).toContain('750.50 €'))
        await expect(chartText(canvasElement)).not.toContain('Loan')
    },
}

export const AccountsInRussian: Story = {
    ...AccountsByBalance,
    globals: { locale: 'ru' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Разбивка по счетам')).toBeVisible()
        await expectChartDrawn(canvasElement)
    },
}

export const NoAccounts: Story = {
    render: (args) => <AccountsStructureChart filters={args.filters} />,
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/net-worth', worth('USD', [['Loan', -300]]))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No accounts found')).toBeVisible()
    },
}

export const FailedToLoad: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/reports/transactions/by-category')),
    play: async ({ canvasElement }) => {
        // The card keeps its title and shows nothing to draw.
        await expect(await within(canvasElement).findByText('Expenses structure')).toBeVisible()
    },
}
