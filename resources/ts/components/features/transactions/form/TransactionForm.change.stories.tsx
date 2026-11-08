import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, tags } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionForm } from './TransactionForm'

// Change: what is given back on top of the payments.
const meta = {
    title: 'features/transactions/form/TransactionForm/Change',
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

const expense = {
    type: 'expense' as const,
    payments: [{ account_id: 1, amount: 20 }],
    default_category_id: 1,
    date: PAST,
    description: 'Weekly shop',
}

const split = {
    ...expense,
    lines: [
        { category_id: 3, currency_id: null, amount: 12, note: null },
        { category_id: 1, currency_id: null, amount: 8, note: null },
    ],
}

const twoAccounts = {
    ...expense,
    payments: [
        { account_id: 1, amount: 10 },
        { account_id: 2, amount: 8 },
    ],
}

const lineGroup = (canvasElement: HTMLElement) => within(canvasElement).findByRole('group', { name: /lines|строки/i })
const saveButton = (canvasElement: HTMLElement) => within(canvasElement).getByRole('button', { name: /save/i })

/** The accounts come from the mocked API: the form is ready when the first balance is drawn. */
const ready = (canvasElement: HTMLElement) =>
    waitFor(() => expect(within(canvasElement).getAllByTestId('balance-after')[0]).not.toBeEmptyDOMElement())

const addChange = /add change|добавить сдачу/i
const removeChange = /remove the change|убрать сдачу/i
const changeSection = (canvasElement: HTMLElement) =>
    within(canvasElement).findByRole('region', { name: /^change$|^сдача$/i })

// The Change section is apart from the payments, with its own (+).
export const ChangeSectionOfAnExpense: Story = {
    args: { defaultValues: expense },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const section = await changeSection(canvasElement)
        await expect(within(section).queryAllByRole('textbox', { name: /^amount$/i })).toHaveLength(0)
        await userEvent.click(within(section).getByRole('button', { name: addChange }))
        await waitFor(() => expect(within(section).getAllByRole('textbox', { name: /^amount$/i })).toHaveLength(1))
        await expect(within(section).getByRole('combobox')).toHaveTextContent(/euro card/i)
    },
}

// A saved change comes back in the section, with the balance of its account.
export const ChangeOfASavedExpense: Story = {
    args: { defaultValues: { ...expense, change: [{ account_id: 2, amount: 5 }] } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const section = await changeSection(canvasElement)
        await expect(within(section).getByRole('textbox', { name: /^amount$/i })).toHaveValue('5')
        await expect(await within(section).findByText(/balance after|баланс после/i)).toBeVisible()
        await userEvent.click(within(section).getByRole('button', { name: removeChange }))
        await waitFor(() => expect(within(section).queryAllByRole('textbox', { name: /^amount$/i })).toHaveLength(0))
    },
}

// Change worth as much as the payments is refused by the API, so the form says so and does not save.
export const ChangeTooLarge: Story = {
    args: { defaultValues: { ...expense, change: [{ account_id: 2, amount: 20 }] } },
    play: async ({ canvasElement, args }) => {
        await ready(canvasElement)
        const section = await changeSection(canvasElement)
        await expect((await within(section).findAllByRole('alert'))[0]).toBeVisible()
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(within(section).getAllByRole('alert')[0]).toBeVisible())
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

// An income has no change to give, unless a saved one has some.
export const IncomeHasNoChangeSection: Story = {
    args: { defaultValues: { ...expense, type: 'income', default_category_id: 2 } },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        await within(canvasElement).findByRole('textbox', { name: /description/i })
        await expect(within(canvasElement).queryByRole('region', { name: /^change$|^сдача$/i })).toBeNull()
    },
}

// Payments and change are saved with their signs: the change is the opposite of the payments.
export const SubmitsPaymentsAndChange: Story = {
    args: {
        defaultValues: {
            ...twoAccounts,
            change: [{ account_id: 3, amount: 100 }],
            lines: [{ category_id: 1, currency_id: null, amount: 10, note: '' }],
        },
    },
    play: async ({ canvasElement, args }) => {
        await ready(canvasElement)
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            payments: [
                { account_id: 1, amount: -10 },
                { account_id: 2, amount: -8 },
                { account_id: 3, amount: 100 },
            ],
        })
    },
}

// A line can be in the currency of a change account too.
export const LineCurrencyIncludesTheChange: Story = {
    args: {
        defaultValues: {
            ...split,
            change: [{ account_id: 2, amount: 2 }],
            lines: [{ category_id: 3, currency_id: null, amount: 5, note: null }, split.lines[1]],
        },
    },
    play: async ({ canvasElement }) => {
        await ready(canvasElement)
        const group = await lineGroup(canvasElement)
        const currency = within(group).getAllByRole('combobox', { name: /currency of the line|валюта строки/i })[0]
        await userEvent.click(currency)
        const options = await screen.findAllByRole('option')
        await expect(options.map((o) => o.textContent)).toEqual(['USD', 'EUR'])
    },
}
