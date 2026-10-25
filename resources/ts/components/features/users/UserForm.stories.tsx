import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { UserForm } from './UserForm'

const meta = {
    title: 'features/users/UserForm',
    component: UserForm,
    parameters: { a11y: overlayA11y },
    args: { onSubmit: fn() },
} satisfies Meta<typeof UserForm>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}
export const Submitting: Story = { args: { isSubmitting: true } }

export const Edit: Story = {
    args: { isEdit: true, defaultValues: { id: 7, name: 'Sam', email: 'sam@example.com', role: 'guest' } },
    play: async ({ canvasElement }) => {
        const form = within(canvasElement)
        await expect(form.getByDisplayValue('Sam')).toBeVisible()
        await expect(form.getByLabelText(/new password/i)).toBeVisible()
        await expect(form.queryByRole('checkbox')).toBeNull()
    },
}

// The signed-in user (id 1) cannot take their own admin role away.
export const EditingYourself: Story = {
    args: { isEdit: true, defaultValues: { id: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin' } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('combobox', { name: /role/i })).toBeDisabled()
    },
}

export const WithoutSubmitButton: Story = {
    args: { hideSubmit: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button', { name: /save|create/i })).toBeNull()
    },
}

export const PasswordFieldAppearsOnRequest: Story = {
    play: async ({ canvasElement }) => {
        const form = within(canvasElement)
        await expect(form.queryByLabelText(/^password/i)).toBeNull()
        await userEvent.click(form.getByRole('checkbox', { name: /password/i }))
        await expect(await form.findByLabelText(/^password/i)).toBeVisible()
    },
}

// ---- how the set-password link reaches the user

export const MailAvailable: Story = {
    args: { mail: { canEmail: true, mustEmail: false } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('checkbox', { name: /email the link/i })).toBeChecked()
    },
}

export const MailRequired: Story = {
    args: { mail: { canEmail: true, mustEmail: true } },
    play: async ({ canvasElement }) => {
        const box = within(canvasElement).getByRole('checkbox', { name: /email the link/i })
        await expect(box).toBeChecked()
        await expect(box).toBeDisabled()
    },
}

export const MailRequiredButNotConfigured: Story = {
    args: { mail: { canEmail: false, mustEmail: true } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/mail is not set up/i)).toBeVisible()
    },
}

// ---- validation and submit

export const RejectsEmptyFields: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /save|create/i }))
        await expect(await within(canvasElement).findByText(/name is required/i)).toBeVisible()
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const Submitted: Story = {
    play: async ({ canvasElement, args }) => {
        const form = within(canvasElement)
        await userEvent.type(form.getByLabelText(/name/i), 'Sam')
        await userEvent.type(form.getByLabelText(/email/i), 'sam@example.com')
        await userEvent.click(form.getByRole('button', { name: /save|create/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalled())
        await expect(vi_first(args.onSubmit)).toMatchObject({ name: 'Sam', email: 'sam@example.com', role: 'user' })
    },
}

const vi_first = (spy: unknown) => (spy as { mock: { calls: unknown[][] } }).mock.calls[0][0]
