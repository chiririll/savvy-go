import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import type { BalanceHistoryResponse } from '@/api/accounts'
import { expectChartDrawn } from '@/pages/reports/story-fixtures'
import { hang, raw } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { BalanceDynamicsChart } from './BalanceDynamicsChart'

const history = (overrides: Partial<BalanceHistoryResponse> = {}): BalanceHistoryResponse => ({
    dates: ['2030-01-01', '2030-01-02', '2030-01-03'],
    series: [
        { id: null, name: 'Total', type: 'total', data: [1100, 1250.5, 1300], currency: 'USD' },
        { id: 1, name: 'Cash', type: 'cash', data: [100, 250.5, 300], currency: 'USD' },
        { id: 3, name: 'Yen wallet', type: 'cash', data: [1000, 1000, 1000], native_data: [200000, 200000, 200000], currency: 'JPY' },
    ],
    currency: 'USD',
    decimals: 2,
    ...overrides,
})

const meta = {
    title: 'features/accounts/BalanceDynamicsChart',
    component: BalanceDynamicsChart,
    args: { startDate: '2030-01-01', endDate: '2030-01-03' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(raw('get', '/accounts-balance-history', history()))
    },
} satisfies Meta<typeof BalanceDynamicsChart>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Balance dynamics')).toBeVisible()
        await expectChartDrawn(canvasElement)
        await expect(await canvas.findByRole('button', { name: /Cash/ })).toBeVisible()
    },
}

export const HidesASeries: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const cash = await canvas.findByRole('button', { name: /Cash/ })
        await expect(cash).toHaveAttribute('aria-pressed', 'true')
        await userEvent.click(cash)
        await waitFor(() => expect(cash).toHaveAttribute('aria-pressed', 'false'))
        await expectChartDrawn(canvasElement)
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            raw(
                'get',
                '/accounts-balance-history',
                history({
                    currency: 'JPY',
                    decimals: 0,
                    series: [{ id: null, name: 'Total', type: 'total', data: [120000, 150000, 90000], currency: 'JPY' }],
                }),
            ),
        ),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) =>
        msw.use(
            raw(
                'get',
                '/accounts-balance-history',
                history({
                    currency: 'KWD',
                    decimals: 3,
                    series: [{ id: null, name: 'Total', type: 'total', data: [12.345, 15.5, 9.125], currency: 'KWD' }],
                }),
            ),
        ),
    play: async ({ canvasElement }) => {
        await expectChartDrawn(canvasElement)
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(raw('get', '/accounts-balance-history', history({ dates: [], series: [] }))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('No data for this period')).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/accounts-balance-history')) }
