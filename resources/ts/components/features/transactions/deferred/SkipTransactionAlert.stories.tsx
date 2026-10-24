import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { Button } from '@/components/ui/button'

import { SkipTransactionAlert } from './SkipTransactionAlert'

const meta = {
    title: 'features/transactions/deferred/SkipTransactionAlert',
    component: SkipTransactionAlert,
    args: { onConfirm: fn(), trigger: <Button>Skip</Button> },
} satisfies Meta<typeof SkipTransactionAlert>

export default meta
type Story = StoryObj<typeof meta>

export const Closed: Story = {
    play: async () => {
        await expect(screen.queryByRole('alertdialog')).toBeNull()
    },
}

export const Asking: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Skip' }))
        await expect(await screen.findByRole('alertdialog')).toBeVisible()
    },
}

export const Confirmed: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Skip' }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /^skip$/i }))
        await waitFor(() => expect(args.onConfirm).toHaveBeenCalledOnce())
    },
}

export const Cancelled: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Skip' }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /cancel/i }))
        await expect(args.onConfirm).not.toHaveBeenCalled()
    },
}
