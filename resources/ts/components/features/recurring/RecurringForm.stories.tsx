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
        msw.use(reply('get', '/accounts', accounts), reply('get', '/categories', categories), reply('get', '/tags', tags))
    },
} satisfies Meta<typeof RecurringForm>

export default meta
type Story = StoryObj<typeof meta>

const filled = {
    type: 'expense' as const,
    account_id: 1,
    category_id: 1,
    amount: 1100,
    description: 'Rent',
    frequency: 'monthly' as const,
    start_date: '2026-01-01',
}

export const Empty: Story = {}
export const Filled: Story = { args: { defaultValues: filled } }
export const Income: Story = { args: { defaultValues: { ...filled, type: 'income', category_id: 2 } } }
export const Transfer: Story = {
    args: { defaultValues: { ...filled, type: 'transfer', category_id: null, to_account_id: 2, to_amount: 1000 } },
}
export const Weekly: Story = { args: { defaultValues: { ...filled, frequency: 'weekly', day_of_week: 3 } } }
export const Estimated: Story = { args: { defaultValues: { ...filled, is_estimated: true } } }
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

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await waitFor(() => expect(within(canvasElement).getAllByText(/select|greater|required/i).length).toBeGreaterThan(0))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const Submits: Story = {
    args: { defaultValues: filled },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            type: 'expense',
            account_id: 1,
            category_id: 1,
            amount: 1100,
            frequency: 'monthly',
        })
    },
}

// Switching to a transfer drops the category and asks for a destination.
export const SwitchedToTransfer: Story = {
    args: { defaultValues: filled },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /transfer/i }))
        await waitFor(() => expect(canvas.queryByRole('combobox', { name: /category/i })).toBeNull())
    },
}
