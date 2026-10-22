import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { Button } from './button'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from './dialog'

const meta = {
    title: 'ui/Dialog',
    component: Dialog,
    render: (args) => (
        <Dialog {...args}>
            <DialogTrigger asChild><Button>Open</Button></DialogTrigger>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Delete account</DialogTitle>
                    <DialogDescription>This cannot be undone.</DialogDescription>
                </DialogHeader>
                <DialogFooter><Button variant="destructive">Delete</Button></DialogFooter>
            </DialogContent>
        </Dialog>
    ),
} satisfies Meta<typeof Dialog>

export default meta
type Story = StoryObj<typeof meta>

export const Closed: Story = {}

export const Opened: Story = { args: { defaultOpen: true } }

export const OpenedByClick: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Open' }))
        await expect(await screen.findByRole('dialog', { name: 'Delete account' })).toBeVisible()
    },
}

export const ClosedByEscape: Story = {
    args: { defaultOpen: true },
    play: async () => {
        await userEvent.keyboard('{Escape}')
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}
