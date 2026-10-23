import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { UserFormDialog } from './UserFormDialog'

const sam = {
    id: 7,
    name: 'Sam',
    email: 'sam@example.com',
    role: 'user' as const,
    isInactive: false,
    isSsoOnly: false,
    emailVerified: true,
    createdAt: '2025-01-01T00:00:00Z',
}

const meta = {
    title: 'features/users/UserFormDialog',
    component: UserFormDialog,
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
} satisfies Meta<typeof UserFormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}
export const Edit: Story = {
    args: { user: sam },
    play: async () => {
        await expect(await screen.findByDisplayValue('Sam')).toBeVisible()
    },
}
export const Submitting: Story = { args: { isSubmitting: true } }
export const WithMail: Story = { args: { mail: { canEmail: true, mustEmail: false } } }

export const SubmittedFromTheFooter: Story = {
    args: { user: sam },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await expect(args.onSubmit).toHaveBeenCalled()
    },
}
