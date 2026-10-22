import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'

import { TagForm } from './TagForm'

const meta = {
    title: 'features/tags/TagForm',
    component: TagForm,
    args: { onSubmit: fn(), formId: 'tag-form' },
} satisfies Meta<typeof TagForm>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {}
export const Filled: Story = { args: { defaultValues: { name: 'holiday' } } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const CustomSubmitLabel: Story = { args: { submitLabel: 'Add tag' } }
export const WithoutOwnSubmit: Story = { args: { hideSubmit: true } }
export const InRussian: Story = { globals: { locale: 'ru' } }

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { defaultValues: { name: 'holiday' } },
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

export const Submits: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.type(within(canvasElement).getByRole('textbox'), 'trip')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledWith({ name: 'trip' }))
    },
}

// A built-in tag keeps its stored name while the text is unchanged, so it stays translated.
export const BuiltInNameStaysStored: Story = {
    args: { defaultValues: { name: '#VACATION' } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledWith({ name: '#VACATION' }))
    },
}
