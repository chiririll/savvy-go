import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, tags } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionForm } from './TransactionForm'

// Transfers: two accounts, what leaves one and what arrives on the other.
const meta = {
    title: 'features/transactions/form/TransactionForm/Transfers',
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

const lineGroup = (canvasElement: HTMLElement) => within(canvasElement).findByRole('group', { name: /lines|строки/i })
const saveButton = (canvasElement: HTMLElement) => within(canvasElement).getByRole('button', { name: /save/i })

const transfer = {
    type: 'transfer' as const,
    payments: [{ account_id: 1, amount: 100 }],
    to_account_id: 2,
    to_amount: 80,
    date: PAST,
    description: 'To the card',
    default_category_id: null,
}

const savings = { ...accounts[0], id: 5, name: 'Savings', currentBalance: 50 }
const withSavings = ({ msw }: { msw: { use: (...handlers: never[]) => void } }) =>
    msw.use(reply('get', '/accounts', [...accounts, savings]) as never)

const receiveAmount = /receive amount|сумма получения/i
const transferRate = /transfer rate|курс перевода/i
const feeSection = (canvasElement: HTMLElement) =>
    within(canvasElement).findByRole('region', { name: /commission|комиссия/i })

// An opener that asks for a transfer gets the transfer form: two accounts and no category.
export const TransferRequestedOpensATransfer: Story = {
    args: { defaultValues: { type: 'transfer', payments: [{ account_id: 1, amount: 100 }] } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('button', { name: /transfer/i })).toHaveAttribute('aria-pressed', 'true')
        await expect(await canvas.findByText(/destination account|счёт назначения/i)).toBeVisible()
        await expect(canvas.queryByText(/^category$|^категория$/i)).toBeNull()
    },
}

// Between two currencies the destination has an amount of its own, and the rate between them is shown.
export const TransferBetweenTwoCurrencies: Story = {
    args: { defaultValues: transfer },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByLabelText(receiveAmount)).toHaveValue(80)
        await waitFor(() => expect(canvas.getByLabelText(transferRate)).toHaveValue(0.8))
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            type: 'transfer',
            payments: [
                { account_id: 1, amount: -100 },
                { account_id: 2, amount: 80 },
            ],
            lines: [],
        })
    },
}

// Typing a rate works out the amount the destination gets.
export const RateWorksOutTheReceivedAmount: Story = {
    args: { defaultValues: { ...transfer, to_amount: null } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const rate = await canvas.findByLabelText(transferRate)
        await userEvent.clear(rate)
        await userEvent.type(rate, '0.9')
        await userEvent.tab()
        await waitFor(() => expect(canvas.getByLabelText(receiveAmount)).toHaveValue(90))
    },
}

// In one currency the amount is the same on both, so there is one amount and no rate.
export const TransferInOneCurrencyHasOneAmount: Story = {
    beforeEach: withSavings,
    args: { defaultValues: { ...transfer, to_account_id: 5, to_amount: null } },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await waitFor(() => expect(canvas.getAllByRole('combobox')[1]).toHaveTextContent(/savings/i))
        await expect(canvas.queryByLabelText(receiveAmount)).toBeNull()
        await expect(canvas.queryByLabelText(transferRate)).toBeNull()
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            payments: [
                { account_id: 1, amount: -100 },
                { account_id: 5, amount: 100 },
            ],
        })
    },
}

// Both accounts show what they hold after the transfer, a fee included.
export const TransferShowsBothBalances: Story = {
    args: { defaultValues: transfer },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/20\.50/)).toBeVisible()
        await expect(canvas.getByText(/1,?060\.10/)).toBeVisible()
    },
}

// A commission is paid from the source on top of the amount and saved as a line (F7b pays a card with it).
export const TransferWithACommission: Story = {
    beforeEach: withSavings,
    args: { defaultValues: { ...transfer, to_account_id: 5, to_amount: null } },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /add commission|добавить комиссию/i }))
        const section = await feeSection(canvasElement)
        await userEvent.type(within(section).getByRole('spinbutton'), '1.5')
        await waitFor(() => expect(canvas.getAllByRole('combobox')[1]).toHaveTextContent(/savings/i))
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            payments: [
                { account_id: 1, amount: -101.5 },
                { account_id: 5, amount: 100 },
            ],
            lines: [{ category_id: null, currency_id: 1, amount: 1.5 }],
        })
    },
}

// A commission asked for needs an amount.
export const CommissionNeedsAnAmount: Story = {
    args: { defaultValues: { ...transfer, fee: { amount: '' as unknown as number, category_id: null } } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /save/i }))
        const section = await feeSection(canvasElement)
        await expect(await within(section).findByText(/amount|сумм/i, { selector: '[id$="-form-item-message"]' })).toBeVisible()
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const CommissionCanBeRemoved: Story = {
    args: { defaultValues: { ...transfer, fee: { amount: 2, category_id: null } } },
    play: async ({ canvasElement }) => {
        const section = await feeSection(canvasElement)
        await userEvent.click(within(section).getByRole('button', { name: /remove the commission|убрать комиссию/i }))
        await expect(await within(canvasElement).findByRole('button', { name: /add commission|добавить комиссию/i })).toBeVisible()
    },
}

