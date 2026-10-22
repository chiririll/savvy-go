import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { categories, currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { ExpensesByCategoryCard } from './ExpensesByCategoryCard'

const summary = {
    data: [
        { ...categories[0], totalAmount: 420 },
        { ...categories[2], totalAmount: 180.5 },
        { ...categories[1], id: 9, name: 'Cinema', totalAmount: 60 },
    ],
    total: 660.5,
    currency: 'USD',
}

const meta = {
    title: 'features/dashboard/ExpensesByCategoryCard',
    component: ExpensesByCategoryCard,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { summary, currency: 'USD' },
} satisfies Meta<typeof ExpensesByCategoryCard>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Groceries')).toBeVisible()
    },
}
export const NoData: Story = { args: { summary: { ...summary, data: [] } } }
export const NothingSpent: Story = { args: { summary: { ...summary, data: summary.data.map((c) => ({ ...c, totalAmount: 0 })) } } }
export const Loading: Story = { args: { summary: undefined } }
export const OneCategory: Story = { args: { summary: { ...summary, data: [summary.data[0]] } } }
export const BuiltInNames: Story = {
    args: {
        summary: {
            ...summary,
            data: [
                { ...categories[0], name: '#GROCERIES', totalAmount: 100 },
                { ...categories[2], name: '#TRANSPORT', totalAmount: 50 },
            ],
        },
    },
}
export const InRussian: Story = { ...BuiltInNames, globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }
export const CurrencyFromTheSummary: Story = {
    args: { currency: undefined, summary: { ...summary, currency: 'JPY' } },
}
