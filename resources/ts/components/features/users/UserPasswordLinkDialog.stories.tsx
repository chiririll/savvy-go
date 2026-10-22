import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { UserPasswordLinkDialog } from './UserPasswordLinkDialog'

const url = 'https://savvy.example.com/set-password?token=abc123'

const meta = {
    title: 'features/users/UserPasswordLinkDialog',
    component: UserPasswordLinkDialog,
    args: { open: true, onOpenChange: fn(), url, expiresAt: '2030-01-10T12:00:00Z' },
} satisfies Meta<typeof UserPasswordLinkDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Link: Story = {
    play: async () => {
        await expect(await screen.findByDisplayValue(url)).toHaveAttribute('readonly')
    },
}
export const WithoutExpiry: Story = { args: { expiresAt: null } }
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Closed: Story = { args: { open: false } }

export const Done: Story = {
    play: async ({ args }) => {
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /done/i }))
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}

export const Copied: Story = {
    beforeEach: () => {
        const writeText = fn().mockResolvedValue(undefined)
        Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        const [copy] = within(dialog).getAllByRole('button').filter((b) => !/done|close/i.test(b.textContent ?? ''))
        await userEvent.click(copy)
        await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith(url))
    },
}
