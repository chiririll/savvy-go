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

const lineGroup = (canvasElement: HTMLElement) => within(canvasElement).findByRole('group', { name: /lines|строки/i })
const amountsOf = (group: HTMLElement) => within(group).getAllByRole('textbox', { name: /^amount$/i }) as HTMLInputElement[]
const closeButton = /close the editor|закрыть редактор/i
const noteButtons = /format the note|форматировать заметку/i

/** The accounts come from the mocked API: the form is ready when the first balance is drawn. */
const ready = (canvasElement: HTMLElement) =>
    waitFor(() => expect(within(canvasElement).getAllByTestId('balance-after')[0]).not.toBeEmptyDOMElement())

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
        await ready(canvasElement)
        await waitFor(() => expect(within(canvasElement).getByRole('textbox', { name: /description/i })).toBeDisabled())
    },
}

// A new transaction starts with the first account and the default category of its type.
export const StartsWithDefaults: Story = {
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await waitFor(() => expect(canvas.getAllByRole('combobox')[0]).toHaveTextContent(/cash/i))
    },
}

// The currency of the account stands at the end of the amount.
export const AmountShowsTheCurrency: Story = {
    args: { defaultValues: { ...expense, payments: [{ account_id: 2, amount: 8 }] } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        // The line is in the currency of the payment, so the code is shown twice.
        await expect((await within(canvasElement).findAllByText('EUR'))[0]).toBeVisible()
    },
}

// What the balance becomes is shown under the payment while a transaction is dated today or earlier.
export const BalancePreview: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const balance = await within(canvasElement).findByTestId('balance-after')
        await expect(balance).toHaveTextContent(/120\.50.*100\.50/)
    },
}

// The amount takes digits and one separator, no more decimals than the currency has; a comma works as the point.
export const AmountTakesDigitsOnly: Story = {
    args: { defaultValues: { ...expense, payments: [{ account_id: 1, amount: '' as unknown as number }] } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        // The first amount is the payment's; the line below it has one too.
        const amount = (await within(canvasElement).findAllByRole('textbox', { name: /^amount$/i }))[0]
        await userEvent.type(amount, '-1e2,567x')
        await expect(amount).toHaveValue('12.56')
    },
}

export const GoesNegative: Story = {
    args: { defaultValues: { ...expense, payments: [{ account_id: 1, amount: 500 }] } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await expect(await within(canvasElement).findByText(/negative/i)).toBeVisible()
    },
}

// A transaction dated in the future waits as pending, so no balance is previewed.
export const PendingDate: Story = {
    args: { defaultValues: { ...expense, date: FUTURE } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await expect(await within(canvasElement).findByText(/pending|scheduled|will/i)).toBeVisible()
        await expect(within(canvasElement).queryByText('→')).toBeNull()
    },
}

// Clearing the date makes the transaction pending.
export const ClearingTheDateMakesItPending: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
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
        await ready(canvasElement)
        const date = await within(canvasElement).findByLabelText(/date/i)
        await expect(date).toHaveAttribute('max')
    },
}

// The transfer tabs are pickable: a transfer to a linked space needs a link the user can write to.
export const TransferTabsCanBePicked: Story = {
    beforeEach: ({ msw }) => {
        msw.use(
            reply('get', '/spaces', [space('admin'), { ...space('editor'), id: 2, name: 'Trip' }]),
            reply('get', '/links', [{ id: 2, name: 'Trip' }]),
        )
    },
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('button', { name: /transfer/i })).toBeEnabled()
        await expect(await canvas.findByRole('button', { name: /space/i })).toBeEnabled()
    },
}

export const SwitchingToIncomeKeepsEachCategory: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /income/i }))
        await waitFor(() => expect(canvas.getByRole('button', { name: /income/i })).toHaveAttribute('aria-pressed', 'true'))
    },
}

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await ready(canvasElement)
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(within(canvasElement).getAllByText(/select|greater|required|positive/i).length).toBeGreaterThan(0))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

// The form hands over what the API takes: the payment is negative for an expense, one line covers it.
export const Submits: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement, args }) => {
        await ready(canvasElement)
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
        await ready(canvasElement)
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
        await ready(canvasElement)
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
        await ready(canvasElement)
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({ date: null })
    },
}

