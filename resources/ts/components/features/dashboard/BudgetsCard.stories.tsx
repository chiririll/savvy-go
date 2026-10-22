import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { budget, categories, currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { BudgetsCard } from './BudgetsCard'

const progress = (spent: number, total: number, exceeded = false) => ({
    spent,
    remaining: total - spent,
    percent: (spent / total) * 100,
    periodStart: '2026-10-01',
    periodEnd: '2026-10-31',
    isExceeded: exceeded,
})

const meta = {
    title: 'features/dashboard/BudgetsCard',
    component: BudgetsCard,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        budgets: [
            budget({ id: 1 }),
            budget({ id: 2, name: 'Transport', categories: [categories[2]], amount: 100, progress: progress(95, 100) }),
            budget({ id: 3, name: 'Everything', isGlobal: true, categories: [], amount: 2000, progress: progress(2300, 2000, true) }),
        ],
    },
} satisfies Meta<typeof BudgetsCard>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {}
export const Empty: Story = {
    args: { budgets: [] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('link', { name: /create/i })).toHaveAttribute('href', '/budgets?create=1')
    },
}
export const Exceeded: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('115%')).toBeVisible()
    },
}
export const NoProgress: Story = { args: { budgets: [budget({ progress: null })] } }
export const NoCategories: Story = { args: { budgets: [budget({ categories: [] })] } }
export const InYen: Story = {
    args: { budgets: [budget({ currency: currencies[2], amount: 100000, progress: progress(65000, 100000) })] },
}
export const LongName: Story = { args: { budgets: [budget({ name: 'A budget name much too long for its tile in the dashboard grid' })] } }
export const InRussian: Story = { globals: { locale: 'ru' } }

export const LinksToTheBudget: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('link', { name: /Groceries/ })).toHaveAttribute('href', '/budgets?edit=1')
    },
}
