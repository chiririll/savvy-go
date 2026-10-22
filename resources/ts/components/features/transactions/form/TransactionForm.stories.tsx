import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, tags } from '@/test/fixtures'
import { reply, slowly } from '@/test/api-mocks'
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
    account_id: 1,
    category_id: 1,
    amount: 20,
    date: PAST,
    description: 'Weekly shop',
}

const saveButton = (canvasElement: HTMLElement) => within(canvasElement).getByRole('button', { name: /save/i })

export const Empty: Story = {}
export const Expense: Story = { args: { defaultValues: expense } }
export const Income: Story = { args: { defaultValues: { ...expense, type: 'income', category_id: 2 } } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
export const InRussian: Story = { globals: { locale: 'ru' } }

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('textbox', { name: /description|notes/i })).toBeDisabled())
    },
}

// What the balance becomes is shown while a transaction is dated today or earlier.
export const BalancePreview: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/120\.50/)).toBeVisible()
        await expect(within(canvasElement).getByText(/100\.50/)).toBeVisible()
    },
}

export const GoesNegative: Story = {
    args: { defaultValues: { ...expense, amount: 500 } },
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

export const PendingMayBeEstimated: Story = {
    args: { defaultValues: { ...expense, date: FUTURE } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('checkbox', { name: /approximate/i })).toBeVisible()
    },
}

export const Transfer: Story = {
    args: { defaultValues: { type: 'transfer', account_id: 1, to_account_id: 2, amount: 100, date: PAST } },
}

// Between currencies the rate is editable and the received amount follows from it.
export const TransferBetweenCurrencies: Story = {
    args: { defaultValues: { type: 'transfer', account_id: 1, to_account_id: 3, amount: 100, date: PAST } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const rate = await canvas.findByRole('spinbutton', { name: /rate/i })
        await waitFor(() => expect(rate).toBeEnabled())
        // 1 USD buys 200 JPY in the sample rates: 100 USD arrive as 20,000 yen, in whole units.
        await waitFor(() => expect(rate).toHaveValue(200), { timeout: 4000 })
        await waitFor(() => expect(canvas.getByRole('spinbutton', { name: /receive/i })).toHaveValue(20000), { timeout: 4000 })
    },
}

// The accounts arrive before the currencies: the rate must wait for them, not open as 1.
export const RateWaitsForTheCurrencyList: Story = {
    beforeEach: ({ msw }) => msw.use(slowly('get', '/currencies', currencies, 600)),
    args: { defaultValues: { type: 'transfer', account_id: 1, to_account_id: 3, amount: 100, date: PAST } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const rate = await canvas.findByRole('spinbutton', { name: /rate/i })
        await waitFor(() => expect(rate).toHaveValue(200), { timeout: 5000 })
        await expect(canvas.getByRole('spinbutton', { name: /receive/i })).toHaveValue(20000)
    },
}

// Typing a rate recalculates the received amount when the field is left.
export const RateEdited: Story = {
    args: { defaultValues: { type: 'transfer', account_id: 1, to_account_id: 3, amount: 100, date: PAST } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const rate = await canvas.findByRole('spinbutton', { name: /rate/i })
        await waitFor(() => expect(rate).toHaveValue(200), { timeout: 4000 })
        await userEvent.clear(rate)
        await userEvent.type(rate, '150')
        await userEvent.tab()
        await waitFor(() => expect(canvas.getByRole('spinbutton', { name: /receive/i })).toHaveValue(15000), { timeout: 4000 })
    },
}

// Editing the received amount works the rate out from it.
export const ReceivedAmountEdited: Story = {
    args: { defaultValues: { type: 'transfer', account_id: 1, to_account_id: 3, amount: 100, date: PAST } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const received = await canvas.findByRole('spinbutton', { name: /receive/i })
        await waitFor(() => expect(received).toHaveValue(20000), { timeout: 4000 })
        // Clearing the field would put the calculated amount straight back, so the text is selected and replaced.
        const input = received as HTMLInputElement
        input.focus()
        input.select()
        await userEvent.keyboard('18000')
        await waitFor(() => expect(canvas.getByRole('spinbutton', { name: /rate/i })).toHaveValue(180), { timeout: 4000 })
    },
}

// Swapping puts the accounts the other way round and keeps both amounts.
export const AccountsSwapped: Story = {
    args: { defaultValues: { type: 'transfer', account_id: 1, to_account_id: 2, amount: 100, date: PAST } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await canvas.findByRole('spinbutton', { name: /rate/i })
        await userEvent.click(canvas.getByRole('button', { name: /swap/i }))
        await waitFor(() => expect(canvas.getByRole('spinbutton', { name: /send/i })).toBeVisible())
    },
}

// Within one currency the rate is 1 and cannot be changed.
export const TransferInOneCurrency: Story = {
    args: { defaultValues: { type: 'transfer', account_id: 1, to_account_id: 1, amount: 100, date: PAST } },
}

export const WithItems: Story = {
    args: {
        defaultValues: {
            ...expense,
            amount: 0,
            items: [
                { name: 'Bread', quantity: 2, price_per_unit: 2.5 },
                { name: 'Cheese', quantity: 1, price_per_unit: 12.5 },
            ],
        },
    },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('17.50')).toBeVisible()
    },
}

export const ItemsFixTheAmount: Story = {
    args: { defaultValues: { ...expense, amount: 0, items: [{ name: 'Bread', quantity: 2, price_per_unit: 2.5 }] } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('spinbutton', { name: /amount/i })).toHaveValue(5))
    },
}

export const ItemAdded: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /add item/i }))
        await expect(await within(canvasElement).findByRole('textbox', { name: /item name/i })).toBeVisible()
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

export const SpaceTransferOffered: Story = {
    beforeEach: ({ msw }) => {
        msw.use(
            reply('get', '/spaces', [space('admin'), { ...space('editor'), id: 2, name: 'Trip' }]),
            reply('get', '/links', [{ id: 2, name: 'Trip' }]),
        )
    },
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getAllByRole('button').length).toBeGreaterThan(3))
        await expect(await within(canvasElement).findByRole('button', { name: /space/i })).toBeVisible()
    },
}

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(within(canvasElement).getAllByText(/select|greater|required/i).length).toBeGreaterThan(0))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const Submits: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            type: 'expense',
            account_id: 1,
            category_id: 1,
            amount: 20,
            date: PAST,
        })
    },
}

// Spending more than the account holds asks first.
export const OverdrawingAsksFirst: Story = {
    args: { defaultValues: { ...expense, amount: 500 } },
    play: async ({ canvasElement, args }) => {
        await within(canvasElement).findByText(/negative/i)
        await userEvent.click(saveButton(canvasElement))
        const alert = await screen.findByRole('alertdialog')
        await expect(args.onSubmit).not.toHaveBeenCalled()
        await userEvent.click(within(alert).getByRole('button', { name: /continue|confirm|proceed|yes/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
    },
}