// Spending more than the account holds asks first.
export const OverdrawingAsksFirst: Story = {
    args: { defaultValues: { ...expense, payments: [{ account_id: 1, amount: 500 }] } },
    play: async ({ canvasElement, args }) => {
        await ready(canvasElement)
        await within(canvasElement).findByText(/negative/i)
        await userEvent.click(saveButton(canvasElement))
        const alert = await screen.findByRole('alertdialog')
        await expect(args.onSubmit).not.toHaveBeenCalled()
        await userEvent.click(within(alert).getByRole('button', { name: /continue|confirm|proceed|yes/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
    },
}

const split = {
    ...expense,
    lines: [
        { category_id: 3, currency_id: null, amount: 12, note: null },
        { category_id: 1, currency_id: null, amount: 8, note: null },
    ],
}

// The first (+) makes two lines: the first takes the whole payment, so the default category has nothing to take.
export const SplitByCategory: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /split by category|разбить по категориям/i }))
        const amounts = amountsOf(await lineGroup(canvasElement))
        await expect(amounts).toHaveLength(2)
        await expect(amounts[0]).toHaveValue('20')
        await expect(amounts[1]).toHaveValue('')
        await expect(canvas.getAllByRole('combobox')[1]).toBeDisabled()
    },
}

// The cross of a line takes it away; the last one gone, the category is the transaction's again.
export const RemovingTheLines: Story = {
    args: { defaultValues: split },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await userEvent.click((await canvas.findAllByRole('button', { name: /remove the line|удалить строку/i }))[0])
        await userEvent.click(canvas.getByRole('button', { name: /remove the line|удалить строку/i }))
        await waitFor(() => expect(canvas.queryByRole('group', { name: /lines|строки/i })).toBeNull())
    },
}

// Lines that cover the payment leave nothing for the default category: it is locked and no warning shows.
export const LinesCoverThePayment: Story = {
    args: { defaultValues: split },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await lineGroup(canvasElement)
        await waitFor(() => expect(canvas.getAllByRole('combobox')[1]).toBeDisabled())
        await expect(canvas.queryByRole('button', { name: /8\.00|8,00/ })).toBeNull()
    },
}

// Lines that cover less: the default category opens, and the one warning under the lines says how much goes to it.
export const LinesLeaveARest: Story = {
    args: { defaultValues: { ...split, lines: [split.lines[0]] } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('status')).toHaveTextContent(/8\.00|8,00/)
        await expect(canvas.getAllByRole('combobox')[1]).toBeEnabled()
    },
}

// Lines above the payment are only a warning; the form still saves.
export const LinesAboveThePayment: Story = {
    args: { defaultValues: { ...split, lines: [{ ...split.lines[0], amount: 25 }] } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await expect(await within(canvasElement).findByRole('status')).toHaveTextContent(/5\.00|5,00/)
        await userEvent.click(saveButton(canvasElement))
    },
}

// A single line has no note: the title says what it is. From two lines on every line has one.
export const NotesFromTwoLines: Story = {
    args: { defaultValues: { ...split, lines: [split.lines[0]] } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await lineGroup(canvasElement)
        await expect(canvas.queryByRole('button', { name: noteButtons })).toBeNull()
        await userEvent.click(canvas.getByRole('button', { name: /split by category|разбить по категориям/i }))
        await waitFor(() => expect(canvas.getAllByRole('button', { name: noteButtons })).toHaveLength(2))
    },
}

// The pencil opens the markdown editor in place of the compact note, and the cross folds it back.
export const NoteOpensInTheEditor: Story = {
    args: { defaultValues: split },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await userEvent.click((await canvas.findAllByRole('button', { name: noteButtons }))[0])
        await userEvent.click(await canvas.findByRole('button', { name: closeButton }))
        await waitFor(() => expect(canvas.queryByRole('button', { name: closeButton })).toBeNull())
        await expect(canvas.getAllByRole('button', { name: noteButtons })).toHaveLength(2)
    },
}

// A note that was saved opens closed: its text shows as written, and the pencil opens the editor.
export const SavedNoteOpensClosed: Story = {
    args: {
        defaultValues: { ...split, lines: [{ ...split.lines[0], note: '- milk\n- bread' }, split.lines[1]] },
    },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await expect(await within(canvasElement).findByDisplayValue(/- milk/)).toBeVisible()
        await expect(within(canvasElement).queryByRole('button', { name: closeButton })).toBeNull()
        await expect(within(canvasElement).getAllByRole('button', { name: noteButtons })).toHaveLength(2)
    },
}

// Accounts in two currencies: a line picks the currency it is in.
export const LineCurrencyFromTheAccounts: Story = {
    args: {
        defaultValues: {
            ...split,
            payments: [
                { account_id: 1, amount: 10 },
                { account_id: 2, amount: 8 },
            ],
            lines: [{ category_id: 3, currency_id: 2, amount: 4, note: null }, split.lines[1]],
        },
    },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const group = await lineGroup(canvasElement)
        await expect(within(group).getAllByRole('combobox', { name: /currency of the line|валюта строки/i })).toHaveLength(2)
        await expect(within(group).getAllByRole('combobox', { name: /currency of the line|валюта строки/i })[0]).toHaveTextContent('EUR')
    },
}

