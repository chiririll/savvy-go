import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { accounts, categories, currencies, recurring } from '@/test/fixtures'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { RecurringRow } from './RecurringRow'

const meta = {
    title: 'features/recurring/RecurringRow',
    component: RecurringRow,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { recurring: recurring(), onEdit: fn(), onDelete: fn() },
} satisfies Meta<typeof RecurringRow>

export default meta
type Story = StoryObj<typeof meta>

export const Expense: Story = {}
export const Income: Story = {
    args: { recurring: recurring({ type: 'income', description: 'Salary', amount: 3200, category: categories[1] }) },
}
export const Transfer: Story = {
    args: {
        recurring: recurring({ type: 'transfer', description: 'To savings', category: null, toAccount: accounts[1], toAmount: 920 }),
    },
}
export const Estimated: Story = { args: { recurring: recurring({ isEstimated: true, description: 'Electricity' }) } }
export const Paused: Story = { args: { recurring: recurring({ isActive: false }) } }

// A run date that has passed is called out.
export const OverdueRun: Story = { args: { recurring: recurring({ nextRunDate: '2020-01-01' }) } }
export const PausedOverdueRunIsNotCalledOut: Story = {
    args: { recurring: recurring({ nextRunDate: '2020-01-01', isActive: false }) },
}
export const NoDescription: Story = { args: { recurring: recurring({ description: null }) } }
export const YenAmount: Story = {
    args: { recurring: recurring({ amount: 80000, account: accounts[2], description: 'Rent in Tokyo' }) },
}
export const ThreeDecimals: Story = {
    args: { recurring: recurring({ amount: 12.345, account: accounts[3], description: 'Fee' }) },
}
export const LongDescription: Story = {
    args: { recurring: recurring({ description: 'A description far too long for the single line of a row in a list' }) },
}
export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Rent'))
        await expect(args.onEdit).toHaveBeenCalledWith(args.recurring)
    },
}

export const Deleted: Story = {
    tags: ['desktop-only'],
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1)
    },
}

