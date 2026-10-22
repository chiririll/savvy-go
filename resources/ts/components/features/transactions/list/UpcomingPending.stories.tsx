import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, currencies, pendingTransaction } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { UpcomingPendingCard } from './UpcomingPendingCard'
import { UpcomingPendingStrip } from './UpcomingPendingStrip'

const overdue = pendingTransaction({ id: 1, date: '2020-01-10', description: 'Phone bill' })
const soon = pendingTransaction({ id: 2, date: '2099-01-10', description: 'Rent', amount: 1100 })

const meta = {
    title: 'features/transactions/list/UpcomingPending',
    component: UpcomingPendingCard,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { transaction: soon, onConfirm: fn(), onSkip: fn() },
} satisfies Meta<typeof UpcomingPendingCard>

export default meta
type Story = StoryObj<typeof meta>

export const Upcoming: Story = {}
export const Overdue: Story = { args: { transaction: overdue } }
export const Income: Story = { args: { transaction: pendingTransaction({ type: 'income', description: 'Salary', amount: 3200 }) } }
export const Transfer: Story = { args: { transaction: pendingTransaction({ type: 'transfer', description: null, toAccount: accounts[1] }) } }
export const Estimated: Story = { args: { transaction: pendingTransaction({ isEstimated: true, description: 'Electricity' }) } }
export const NoDate: Story = { args: { transaction: pendingTransaction({ date: null }) } }
export const InYen: Story = { args: { transaction: pendingTransaction({ account: accounts[2], amount: 80000 }) } }
export const LongTitle: Story = { args: { transaction: pendingTransaction({ description: 'A title far too long for a card only this wide' }) } }
export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const CannotBeSkipped: Story = {
    args: { transaction: pendingTransaction({ actions: { edit: true, duplicate: false, delete: true, confirm: true, skip: false } }) },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.queryByRole('button', { name: /skip/i })).toBeNull()
        await expect(canvas.getByRole('button', { name: /confirm/i })).toBeVisible()
    },
}

export const Confirmed: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /confirm/i }))
        await expect(args.onConfirm).toHaveBeenCalledWith(args.transaction)
    },
}

export const SkippedAfterAsking: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /skip/i }))
        const dialog = await screen.findByRole('alertdialog')
        await expect(args.onSkip).not.toHaveBeenCalled()
        await userEvent.click(within(dialog).getByRole('button', { name: /^skip$/i }))
        await waitFor(() => expect(args.onSkip).toHaveBeenCalledWith(2))
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const Strip: Story = {
    render: (args) => <UpcomingPendingStrip transactions={[overdue, soon]} onConfirm={args.onConfirm} onSkip={args.onSkip} />,
}

export const StripOnlyUpcoming: Story = {
    render: (args) => <UpcomingPendingStrip transactions={[soon]} onConfirm={args.onConfirm} onSkip={args.onSkip} />,
}

export const StripEmpty: Story = {
    render: (args) => <UpcomingPendingStrip transactions={[]} onConfirm={args.onConfirm} onSkip={args.onSkip} />,
    play: async ({ canvasElement }) => {
        await expect(canvasElement.querySelector('[data-slot]')).toBeNull()
    },
}

export const StripReadOnly: Story = {
    render: (args) => (
        <UpcomingPendingStrip transactions={[overdue, soon]} isReadOnly onConfirm={args.onConfirm} onSkip={args.onSkip} />
    ),
}
