import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import type { TransactionSummaryData } from '@/api/reports'
import { currencies } from '@/test/fixtures'
import { fail, hang, reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { TransactionTypeSummary } from './TransactionTypeSummary'

const summary = (overrides: Partial<TransactionSummaryData> = {}): TransactionSummaryData => ({
    total: 3100.5,
    previous: 2800,
    avgPerDay: 103.35,
    avgPerWeek: 723.45,
    prevAvgPerDay: 93.33,
    prevAvgPerWeek: 653.33,
    daysInPeriod: 30,
    currency: 'USD',
    ...overrides,
})

const meta = {
    title: 'pages/reports/TransactionTypeSummary',
    component: TransactionTypeSummary,
    parameters: { layout: 'padded' },
    args: { filters: DEFAULT_FILTERS, type: 'expense' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/reports/transactions/summary', summary()))
    },
} satisfies Meta<typeof TransactionTypeSummary>

export default meta
type Story = StoryObj<typeof meta>

export const Expenses: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/3,100\.50 \$/)).toBeVisible()
        await expect(canvas.getByText('Total expenses')).toBeVisible()
        await expect(canvas.getByText('10.7%')).toBeVisible()
        await expect(canvas.getByText(/\+300\.50 \$ vs previous period/)).toBeVisible()
        await expect(canvas.getByText(/103\.35 \$/)).toBeVisible()
        await expect(canvas.getByText('vs 93.33 $')).toBeVisible()
    },
}

export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }

export const Income: Story = {
    args: { type: 'income' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Total income')).toBeVisible()
    },
}

// The report is in the base currency of the space, which can be one without a fraction.
export const InYen: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            reply(
                'get',
                '/reports/transactions/summary',
                summary({
                    total: 465_000,
                    previous: 500_000,
                    avgPerDay: 15_500,
                    avgPerWeek: 108_500,
                    prevAvgPerDay: 16_667,
                    prevAvgPerWeek: 116_667,
                    currency: 'JPY',
                }),
            ),
        ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('465,000 ¥')).toBeVisible()
        await expect(canvas.getByText('7.0%')).toBeVisible()
        await expect(canvas.getByText(/-35,000 ¥ vs previous period/)).toBeVisible()
    },
}

export const NoComparison: Story = {
    args: { filters: { ...DEFAULT_FILTERS, compareWith: 'none' } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/3,100\.50 \$/)).toBeVisible()
        await expect(canvas.queryByText(/vs previous period/)).toBeNull()
        await expect(canvas.queryByText(/^vs /)).toBeNull()
    },
}

export const FirstPeriod: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/transactions/summary', summary({ previous: null, prevAvgPerDay: null, prevAvgPerWeek: null }))),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/3,100\.50 \$/)).toBeVisible()
        await expect(canvas.queryByText(/vs previous period/)).toBeNull()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/transactions/summary')) }

export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/reports/transactions/summary')),
}
