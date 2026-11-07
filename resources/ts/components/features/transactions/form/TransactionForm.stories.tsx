import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, tags } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionForm } from './TransactionForm'

const meta = {
    title: 'features/transactions/form/TransactionForm',
    component: TransactionForm,
    parameters: { a11y: overlayA11y },
    args: { onSubmit: fn(), formId: 'transaction-form' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/accounts', accounts),
            reply('get', '/categories', categories),
            reply('get', '/currencies', currencies),
            reply('get', '/tags', tags),
            reply('get', '/links', []),
        )
    },
} satisfies Meta<typeof TransactionForm>

export default meta
type Story = StoryObj<typeof meta>

const PAST = '2020-05-04'
const FUTURE = '2099-05-04'

const expense = {
    type: 'expense' as const,
    payments: [{ account_id: 1, amount: 20 }],
    default_category_id: 1,
    date: PAST,
    description: 'Weekly shop',
}

const saveButton = (canvasElement: HTMLElement) => within(canvasElement).getByRole('button', { name: /save/i })

export const Empty: Story = {}
export const Expense: Story = { args: { defaultValues: expense } }
export const Income: Story = { args: { defaultValues: { ...expense, type: 'income', default_category_id: 2 } } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('textbox', { name: /description/i })).toBeDisabled())
    },
}

// A new transaction starts with the first account and the default category of its type.
export const StartsWithDefaults: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await waitFor(() => expect(canvas.getAllByRole('combobox')[0]).toHaveTextContent(/cash/i))
    },
}

// The currency of the account stands at the end of the amount.
export const AmountShowsTheCurrency: Story = {
    args: { defaultValues: { ...expense, payments: [{ account_id: 2, amount: 8 }] } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('EUR')).toBeVisible()
    },
}

// What the balance becomes is shown under the payment while a transaction is dated today or earlier.
export const BalancePreview: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/120\.50/)).toBeVisible()
        await expect(within(canvasElement).getByText(/100\.50/)).toBeVisible()
    },
}

export const GoesNegative: Story = {
    args: { defaultValues: { ...expense, payments: [{ account_id: 1, amount: 500 }] } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/negative/i)).toBeVisible()
    },
}

// A transaction dated in the future waits as pending, so no balance is previewed.
export const PendingDate: Story = {
    args: { defaultValues: { ...expense, date: FUTURE } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/pending|scheduled|will/i)).toBeVisible()
        await expect(within(canvasElement).queryByText(/120\.50/)).toBeNull()
    },
}

// Clearing the date makes the transaction pending.
export const ClearingTheDateMakesItPending: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /clear/i }))
        await expect(await canvas.findByText(/no date set/i)).toBeVisible()
    },
}

export const Editing: Story = {
    args: { isEdit: true, originalAffectsBalance: true, defaultValues: expense },
}

// A transaction that already moved a balance cannot be moved into the future.
export const EditingCannotMoveToTheFuture: Story = {
    args: { isEdit: true, originalAffectsBalance: true, defaultValues: expense },
    play: async ({ canvasElement }) => {
        const date = await within(canvasElement).findByLabelText(/date/i)
        await expect(date).toHaveAttribute('max')
    },
}

// The transfer tabs are in the row but cannot be picked until the transfer form is back.
export const TransfersAreDisabled: Story = {
    beforeEach: ({ msw }) => {
        msw.use(
            reply('get', '/spaces', [space('admin'), { ...space('editor'), id: 2, name: 'Trip' }]),
            reply('get', '/links', [{ id: 2, name: 'Trip' }]),
        )
    },
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const transfer = await canvas.findByRole('button', { name: /transfer/i })
        await expect(transfer).toBeDisabled()
        await expect(await canvas.findByRole('button', { name: /space/i })).toBeDisabled()
    },
}

// An opener that asks for a transfer gets an expense.
export const TransferRequestedOpensAnExpense: Story = {
    args: { defaultValues: { type: 'transfer', payments: [{ account_id: 1, amount: 100 }] } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('button', { name: /expense/i })).toHaveAttribute('aria-pressed', 'true')
    },
}

export const SwitchingToIncomeKeepsEachCategory: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /income/i }))
        await waitFor(() => expect(canvas.getByRole('button', { name: /income/i })).toHaveAttribute('aria-pressed', 'true'))
    },
}

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(within(canvasElement).getAllByText(/select|greater|required|positive/i).length).toBeGreaterThan(0))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

// The form hands over what the API takes: the payment is negative for an expense, one line covers it.
export const Submits: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            type: 'expense',
            description: 'Weekly shop',
            date: PAST,
            payments: [{ account_id: 1, amount: -20 }],
            lines: [{ category_id: 1, amount: 20 }],
        })
    },
}

export const SubmitsAnIncomeAsPositive: Story = {
    args: { defaultValues: { ...expense, type: 'income', default_category_id: 2 } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            type: 'income',
            payments: [{ account_id: 1, amount: 20 }],
            lines: [{ category_id: 2, amount: 20 }],
        })
    },
}

// The note of a transaction saved with one line is kept when it is changed.
export const KeepsTheNoteOfTheLine: Story = {
    args: { defaultValues: { ...expense, default_note: '- milk\n- bread' } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            lines: [{ note: '- milk\n- bread' }],
        })
    },
}

// A date left empty is sent as null: the transaction is pending.
export const SubmitsPendingWithoutDate: Story = {
    args: { defaultValues: { ...expense, date: '' } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({ date: null })
    },
}

// Spending more than the account holds asks first.
export const OverdrawingAsksFirst: Story = {
    args: { defaultValues: { ...expense, payments: [{ account_id: 1, amount: 500 }] } },
    play: async ({ canvasElement, args }) => {
        await within(canvasElement).findByText(/negative/i)
        await userEvent.click(saveButton(canvasElement))
        const alert = await screen.findByRole('alertdialog')
        await expect(args.onSubmit).not.toHaveBeenCalled()
        await userEvent.click(within(alert).getByRole('button', { name: /continue|confirm|proceed|yes/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
    },
}
