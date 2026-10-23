import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, waitFor, within } from 'storybook/test'

import type { MoneyFlowData } from '@/api/reports'
import { fail, hang, reply } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { chartText, expectChartDrawn } from '../story-fixtures'
import { SankeyDiagram } from './SankeyDiagram'

const node = (name: string, color = '#3b82f6') => ({ name, itemStyle: { color } })

const flow = (currency: string | null, amount: number): MoneyFlowData => ({
    nodes: [node('Salary'), node('Groceries', '#16a34a'), node('Rent', '#f97316'), node('Savings', '#8b5cf6')],
    links: [
        { source: 'Salary', target: 'Groceries', value: amount * 0.2 },
        { source: 'Salary', target: 'Rent', value: amount * 0.5 },
        { source: 'Salary', target: 'Savings', value: amount * 0.3 },
    ],
    totals: { income: amount, expenses: amount * 0.7, savings: amount * 0.3 },
    currency,
})

const meta = {
    title: 'pages/reports/SankeyDiagram',
    component: SankeyDiagram,
    parameters: { layout: 'padded' },
    args: { filters: DEFAULT_FILTERS },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/reports/money-flow', flow('USD', 1000)))
    },
} satisfies Meta<typeof SankeyDiagram>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Money flow')).toBeVisible()
        await expectChartDrawn(canvasElement)
        // The nodes are named in the drawing.
        await waitFor(() => expect(chartText(canvasElement)).toContain('Groceries'))
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/money-flow', flow('JPY', 300_000))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/money-flow', flow('KWD', 1234.5))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

// The built-in savings node is shown under its translated name.
export const SavingsNodeIsTranslated: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/reports/money-flow', {
                ...flow('USD', 1000),
                nodes: [node('Salary'), node('Rent'), node('__savings__')],
                links: [
                    { source: 'Salary', target: 'Rent', value: 600 },
                    { source: 'Salary', target: '__savings__', value: 400 },
                ],
            }),
        ),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
        await waitFor(() => expect(chartText(canvasElement)).toContain('Savings'))
        await expect(chartText(canvasElement)).not.toContain('__savings__')
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/money-flow', { ...flow('USD', 0), nodes: [], links: [] })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no data/i)).toBeVisible()
    },
}

export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/reports/money-flow')),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/money flow/i)).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/money-flow')) }
