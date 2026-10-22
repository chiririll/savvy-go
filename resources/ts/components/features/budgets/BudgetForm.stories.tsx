import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { categories, tags } from '@/test/fixtures'
import { currencyApi, reply } from '@/test/api-mocks'

import { BudgetForm } from './BudgetForm'

const meta = {
    title: 'features/budgets/BudgetForm',
    component: BudgetForm,
    args: { onSubmit: fn(), formId: 'budget-form' },
    beforeEach: ({ msw }) => {
        msw.use(...currencyApi(), reply('get', '/categories', categories), reply('get', '/tags', tags))
    },
} satisfies Meta<typeof BudgetForm>

export default meta
type Story = StoryObj<typeof meta>

const filled = {
    name: 'Groceries',
    amount: 300,
    currency_id: 1,
    period: 'monthly' as const,
    category_ids: [1],
    tag_ids: [2],
}

export const Empty: Story = {}
export const Filled: Story = { args: { defaultValues: filled } }
export const AllExpenses: Story = { args: { defaultValues: { ...filled, is_global: true, category_ids: [] } } }
export const OneTime: Story = {
    args: { defaultValues: { ...filled, period: 'one_time', start_date: '2026-11-01', end_date: '2026-11-30' } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByLabelText(/start/i)).toBeVisible()
    },
}
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
export const InRussian: Story = { globals: { locale: 'ru' } }

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: filled },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('textbox', { name: /name/i })).toBeDisabled())
    },
}

export const CategoriesHiddenForAllExpenses: Story = {
    args: { defaultValues: filled },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await canvas.findByText('Transport')
        await userEvent.click(canvas.getByRole('checkbox', { name: /all expenses|global/i }))
        await waitFor(() => expect(canvas.queryByText('Transport')).toBeNull())
    },
}

export const RejectsEmptyForm: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await expect((await within(canvasElement).findAllByText(/required|greater|characters/i)).length).toBeGreaterThan(0)
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const OnlyPositiveLimit: Story = {
    args: { defaultValues: { ...filled, amount: 0 } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).not.toHaveBeenCalled())
    },
}

export const Submits: Story = {
    args: { defaultValues: filled },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            name: 'Groceries',
            amount: 300,
            period: 'monthly',
            category_ids: [1],
            tag_ids: [2],
        })
    },
}

export const PickedPeriod: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('combobox', { name: /period/i }))
        await userEvent.click(await screen.findByRole('option', { name: /one.?time/i }))
        await expect(await canvas.findByLabelText(/start/i)).toBeVisible()
    },
}
