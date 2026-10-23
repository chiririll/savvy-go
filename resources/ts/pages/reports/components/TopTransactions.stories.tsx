import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { useLocation } from 'react-router-dom'

import type { TopTransactionItem, TopTransactionsData } from '@/api/reports'
import { currencies } from '@/test/fixtures'
import { hang, reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { TopTransactions } from './TopTransactions'

const item = (id: number, amount: number, overrides: Partial<TopTransactionItem> = {}): TopTransactionItem => ({
    id,
    description: `Payment ${id}`,
    amount,
    date: '2030-01-05',
    category: { id: 1, name: 'Groceries', icon: '🛒', color: '#16a34a' },
    account: { id: 1, name: 'Cash' },
    ...overrides,
})

const top = (currency: string | null, items: TopTransactionItem[]): TopTransactionsData => ({ currency, items })

// Shows where a click on a row went.
const Place = () => <p data-testid="place">{useLocation().pathname + useLocation().search}</p>

const meta = {
    title: 'pages/reports/TopTransactions',
    component: TopTransactions,
    parameters: { layout: 'padded' },
    args: { filters: DEFAULT_FILTERS, type: 'expense' },
    render: (args) => (
        <>
            <TopTransactions {...args} />
            <Place />
        </>
    ),
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply(
                'get',
                '/reports/transactions/top',
                top('USD', [item(11, 450.25), item(12, 120, { description: '', account: { id: 2, name: 'Card' } })]),
            ),
        )
    },
} satisfies Meta<typeof TopTransactions>

export default meta
type Story = StoryObj<typeof meta>

export const Expenses: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Payment 11')).toBeVisible()
        await expect(canvas.getByText('-450.25 $')).toBeVisible()
        // The sum of the top rows, in the report's currency.
        await expect(canvas.getByText('570.25 $')).toBeVisible()
        // A row without a description falls back to its category.
        await expect(canvas.getAllByText('Groceries').length).toBeGreaterThan(0)
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const Income: Story = {
    args: { type: 'income' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('+450.25 $')).toBeVisible()
    },
}

// Small amounts still add up to the cent.
export const SumOfSmallAmounts: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/transactions/top', top('USD', [item(1, 0.1), item(2, 0.2), item(3, 0.3)]))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('0.60 $')).toBeVisible()
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/transactions/top', top('JPY', [item(1, 12_000), item(2, 3_500)]))),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('-12,000 ¥')).toBeVisible()
        await expect(canvas.getByText('15,500 ¥')).toBeVisible()
    },
}

export const OpensTheTransaction: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByText('Payment 11'))
        await expect(canvas.getByTestId('place')).toHaveTextContent('/transactions?id=11')
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/transactions/top', top('USD', []))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no expenses for selected period/i)).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/transactions/top')) }
