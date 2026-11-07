import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, tags } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { RecurringForm } from './RecurringForm'

const meta = {
    title: 'features/recurring/RecurringForm',
    component: RecurringForm,
    parameters: { a11y: overlayA11y },
    args: { onSubmit: fn(), formId: 'recurring-form' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/accounts', accounts),
            reply('get', '/categories', categories),
            reply('get', '/currencies', currencies),
            reply('get', '/tags', tags),
        )
    },
} satisfies Meta<typeof RecurringForm>

export default meta
type Story = StoryObj<typeof meta>

const filled = {
    type: 'expense' as const,
    description: 'Rent',
    payments: [{ account_id: 1, amount: 1100 }],
    default_category_id: 1,
    frequency: 'monthly' as const,
    day_of_month: 5,
    start_date: '2026-01-01',
}

// Cash (USD) to the Euro card: the amount received is left open, so each occurrence works it out at its rates.
const transfer = {
    ...filled,
    type: 'transfer' as const,
    description: 'To the card',
    default_category_id: null,
    payments: [{ account_id: 1, amount: 100 }],
    to_account_id: 2,
}

const saveButton = (canvasElement: HTMLElement) => within(canvasElement).getByRole('button', { name: /save/i })

export const Empty: Story = {}
export const Filled: Story = { args: { defaultValues: filled } }
export const Income: Story = { args: { defaultValues: { ...filled, type: 'income', default_category_id: 2 } } }
export const Transfer: Story = { args: { defaultValues: { ...transfer, to_amount: 80 } } }
export const TransferLeftOpen: Story = {
    args: { defaultValues: { ...transfer, to_amount: null } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByPlaceholderText(/at each run/i)).toBeVisible()
    },
}
export const Weekly: Story = { args: { defaultValues: { ...filled, frequency: 'weekly', day_of_week: 3 } } }
export const Estimated: Story = { args: { defaultValues: { ...filled, estimated: true } } }
export const Paused: Story = { args: { defaultValues: { ...filled, is_active: false } } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: filled },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('textbox', { name: /description/i })).toBeDisabled())
    },
}

// A rule that stopped says why, above the form.
export const StoppedWithAReason: Story = {
    args: { defaultValues: { ...filled, is_active: false }, error: 'unknown account' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/unknown account/i)).toBeVisible()
    },
}

// What a rule makes is not dated: each occurrence has the day it falls on.
export const HasNoDateOfItsOwn: Story = {
    args: { defaultValues: filled },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await canvas.findByRole('textbox', { name: /description/i })
        await expect(canvas.queryByLabelText(/^date$/i)).toBeNull()
        await expect(canvas.getByLabelText(/start date/i)).toBeVisible()
    },
}

// There is no balance to preview: nothing moves until an occurrence is confirmed.
export const ShowsNoBalanceAfter: Story = {
    args: { defaultValues: filled },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await canvas.findByRole('textbox', { name: /description/i })
        await expect(canvas.queryByText(/balance after/i)).toBeNull()
    },
}

export const SeveralLines: Story = {
    args: {
        defaultValues: {
            ...filled,
            description: 'Market',
            payments: [{ account_id: 1, amount: 100 }],
            default_category_id: null,
            lines: [
                { category_id: 1, currency_id: 1, amount: 60, note: '- milk' },
                { category_id: 3, currency_id: 1, amount: 40, note: null },
            ],
        },
    },
}

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() =>
            expect(within(canvasElement).getAllByText(/select|greater|positive|required/i).length).toBeGreaterThan(0),
        )
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const RejectsAnEndBeforeTheStart: Story = {
    args: { defaultValues: { ...filled, end_date: '2025-01-01' } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await expect(await within(canvasElement).findByText(/end date must not be before/i)).toBeVisible()
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

// The rule is the template (signed payments and lines, no date) and the schedule.
export const Submits: Story = {
    args: { defaultValues: filled },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toEqual({
            template: {
                type: 'expense',
                description: 'Rent',
                payments: [{ account_id: 1, amount: -1100 }],
                lines: [{ category_id: 1, currency_id: 1, amount: 1100 }],
                tag_ids: [],
            },
            schedule: { frequency: 'monthly', interval: 1, day_of_month: 5, start: '2026-01-01', end: null },
            is_active: true,
        })
    },
}

export const SubmitsAnEstimate: Story = {
    args: { defaultValues: { ...filled, estimated: true } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            template: { payments: [{ account_id: 1, amount: -1100, estimated: true }] },
        })
    },
}

// An amount received left open is sent as no amount: the occurrence works it out.
export const SubmitsATransferWithTheAmountReceivedOpen: Story = {
    args: { defaultValues: { ...transfer, to_amount: null } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(saveButton(canvasElement))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            template: {
                type: 'transfer',
                payments: [
                    { account_id: 1, amount: -100 },
                    { account_id: 2, amount: null, estimated: true },
                ],
                lines: [],
            },
        })
    },
}

// Switching to a transfer drops the category and asks for a destination.
export const SwitchedToTransfer: Story = {
    args: { defaultValues: filled },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /transfer/i }))
        await waitFor(() => expect(canvas.queryByRole('combobox', { name: /category/i })).toBeNull())
    },
}

// A rule cannot send money to another space.
export const OffersNoTransferToAnotherSpace: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await canvas.findByRole('textbox', { name: /description/i })
        await expect(canvas.queryByRole('button', { name: /to space|other space/i })).toBeNull()
    },
}
