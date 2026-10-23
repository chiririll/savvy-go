import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import type { NetWorthHistoryData } from '@/api/reports'
import { hang, reply } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { expectChartDrawn } from '../story-fixtures'
import { NetWorthChart } from './NetWorthChart'

const history = (currency: string | null, scale = 1): NetWorthHistoryData => ({
    dates: ['2030-01-01', '2030-01-02', '2030-01-03', '2030-01-04'],
    values: [1000 * scale, 1250.5 * scale, 900.25 * scale, 1500 * scale],
    currency,
})

const meta = {
    title: 'pages/reports/NetWorthChart',
    component: NetWorthChart,
    parameters: { layout: 'padded' },
    args: { filters: DEFAULT_FILTERS },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/reports/net-worth-history', history('USD')))
    },
} satisfies Meta<typeof NetWorthChart>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Net worth over time')).toBeVisible()
        await expectChartDrawn(canvasElement)
    },
}

export const InRussian: Story = {
    globals: { locale: 'ru' },
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const ByWeek: Story = {
    args: { filters: { ...DEFAULT_FILTERS, periodType: 'quarter', selectedQuarter: '2030-Q1' } },
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/net-worth-history', history('JPY', 1000))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/net-worth-history', history('KWD', 0.125))),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InEuroInDebt: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/net-worth-history', { ...history('EUR'), values: [100, -50.25, -300, -10] })),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/reports/net-worth-history', { dates: [], values: [], currency: 'USD' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no data/i)).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/net-worth-history')) }
