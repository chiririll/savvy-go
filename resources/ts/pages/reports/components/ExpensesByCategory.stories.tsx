import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { useLocation } from 'react-router-dom'

import type { CategoryExpense, ExpensesByCategoryData } from '@/api/reports'
import { fail, hang, reply } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { ExpensesByCategory } from './ExpensesByCategory'

const row = (id: number, name: string, current: number, previous: number): CategoryExpense => ({
    id,
    name,
    icon: '🛒',
    color: '#16a34a',
    current,
    previous,
})

const data = (currency: string | null, ...categories: CategoryExpense[]): ExpensesByCategoryData => ({
    categories,
    currency,
})

// Shows where a click on a row went.
const Place = () => <p data-testid="place">{useLocation().pathname + useLocation().search}</p>

const meta = {
    title: 'pages/reports/ExpensesByCategory',
    component: ExpensesByCategory,
    parameters: { layout: 'padded' },
    args: { filters: DEFAULT_FILTERS },
    render: (args) => (
        <>
            <ExpensesByCategory {...args} />
            <Place />
        </>
    ),
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/reports/expenses-by-category', data('USD', row(7, 'Groceries', 450.25, 300), row(8, 'Rent', 900, 900))),
        )
    },
} satisfies Meta<typeof ExpensesByCategory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Groceries')).toBeVisible()
        await expect(canvas.getByText('450.25 $')).toBeVisible()
        await expect(canvas.getByText('vs 300.00 $')).toBeVisible()
        // 450.25 against 300 is up by half.
        await expect(canvas.getByText('50%')).toBeVisible()
    },
}

export const WithoutComparison: Story = {
    args: { filters: { ...DEFAULT_FILTERS, compareWith: 'none' } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('450.25 $')).toBeVisible()
        await expect(canvas.queryByText(/^vs /)).toBeNull()
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/expenses-by-category', data('JPY', row(7, 'Groceries', 12_000, 8_000)))),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('12,000 ¥')).toBeVisible()
        await expect(canvas.getByText('vs 8,000 ¥')).toBeVisible()
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/expenses-by-category', data('KWD', row(7, 'Groceries', 12.345, 10.5)))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/12\.345/)).toBeVisible()
    },
}

export const BuiltInNames: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/expenses-by-category', data('USD', row(7, '#GROCERIES', 10, 5)))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Groceries')).toBeVisible()
    },
}

export const OpensTheTransactions: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByText('Groceries'))
        await expect(canvas.getByTestId('place')).toHaveTextContent('/transactions?category_ids=7')
    },
}

export const KeyboardAccess: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const groceries = await canvas.findByRole('button', { name: /Groceries/ })
        await userEvent.tab()
        await expect(groceries).toHaveFocus()
        await userEvent.keyboard('{Enter}')
        await expect(canvas.getByTestId('place')).toHaveTextContent('/transactions?category_ids=7')
        const rent = canvas.getByRole('button', { name: /Rent/ })
        rent.focus()
        await userEvent.keyboard(' ')
        await expect(canvas.getByTestId('place')).toHaveTextContent('/transactions?category_ids=8')
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/expenses-by-category', data('USD'))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No expense data for selected period')).toBeVisible()
    },
}

export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/reports/expenses-by-category')),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Failed to load expenses by category')).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/expenses-by-category')) }
