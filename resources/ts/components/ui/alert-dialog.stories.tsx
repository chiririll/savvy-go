import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent } from 'storybook/test'

import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from './alert-dialog'

const meta = {
    title: 'ui/AlertDialog',
    component: AlertDialog,
    args: { defaultOpen: true, onOpenChange: fn() },
    render: ({ onOpenChange, ...args }) => (
        <AlertDialog {...args} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Delete this budget?</AlertDialogTitle>
                    <AlertDialogDescription>Its history is deleted with it.</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>Keep</AlertDialogCancel>
                    <AlertDialogAction>Delete</AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    ),
} satisfies Meta<typeof AlertDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Opened: Story = {}

export const Cancelled: Story = {
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: 'Keep' }))
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}

export const Confirmed: Story = {
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: 'Delete' }))
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}
