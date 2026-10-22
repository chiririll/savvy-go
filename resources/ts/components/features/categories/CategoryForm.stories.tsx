import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'

import { CategoryForm } from './CategoryForm'

const meta = {
    title: 'features/categories/CategoryForm',
    component: CategoryForm,
    args: { onSubmit: fn(), formId: 'category-form' },
} satisfies Meta<typeof CategoryForm>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {}
export const Income: Story = { args: { defaultValues: { name: 'Salary', type: 'income', icon: '💼', color: '#2563eb' } } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
export const InRussian: Story = { globals: { locale: 'ru' } }

// The default category keeps its type.
export const DefaultCategory: Story = {
    args: { isDefault: true, defaultValues: { name: 'Groceries', type: 'expense' } },
    play: async ({ canvasElement }) => {
        const type = within(canvasElement).getAllByRole('button', { name: /expense|income/i })
        for (const button of type) await expect(button).toBeDisabled()
    },
}

// A built-in name is shown translated, with a hint that it stays so while unchanged.
export const BuiltInName: Story = { args: { defaultValues: { name: '#GROCERIES' } } }

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: { name: 'Salary' } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('textbox')).toBeDisabled())
    },
}

export const RejectsEmptyName: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await expect(await within(canvasElement).findByText(/required|characters/i)).toBeVisible()
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const PreviewFollowsTheForm: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByRole('textbox'), 'Cinema')
        await expect(canvas.getByText('Cinema')).toBeVisible()
    },
}

export const SubmitsChosenColor: Story = {
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByRole('textbox'), 'Cinema')
        const swatches = canvas.getAllByRole('button', { name: /^#/ })
        await userEvent.click(swatches[0])
        await expect(swatches[0]).toHaveAttribute('aria-pressed', 'true')
        await userEvent.click(canvas.getByRole('button', { name: /save/i }))
        const color = swatches[0].getAttribute('aria-label')
        await waitFor(() =>
            expect(args.onSubmit).toHaveBeenCalledWith(expect.objectContaining({ name: 'Cinema', color })),
        )
    },
}
