import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import type { TransactionDynamicsData } from '@/api/reports'
import { overlayA11y } from '@/test/a11y'
import { hang, reply } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { chartText, expectChartDrawn } from '../story-fixtures'
import { TransactionDynamicsChart } from './TransactionDynamicsChart'

const dates = ['2030-01-01', '2030-01-02', '2030-01-03']

const dynamics = (currency: string | null, scale = 1): TransactionDynamicsData => ({
    dates,
    datasets: [
        { id: 0, name: 'Total', color: '#3b82f6', data: [100 * scale, 250 * scale, 80 * scale] },
        { id: 1, name: 'Groceries', color: '#16a34a', data: [60 * scale, 150 * scale, 30 * scale] },
        { id: 2, name: 'Rent', color: '#f97316', data: [40 * scale, 100 * scale, 50 * scale] },
    ],
    currency,
})

const meta = {
    title: 'pages/reports/TransactionDynamicsChart',
    component: TransactionDynamicsChart,
    parameters: { layout: 'padded', a11y: overlayA11y },
    args: { filters: DEFAULT_FILTERS, type: 'expense' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/reports/transactions/dynamics', dynamics('USD')))
    },
} satisfies Meta<typeof TransactionDynamicsChart>

export default meta
type Story = StoryObj<typeof meta>

export const Expenses: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Expenses dynamics')).toBeVisible()
        await expectChartDrawn(canvasElement)
    },
}

export const Income: Story = {
    args: { type: 'income' },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Income dynamics')).toBeVisible()
        // Income is split by source, expenses by category.
        await expect(canvas.getByRole('button', { name: /sources/i })).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const SwitchesToBars: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const line = await canvas.findByRole('button', { name: 'Line' })
        await expect(line).toHaveAttribute('aria-pressed', 'true')
        await userEvent.click(canvas.getByRole('button', { name: 'Bar' }))
        await expect(canvas.getByRole('button', { name: 'Bar' })).toHaveAttribute('aria-pressed', 'true')
        await expect(line).toHaveAttribute('aria-pressed', 'false')
        await expectChartDrawn(canvasElement)
    },
}

// The first series is on to begin with, and the picker turns the others on.
export const PicksSeries: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const picker = await canvas.findByRole('button', { name: /categories/i })
        await waitFor(() => expect(picker).toHaveTextContent('1'))
        await userEvent.click(picker)
        await userEvent.click(await screen.findByText('Groceries'))
        await waitFor(() => expect(picker).toHaveTextContent('2'))
        await expect(await screen.findByRole('checkbox', { name: 'Rent' })).not.toBeChecked()
        await waitFor(() => expect(chartText(canvasElement)).toContain('Groceries'))
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/transactions/dynamics', dynamics('JPY', 1000))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/transactions/dynamics', dynamics('KWD', 0.125))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/transactions/dynamics', { dates: [], datasets: [], currency: 'USD' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no data/i)).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/transactions/dynamics')) }
