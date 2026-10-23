import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { MetricCard } from './MetricCard'

const meta = {
    title: 'pages/reports/MetricCard',
    component: MetricCard,
    parameters: { layout: 'padded' },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    decorators: [(Story) => <div className="max-w-xs"><Story /></div>],
    args: {
        title: 'Income',
        value: 1500,
        previousValue: 1000,
        sparklineData: [3, 5, 4, 8, 6, 9, 7],
        type: 'income',
        compareWith: 'previous_period',
        currency: 'USD',
    },
} satisfies Meta<typeof MetricCard>

export default meta
type Story = StoryObj<typeof meta>

export const Income: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('Income')).toBeVisible()
        await expect(canvas.getByText(/1,500\.00 \$/)).toBeVisible()
        await expect(canvas.getByText('50.0%')).toBeVisible()
        await expect(canvas.getByText(/\(\+500\.00 \$\)/)).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

// A zero-decimal currency shows no fraction, in the figure and in the change.
export const YenHasNoDecimals: Story = {
    args: { currency: 'JPY', value: 182_000, previousValue: 200_000 },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/182,000 ¥/)).toBeVisible()
        await expect(canvas.getByText(/\(-18,000 ¥\)/)).toBeVisible()
        await expect(canvas.getByText('9.0%')).toBeVisible()
    },
}

// A currency with three decimals keeps all three.
export const DinarKeepsThreeDecimals: Story = {
    args: { currency: 'KWD', value: 12.345, previousValue: 10 },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/12\.345/)).toBeVisible()
    },
}

export const ExpenseThatFell: Story = {
    args: { title: 'Expenses', type: 'expense', currency: 'EUR', value: 800, previousValue: 1000 },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/800\.00 €/)).toBeVisible()
        await expect(canvas.getByText('20.0%')).toBeVisible()
        await expect(canvas.getByText(/\(-200\.00 €\)/)).toBeVisible()
    },
}

export const SavingsRate: Story = {
    args: { title: 'Savings rate', type: 'percent', value: 32.46, previousValue: 30, currency: undefined },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('32.5%')).toBeVisible()
        await expect(canvas.getByText(/\+2\.5/)).toBeVisible()
    },
}

export const NoComparison: Story = {
    args: { compareWith: 'none' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText(/%/)).toBeNull()
    },
}

// Nothing to compare with when the previous period was empty.
export const PreviousWasZero: Story = {
    args: { previousValue: 0 },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText(/%/)).toBeNull()
    },
}

export const Suffix: Story = { args: { suffix: ' / mo' } }
