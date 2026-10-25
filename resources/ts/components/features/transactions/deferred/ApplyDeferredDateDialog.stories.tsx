import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, currencies, pendingTransaction, transaction } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { ApplyDeferredDateDialog } from './ApplyDeferredDateDialog'

const meta = {
    title: 'features/transactions/deferred/ApplyDeferredDateDialog',
    component: ApplyDeferredDateDialog,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { open: true, onOpenChange: fn(), onConfirm: fn(), transaction: pendingTransaction({ date: '2020-03-01' }) },
} satisfies Meta<typeof ApplyDeferredDateDialog>

export default meta
type Story = StoryObj<typeof meta>

const confirm = async () => {
    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /^confirm$/i }))
    return dialog
}

// An overdue date can be kept, or replaced by today or another day.
export const Overdue: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getAllByRole('radio')).toHaveLength(3)
    },
}

// A date still ahead cannot be used: only today and another day remain.
export const DateInTheFuture: Story = {
    args: { transaction: pendingTransaction({ date: '2099-03-01' }) },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getAllByRole('radio')).toHaveLength(2)
    },
}

export const NoDate: Story = { args: { transaction: pendingTransaction({ date: null }) } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutTransaction: Story = { args: { transaction: null } }
export const AppliedToday: Story = {
    play: async ({ args }) => {
        await confirm()
        await expect(args.onConfirm).toHaveBeenCalledOnce()
    },
}

export const AppliedOnItsOwnDate: Story = {
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getAllByRole('radio')[1])
        await confirm()
        await expect((args.onConfirm as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({ date: '2020-03-01' })
    },
}

export const AppliedOnAnotherDay: Story = {
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getAllByRole('radio')[2])
        const date = await within(dialog).findByLabelText(/^date$/i)
        await userEvent.clear(date)
        await userEvent.type(date, '2020-02-02')
        await confirm()
        await expect((args.onConfirm as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({ date: '2020-02-02' })
    },
}

// A day in the future cannot be chosen.
export const AnotherDayCannotBeInTheFuture: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getAllByRole('radio')[2])
        const date = await within(dialog).findByLabelText(/^date$/i)
        await userEvent.clear(date)
        await userEvent.type(date, '2099-01-01')
        await expect(within(dialog).getByRole('button', { name: /^confirm$/i })).toBeDisabled()
    },
}

// An estimated amount is replaced by what was actually paid.
export const EstimatedAskForTheRealAmount: Story = {
    args: { transaction: pendingTransaction({ isEstimated: true, amount: 100, date: '2020-03-01' }) },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        const amount = within(dialog).getByRole('spinbutton')
        await expect(amount).toHaveValue(100)
        await userEvent.clear(amount)
        await userEvent.type(amount, '87.25')
        await confirm()
        await waitFor(() => expect(args.onConfirm).toHaveBeenCalledOnce())
        await expect((args.onConfirm as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({ amount: 87.25 })
    },
}

export const EstimatedNeedsAnAmount: Story = {
    args: { transaction: pendingTransaction({ isEstimated: true, amount: 100, date: '2020-03-01' }) },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.clear(within(dialog).getByRole('spinbutton'))
        await expect(within(dialog).getByRole('button', { name: /^confirm$/i })).toBeDisabled()
    },
}

// Between currencies, both the amount sent and the amount received are asked for.
export const EstimatedTransferBetweenCurrencies: Story = {
    args: {
        transaction: pendingTransaction({
            type: 'transfer', isEstimated: true, amount: 100, toAmount: 15000, account: accounts[0],
            toAccount: accounts[2], date: '2020-03-01',
        }),
    },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getAllByRole('spinbutton')).toHaveLength(2)
    },
}

// Paying more than the account holds asks before it is applied.
export const OverdrawingAsksFirst: Story = {
    args: {
        transaction: transaction({
            status: 'pending',
            amount: 500,
            date: '2020-03-01',
            account: accounts[0],
            actions: { edit: true, duplicate: false, delete: true, confirm: true, skip: true },
        }),
    },
    play: async ({ args }) => {
        await confirm()
        const alert = await screen.findByRole('alertdialog')
        await expect(args.onConfirm).not.toHaveBeenCalled()
        await userEvent.click(within(alert).getByRole('button', { name: /continue|confirm|proceed|yes/i }))
        await waitFor(() => expect(args.onConfirm).toHaveBeenCalledOnce())
    },
}
