import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { BalanceSeriesLegend } from './BalanceSeriesLegend'
import type { LabeledSeries } from './balance-dynamics'

const series: LabeledSeries[] = [
    { key: 'total', id: null, name: 'Total', type: 'total', data: [100, 120], currency: 'USD', label: 'Total', color: '#6366f1' },
    { key: 'account-1', id: 1, name: 'Cash', type: 'cash', data: [10, 20], currency: 'USD', label: 'Cash', color: '#16a34a' },
    { key: 'account-3', id: 3, name: 'Yen wallet', type: 'cash', data: [1000], native_data: [15000], currency: 'JPY', label: 'Yen wallet', color: '#d97706' },
    {
        key: 'account-4', id: 4, name: 'Kuwait savings', type: 'bank', data: [3], native_data: [12.345],
        currency: 'KWD', label: 'Kuwait savings', color: '#2563eb',
    },
]

const meta = {
    tags: ['desktop-only'],
    title: 'features/accounts/BalanceSeriesLegend',
    component: BalanceSeriesLegend,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { series, hiddenKeys: new Set<string>(), onToggle: () => {} },
    render: (args) => {
        const [hidden, setHidden] = useState(args.hiddenKeys)
        const toggle = (key: string) =>
            setHidden((current) => {
                const next = new Set(current)
                if (!next.delete(key)) next.add(key)
                return next
            })
        return <BalanceSeriesLegend {...args} hiddenKeys={hidden} onToggle={toggle} />
    },
} satisfies Meta<typeof BalanceSeriesLegend>

export default meta
type Story = StoryObj<typeof meta>

// Each series shows its last value in its own currency: 0, 2 and 3 decimals.
export const AllVisible: Story = {}
export const SomeHidden: Story = { args: { hiddenKeys: new Set(['account-1', 'account-4']) } }
export const SingleSeries: Story = { args: { series: series.slice(0, 1) } }
export const NoSeries: Story = { args: { series: [] } }

export const Toggled: Story = {
    play: async ({ canvasElement }) => {
        const cash = within(canvasElement).getByRole('button', { name: /Cash/ })
        await expect(cash).toHaveAttribute('aria-pressed', 'true')
        await userEvent.click(cash)
        await expect(cash).toHaveAttribute('aria-pressed', 'false')
    },
}
