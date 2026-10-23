import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { budget, categories, currencies } from '@/test/fixtures'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { BudgetRow } from './BudgetRow'

const [, , jpy] = currencies
const progress = (spent: number, total: number, exceeded = false) => ({
    spent,
    remaining: total - spent,
    percent: Math.round((spent / total) * 100),
    periodStart: '2026-10-01',
    periodEnd: '2026-10-31',
    isExceeded: exceeded,
})

const meta = {
    title: 'features/budgets/BudgetRow',
    component: BudgetRow,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { budget: budget(), onEdit: fn(), onDelete: fn() },
} satisfies Meta<typeof BudgetRow>

export default meta
type Story = StoryObj<typeof meta>

export const UnderLimit: Story = {}
export const NearlySpent: Story = { args: { budget: budget({ progress: progress(285, 300) }) } }
export const Exceeded: Story = { args: { budget: budget({ progress: progress(340, 300, true) }) } }
export const NothingSpent: Story = { args: { budget: budget({ progress: progress(0, 300) }) } }
export const NoProgressYet: Story = { args: { budget: budget({ progress: null }) } }
export const Inactive: Story = { args: { budget: budget({ isActive: false }) } }
export const AllExpenses: Story = { args: { budget: budget({ isGlobal: true, categories: [] }) } }
export const OneCategoryIcon: Story = { args: { budget: budget({ categories: [categories[0]] }) } }
export const TwoCategoryIcons: Story = {
    args: { budget: budget({ categories: [categories[0], categories[2]] }) },
}
export const NoCategoryIcons: Story = {
    args: { budget: budget({ categories: [{ ...categories[0], icon: null, color: null }] }) },
}
export const YenBudget: Story = {
    args: { budget: budget({ name: 'Trip', currency: jpy, amount: 100000, progress: progress(65000, 100000) }) },
}
export const LongName: Story = {
    args: { budget: budget({ name: 'A budget name far too long for the line it has to fit on in a list' }) },
}
export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getAllByText('Groceries')[0])
        await expect(args.onEdit).toHaveBeenCalledWith(args.budget)
    },
}

export const Deleted: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1)
    },
}

