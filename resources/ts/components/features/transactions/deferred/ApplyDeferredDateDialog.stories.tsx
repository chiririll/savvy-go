import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, currencies, line, payment, pendingTransaction } from '@/test/fixtures'
import type { Transaction } from '@/types'
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

// The open occurrence of a recurring rule: its payments and lines are what was expected.
const occurrence = (overrides: Partial<Transaction> = {}) =>
    pendingTransaction({
        date: '2020-03-01',
        recurringTransactionId: 5,
        restrictions: [{ reason: 'recurring_pending', message: '' }],
        actions: { edit: false, editDetails: false, duplicate: false, delete: true, confirm: true, skip: true },
        payments: [payment({ amount: -100 })],
        lines: [line({ amount: 100 })],
        ...overrides,
    })

const lastCall = (args: { onConfirm?: unknown }) => (args.onConfirm as ReturnType<typeof fn>).mock.calls[0][0]

// A transaction that is not an occurrence has no amounts to correct.
export const OrdinaryPendingAsksNoAmounts: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).queryByRole('spinbutton')).toBeNull()
    },
}

// An occurrence shows the expected amount of its payment; the line that follows it is not asked for.
export const OccurrenceAsksForTheRealAmount: Story = {
    args: { transaction: occurrence() },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        const amount = within(dialog).getByRole('spinbutton')
        await expect(within(dialog).getAllByRole('spinbutton')).toHaveLength(1)
        await expect(amount).toHaveValue(100)
        await userEvent.clear(amount)
        await userEvent.type(amount, '87.25')
        await confirm()
        await waitFor(() => expect(args.onConfirm).toHaveBeenCalledOnce())
        await expect(lastCall(args)).toMatchObject({
            date: expect.any(String),
            payments: [{ account_id: 1, amount: -87.25 }],
            lines: [{ category_id: 1, amount: 87.25 }],
        })
    },
}

// The expected amounts stand when nothing is changed: only the date is sent.
export const OccurrenceKeepsItsAmounts: Story = {
    args: { transaction: occurrence() },
    play: async ({ args }) => {
        await confirm()
        await waitFor(() => expect(args.onConfirm).toHaveBeenCalledOnce())
        const options = lastCall(args)
        await expect(options).not.toHaveProperty('payments')
        await expect(options).not.toHaveProperty('lines')
    },
}

export const OccurrenceNeedsAnAmount: Story = {
    args: { transaction: occurrence() },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.clear(within(dialog).getByRole('spinbutton'))
        await expect(within(dialog).getByRole('button', { name: /^confirm$/i })).toBeDisabled()
    },
}

// Between currencies, every payment is asked for, and so is every line.
export const OccurrenceInTwoCurrencies: Story = {
    args: {
        transaction: occurrence({
            payments: [payment({ amount: -30 }), payment({ account: accounts[1], accountId: 2, amount: -10 })],
            lines: [line({ amount: 30 }), line({ seq: 1, amount: 10, currencyId: 2 })],
        }),
    },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        const amounts = within(dialog).getAllByRole('spinbutton')
        await expect(amounts).toHaveLength(4)
        await userEvent.clear(amounts[1])
        await userEvent.type(amounts[1], '12')
        await userEvent.clear(amounts[3])
        await userEvent.type(amounts[3], '12')
        await confirm()
        await waitFor(() => expect(args.onConfirm).toHaveBeenCalledOnce())
        await expect(lastCall(args)).toMatchObject({
            payments: [{ amount: -30 }, { amount: -12 }],
            lines: [{ amount: 30 }, { amount: 12 }],
        })
    },
}

// A transfer has two payments and no lines: the money that arrives is asked for as well.
export const OccurrenceOfATransfer: Story = {
    args: {
        transaction: occurrence({
            type: 'transfer',
            payments: [payment({ amount: -100 }), payment({ account: accounts[2], accountId: 3, amount: 15000 })],
            lines: [],
        }),
    },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        const amounts = within(dialog).getAllByRole('spinbutton')
        await expect(amounts).toHaveLength(2)
        await userEvent.clear(amounts[1])
        await userEvent.type(amounts[1], '14800')
        await confirm()
        await waitFor(() => expect(args.onConfirm).toHaveBeenCalledOnce())
        await expect(lastCall(args)).toMatchObject({ payments: [{ amount: -100 }, { amount: 14800 }] })
        await expect(lastCall(args)).not.toHaveProperty('lines')
    },
}

// Paying more than the account holds asks before it is applied.
export const OverdrawingAsksFirst: Story = {
    args: {
        transaction: pendingTransaction({
            date: '2020-03-01',
            payments: [payment({ amount: -500 })],
            lines: [line({ amount: 500 })],
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

// The real amount of an occurrence can take the account below zero too.
export const RealAmountOverdrawsAsksFirst: Story = {
    args: { transaction: occurrence() },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        const amount = within(dialog).getByRole('spinbutton')
        await userEvent.clear(amount)
        await userEvent.type(amount, '500')
        await confirm()
        await screen.findByRole('alertdialog')
        await expect(args.onConfirm).not.toHaveBeenCalled()
    },
}
