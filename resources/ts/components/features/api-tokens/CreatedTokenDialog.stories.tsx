import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { CreatedTokenDialog } from './CreatedTokenDialog'

const secret = 'sv_ab12_0123456789abcdef0123456789abcdef'

const meta = {
    title: 'features/api-tokens/CreatedTokenDialog',
    component: CreatedTokenDialog,
    parameters: { a11y: overlayA11y },
    args: { token: secret, onClose: fn() },
} satisfies Meta<typeof CreatedTokenDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Shown: Story = {
    play: async () => {
        await expect(await screen.findByText(secret)).toBeVisible()
    },
}
export const Done: Story = {
    play: async ({ args }) => {
        await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /done/i }))
        await expect(args.onClose).toHaveBeenCalled()
    },
}

export const Copied: Story = {
    beforeEach: () => {
        const writeText = fn().mockResolvedValue(undefined)
        Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    },
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /copy/i }))
        await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith(secret))
    },
}

export const CopyRefused: Story = {
    beforeEach: () => {
        const writeText = fn().mockRejectedValue(new Error('denied'))
        Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    },
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /copy/i }))
        await expect(await screen.findByText(/could not copy/i)).toBeVisible()
    },
}