// The swap puts each account where the other was, with its amount, and turns the rate over.
export const SwapAccounts: Story = {
    args: { defaultValues: transfer },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await canvas.findByLabelText(receiveAmount)
        await userEvent.click(canvas.getByRole('button', { name: /swap accounts|поменять счета/i }))
        await waitFor(() => expect(canvas.getAllByRole('combobox')[0]).toHaveTextContent(/euro card/i))
        await expect(canvas.getAllByRole('combobox')[1]).toHaveTextContent(/cash/i)
        await expect(canvas.getByLabelText(receiveAmount)).toHaveValue(100)
        await expect(canvas.getByLabelText(transferRate)).toHaveValue(1.25)
    },
}

// The source cannot be picked as the destination.
export const DestinationIsNotTheSource: Story = {
    args: { defaultValues: transfer },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click((await canvas.findAllByRole('combobox'))[1])
        const options = await screen.findAllByRole('option')
        await expect(options.some((option) => /^cash/i.test(option.textContent ?? ''))).toBe(false)
        await expect(options.some((option) => /yen wallet/i.test(option.textContent ?? ''))).toBe(true)
    },
}

// A transfer with no destination is not saved.
export const TransferNeedsADestination: Story = {
    args: { defaultValues: { ...transfer, to_account_id: null, to_amount: null } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /save/i }))
        await expect(await within(canvasElement).findByText(/destination account/i, { selector: '[id$="-form-item-message"]' })).toBeVisible()
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

// Going from an expense with lines to a transfer drops what a transfer cannot have: lines, change, the category.
export const SwitchingToATransferDropsTheLines: Story = {
    args: { defaultValues: { ...split, change: [{ account_id: 2, amount: 1 }] } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await lineGroup(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /transfer/i }))
        await waitFor(() => expect(canvas.queryByRole('group', { name: /lines|строки/i })).toBeNull())
        await expect(canvas.queryByRole('region', { name: /^change$|^сдача$/i })).toBeNull()
        await expect(await canvas.findByText(/destination account|счёт назначения/i)).toBeVisible()
    },
}

// A saved transfer opens in the form with its amounts; the fee is taken off what was sent.
export const EditingATransferWithACommission: Story = {
    args: {
        isEdit: true,
        originalAffectsBalance: true,
        defaultValues: { ...transfer, payments: [{ account_id: 1, amount: 100 }], fee: { amount: 2, category_id: null } },
    },
    play: async ({ canvasElement }) => {
        const section = await feeSection(canvasElement)
        await expect(within(section).getByRole('spinbutton')).toHaveValue(2)
        await expect(within(canvasElement).getAllByRole('spinbutton')[0]).toHaveValue(100)
    },
}

const toSpace = {
    ...transfer,
    type: 'transfer_out' as const,
    to_space_id: 2,
    to_account_id: 3,
    to_amount: 15000,
}
const linkedSpace = ({ msw }: { msw: { use: (...handlers: never[]) => void } }) =>
    msw.use(
        reply('get', '/spaces', [space('admin'), { ...space('editor'), id: 2, name: 'Trip' }]) as never,
        reply('get', '/links', [{ id: 2, name: 'Trip' }]) as never,
    )

// A transfer to another space goes to the transfers API, with both amounts and a date, and has no commission or tags.
export const TransferToAnotherSpace: Story = {
    beforeEach: linkedSpace,
    args: { defaultValues: toSpace, onSubmitSpaceTransfer: fn() },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText(/destination space|пространство назначения/i)).toBeVisible()
        await expect(canvas.queryByRole('region', { name: /commission|комиссия/i })).toBeNull()
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmitSpaceTransfer).toHaveBeenCalledOnce())
        await expect(args.onSubmit).not.toHaveBeenCalled()
        await expect((args.onSubmitSpaceTransfer as ReturnType<typeof fn>).mock.calls[0][0]).toEqual({
            to_space_id: 2,
            from_account_id: 1,
            to_account_id: 3,
            from_amount: '100',
            to_amount: '15000',
            date: PAST,
            description: 'To the card',
        })
    },
}

// The transfers API stores a transfer under a date, so one without a date is not sent.
export const TransferToAnotherSpaceNeedsADate: Story = {
    beforeEach: linkedSpace,
    args: { defaultValues: { ...toSpace, date: '' }, onSubmitSpaceTransfer: fn() },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /save/i }))
        await expect(await within(canvasElement).findByText(/date/i, { selector: '[id$="-form-item-message"]' })).toBeVisible()
        await expect(args.onSubmitSpaceTransfer).not.toHaveBeenCalled()
    },
}
