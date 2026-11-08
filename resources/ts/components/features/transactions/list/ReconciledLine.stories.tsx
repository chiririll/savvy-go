import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { reconciliationDay } from '@/test/reconciliation'

import { ReconciledLine } from './ReconciledLine'

const meta = {
    title: 'features/transactions/list/ReconciledLine',
    component: ReconciledLine,
    args: { day: reconciliationDay(), baseCurrency: 'USD', onOpenDetails: fn() },
} satisfies Meta<typeof ReconciledLine>

export default meta
type Story = StoryObj<typeof meta>

// A day that is exact says only that it was reconciled.
export const Exact: Story = {
    args: { day: reconciliationDay({ differenceBase: 0 }) },
    play: async ({ canvasElement }) => {
        const label = within(canvasElement).getByText('Reconciled')
        await expect(label).toHaveClass('text-muted-foreground')
        await expect(label.children).toHaveLength(0)
        await expect(label).toHaveTextContent(/^Reconciled$/)
    },
}

export const Positive: Story = {
    args: { day: reconciliationDay({ differenceBase: 12.5 }) },
    play: async ({ canvasElement }) => {
        const label = within(canvasElement).getByText('Reconciled')
        await expect(label).toHaveClass('text-success')
        await expect(label).toHaveTextContent(/\+.*12\.5/)
    },
}

export const Negative: Story = {
    args: { day: reconciliationDay({ differenceBase: -7.25 }) },
    play: async ({ canvasElement }) => {
        const label = within(canvasElement).getByText('Reconciled')
        await expect(label).toHaveClass('text-destructive')
        await expect(label).toHaveTextContent(/-.*7\.25/)
    },
}

// A point without a rate is left out of the sum, and a warning says so.
export const Unconverted: Story = {
    args: { day: reconciliationDay({ differenceBase: 3, unconverted: true }) },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByLabelText(/no exchange rate/i)).toBeInTheDocument()
    },
}

// The sum is shortened to fit beside the rule.
export const CompactAmount: Story = {
    args: { day: reconciliationDay({ differenceBase: 1500 }) },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('Reconciled')).toHaveTextContent(/\+.*1\.5K/)
    },
}

export const OpensDetails: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /reconciliation details/i }))
        await expect(args.onOpenDetails).toHaveBeenCalledOnce()
    },
}
