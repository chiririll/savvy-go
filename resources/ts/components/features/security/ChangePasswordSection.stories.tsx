import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import { fail, spy } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'

import { ChangePasswordSection } from './ChangePasswordSection'

const meta = {
    title: 'features/security/ChangePasswordSection',
    component: ChangePasswordSection,
} satisfies Meta<typeof ChangePasswordSection>

export default meta
type Story = StoryObj<typeof meta>

const fill = async (canvas: ReturnType<typeof within>, current: string, next: string, again: string) => {
    await userEvent.type(canvas.getByLabelText(/current password/i), current)
    await userEvent.type(canvas.getByLabelText(/^new password/i), next)
    await userEvent.type(canvas.getByLabelText(/confirm/i), again)
}

const sent: { bodies: unknown[] } = { bodies: [] }

export const Form: Story = {}
export const SingleSignOnOnly: Story = {
    beforeEach: () => {
        const user = useAuthStore.getState().user
        if (user) useAuthStore.setState({ user: { ...user, isSsoOnly: true } })
    },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByLabelText(/current password/i)).toBeNull()
    },
}

export const RejectsEmptyFields: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /change|update|save/i }))
        await expect(await within(canvasElement).findByText(/at least 8/i)).toBeVisible()
    },
}

export const PasswordsMustMatch: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await fill(canvas, 'old-secret', 'a-long-enough-one', 'a-different-one!')
        await userEvent.click(canvas.getByRole('button', { name: /change|update|save/i }))
        await expect(await canvas.findByText(/don't match|do not match/i)).toBeVisible()
    },
}

export const Changed: Story = {
    beforeEach: ({ msw }) => {
        const change = spy('put', '/auth/password', { message: 'ok' }, 200, false)
        sent.bodies = change.bodies
        msw.use(change.handler)
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await fill(canvas, 'old-secret', 'a-long-enough-one', 'a-long-enough-one')
        await userEvent.click(canvas.getByRole('button', { name: /change|update|save/i }))
        await waitFor(() => expect(sent.bodies).toHaveLength(1))
        await expect(sent.bodies[0]).toEqual({
            current_password: 'old-secret',
            password: 'a-long-enough-one',
            password_confirmation: 'a-long-enough-one',
        })
        // The form is emptied after a change.
        await waitFor(() => expect(canvas.getByLabelText(/current password/i)).toHaveValue(''))
    },
}

export const WrongCurrentPassword: Story = {
    beforeEach: ({ msw }) => msw.use(fail('put', '/auth/password', 'The current password is wrong', 422)),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await fill(canvas, 'nope', 'a-long-enough-one', 'a-long-enough-one')
        await userEvent.click(canvas.getByRole('button', { name: /change|update|save/i }))
        await new Promise((resolve) => setTimeout(resolve, 300))
        // What was typed stays, so it can be corrected.
        await expect(canvas.getByLabelText(/^new password/i)).toHaveValue('a-long-enough-one')
    },
}
