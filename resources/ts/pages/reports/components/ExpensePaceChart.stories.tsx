import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import type { ExpensePaceData, ExpensePaceMonth } from '@/api/reports'
import { overlayA11y } from '@/test/a11y'
import { fail, hang, reply } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { expectChartDrawn } from '../story-fixtures'
import { ExpensePaceChart } from './ExpensePaceChart'

// A finished month of 31 days (no "today"), so nothing depends on the date the test runs.
const month = (budget: number | null, totalSpent: number, monthStart = '2030-01-01'): ExpensePaceMonth => {
    const daily = Array.from({ length: 31 }, (_, i) => (totalSpent * (i + 1)) / 31)
    return {
        budget,
        dailyExpenses: daily,
        currentDay: null,
        daysInMonth: 31,
        totalSpent,
        monthStart,
        monthEnd: monthStart.replace('-01', '-31'),
    }
}

const pace = (currency: string | null, ...months: ExpensePaceMonth[]): ExpensePaceData => ({ months, currency })

const meta = {
    title: 'pages/reports/ExpensePaceChart',
    component: ExpensePaceChart,
    parameters: { layout: 'padded', a11y: overlayA11y },
    args: { filters: DEFAULT_FILTERS },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/reports/expense-pace', pace('USD', month(1000, 1200.5))))
    },
} satisfies Meta<typeof ExpensePaceChart>

export default meta
type Story = StoryObj<typeof meta>

export const OverBudget: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('Expense pace')).toBeVisible()
        await expect(await canvas.findByText('Budget remaining')).toBeVisible()
        // The overshoot is exact to the cent: 1,200.50 - 1,000.
        await expect(canvas.getByText('+200.50 $ over budget')).toBeVisible()
        await expect(canvas.getByText('-200.50 $')).toBeVisible()
        await expectChartDrawn(canvasElement)
    },
}

export const InRussian: Story = {
    globals: { locale: 'ru' },
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const UnderBudget: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/expense-pace', pace('USD', month(1000, 640.1)))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('359.90 $ under budget')).toBeVisible()
    },
}

export const WithoutBudget: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/expense-pace', pace('USD', month(null, 640.1)))),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Spent so far')).toBeVisible()
        await expect(canvas.getByText(/no budget set/i)).toBeVisible()
        await expect(canvas.getAllByText('640.10 $').length).toBeGreaterThan(0)
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/expense-pace', pace('JPY', month(100_000, 120_000)))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('+20,000 ¥ over budget')).toBeVisible()
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/expense-pace', pace('KWD', month(50, 62.345)))),
    play: async ({ canvasElement }) => {
        // 62.345 - 50 has no float noise left in three decimals.
        await expect(await within(canvasElement).findByText(/\+12\.345 .* over budget/)).toBeVisible()
    },
}

export const SeveralMonths: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/expense-pace', pace('EUR', month(500, 300), month(500, 700, '2030-02-01')))),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        // The last month is the one shown; the select goes back to the earlier one.
        await expect(await canvas.findByText('200.00 € over budget', { exact: false })).toBeVisible()
        await userEvent.click(canvas.getByRole('combobox', { name: 'Month' }))
        const options = await screen.findAllByRole('option')
        await userEvent.click(options[0])
        await waitFor(() => expect(canvas.getByText('200.00 € under budget')).toBeVisible())
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/expense-pace', pace('USD'))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No expense data for selected period')).toBeVisible()
    },
}

export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/reports/expense-pace')),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Failed to load expense pace data')).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/expense-pace')) }
