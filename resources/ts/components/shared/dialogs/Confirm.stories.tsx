import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { Button } from '@/components/ui/button'

import { Confirm } from './Confirm'

const meta = {
    title: 'shared/dialogs/Confirm',
    component: Confirm,
    args: {
        open: true,
        onOpenChange: fn(),
        onConfirm: fn(),
        title: 'Remove the key?',
        description: 'Spaces that trust it will stop accepting transfers from this one.',
        action: 'Remove',
    },
} satisfies Meta<typeof Confirm>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Destructive: Story = { args: { destructive: true } }

export const WithRichDescription: Story = {
    args: {
        description: (
            <>
                <p>The backup of 3 Jan will replace everything.</p>
                <p>It was made by an older version.</p>
            </>
        ),
    },
}

export const WithoutDescription: Story = { args: { description: undefined } }

export const CustomCancelLabel: Story = {
    args: { cancel: 'Keep it' },
    play: async () => {
        await expect(await screen.findByRole('button', { name: 'Keep it' })).toBeVisible()
    },
}

export const Closed: Story = {
    args: { open: false },
    play: async () => {
        await expect(screen.queryByRole('alertdialog')).toBeNull()
    },
}

export const ConfirmCallsBack: Story = {
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: 'Remove' }))
        await expect(args.onConfirm).toHaveBeenCalledOnce()
    },
}

export const CancelDoesNotConfirm: Story = {
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: /cancel/i }))
        await expect(args.onConfirm).not.toHaveBeenCalled()
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}

// A click outside is not an answer.
export const OutsideClickKeepsItOpen: Story = {
    play: async ({ args }) => {
        await screen.findByRole('alertdialog')
        const overlay = document.querySelector('[data-slot="alert-dialog-overlay"]')
        await userEvent.click(overlay as Element)
        await expect(args.onOpenChange).not.toHaveBeenCalled()
    },
}

// A request in flight: the dialog stays, the buttons are locked.
export const Pending: Story = {
    args: { pending: true },
    play: async () => {
        const dialog = await screen.findByRole('alertdialog')
        await expect(within(dialog).getByRole('button', { name: 'Remove' })).toBeDisabled()
        await expect(within(dialog).getByRole('button', { name: /cancel/i })).toBeDisabled()
    },
}

export const WithTrigger: Story = {
    args: { open: undefined, onOpenChange: undefined, trigger: <Button>Remove key</Button> },
    play: async ({ args }) => {
        await expect(screen.queryByRole('alertdialog')).toBeNull()
        await userEvent.click(screen.getByRole('button', { name: 'Remove key' }))
        await userEvent.click(await screen.findByRole('button', { name: 'Remove' }))
        await expect(args.onConfirm).toHaveBeenCalledOnce()
        // Without `pending` the dialog closes at once.
        await waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull())
    },
}

export const ControlledStaysOpenWhilePending: Story = {
    render: function Render(args) {
        const [pending, setPending] = useState(false)
        return (
            <Confirm
                {...args}
                pending={pending}
                onConfirm={() => {
                    args.onConfirm()
                    setPending(true)
                }}
            />
        )
    },
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: 'Remove' }))
        await expect(args.onConfirm).toHaveBeenCalledOnce()
        await expect(screen.getByRole('alertdialog')).toBeVisible()
        await expect(screen.getByRole('button', { name: 'Remove' })).toBeDisabled()
    },
}