// The lines and what they leave over are saved; the rest goes to the default category.
export const SubmitsLines: Story = {
    args: { defaultValues: { ...split, lines: [split.lines[0]] } },
    play: async ({ canvasElement, args }) => {
        await ready(canvasElement)
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            payments: [{ account_id: 1, amount: -20 }],
            lines: [
                { category_id: 3, amount: 12 },
                { category_id: 1, amount: 8 },
            ],
        })
    },
}

const addPayment = /add another account|добавить ещё счёт/i
const removePayment = /remove the payment|убрать платёж/i
const twoAccounts = {
    ...expense,
    payments: [
        { account_id: 1, amount: 10 },
        { account_id: 2, amount: 8 },
    ],
}

// The (+) at the end of the last payment adds a row on the first account no row uses.
export const AddingAPayment: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: addPayment }))
        // Two payments and the line.
        await waitFor(() => expect(canvas.getAllByRole('textbox', { name: /^amount$/i })).toHaveLength(3))
        await expect(canvas.getAllByRole('combobox')[1]).toHaveTextContent(/euro card/i)
    },
}

// An account is in a transaction once: the picker of a row does not offer the accounts of the others.
export const AccountsAreUnique: Story = {
    args: { defaultValues: twoAccounts },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await userEvent.click((await canvas.findAllByRole('combobox'))[1])
        const options = await screen.findAllByRole('option')
        await expect(options.some((option) => /cash/i.test(option.textContent ?? ''))).toBe(false)
        await expect(options.some((option) => /yen wallet/i.test(option.textContent ?? ''))).toBe(true)
    },
}

// With every account in use there is nothing left to add: the (+) is off.
export const AddingIsOffWhenNoAccountIsLeft: Story = {
    args: {
        defaultValues: {
            ...expense,
            payments: [1, 2, 3, 4].map((account_id) => ({ account_id, amount: 1 })),
        },
    },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await expect(await within(canvasElement).findByRole('button', { name: addPayment })).toBeDisabled()
    },
}

// The cross takes a payment away; the only one left cannot be removed.
export const RemovingAPayment: Story = {
    args: { defaultValues: twoAccounts },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const canvas = within(canvasElement)
        await userEvent.click((await canvas.findAllByRole('button', { name: removePayment }))[0])
        // The payment and the line.
        await waitFor(() => expect(canvas.getAllByRole('textbox', { name: /^amount$/i })).toHaveLength(2))
        await expect(canvas.queryByRole('button', { name: removePayment })).toBeNull()
    },
}

// Each account shows what it will hold; the date is today or earlier, so they are posted at once.
export const BalanceOfEachAccount: Story = {
    args: { defaultValues: twoAccounts },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await waitFor(() => expect(within(canvasElement).getAllByText(/balance after|баланс после/i)).toHaveLength(2))
    },
}

// One total per currency, once there are several payments.
export const TotalsPerCurrency: Story = {
    args: { defaultValues: twoAccounts },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const totals = await within(canvasElement).findByTestId('payment-totals')
        await expect(totals).toHaveTextContent(/10\.00/)
        await expect(totals).toHaveTextContent(/8\.00/)
    },
}

// The total is there from the start, so the form does not move when a second payment comes.
export const OnePaymentHasATotal: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await expect(await within(canvasElement).findByTestId('payment-totals')).toHaveTextContent(/20\.00/)
    },
}

// The balance of the account is there before an amount is typed.
export const BalanceBeforeAnAmount: Story = {
    args: { defaultValues: { ...expense, payments: [{ account_id: 1, amount: '' as unknown as number }] } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await expect(await within(canvasElement).findByTestId('balance-after')).toHaveTextContent(/120\.50/)
    },
}

// Two accounts below zero: one question lists both before anything is saved.
export const OverdrawingTwoAccountsAsksOnce: Story = {
    args: {
        defaultValues: {
            ...expense,
            payments: [
                { account_id: 1, amount: 500 },
                { account_id: 2, amount: 2000 },
            ],
            // With two payments the line is the user's to fill.
            lines: [{ category_id: 1, currency_id: null, amount: 500, note: '' }],
        },
    },
    play: async ({ canvasElement, args }) => {
        await ready(canvasElement)
        await within(canvasElement).findAllByText(/negative/i)
        await userEvent.click(saveButton(canvasElement))
        const alert = await screen.findByRole('alertdialog')
        await expect(within(alert).getAllByText(/^cash$/i).length).toBeGreaterThan(0)
        await expect(within(alert).getAllByText(/euro card/i).length).toBeGreaterThan(0)
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}
