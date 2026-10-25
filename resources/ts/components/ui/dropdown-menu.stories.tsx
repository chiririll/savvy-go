import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { Button } from './button'
import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from './dropdown-menu'
import { overlayA11y } from '@/test/a11y'

const meta = {
    title: 'ui/DropdownMenu',
    component: DropdownMenu,
    parameters: { a11y: overlayA11y },
    args: { onOpenChange: fn() },
    render: (args) => (
        <DropdownMenu {...args}>
            <DropdownMenuTrigger asChild><Button variant="outline">Actions</Button></DropdownMenuTrigger>
            <DropdownMenuContent>
                <DropdownMenuLabel>Account</DropdownMenuLabel>
                <DropdownMenuItem>Edit</DropdownMenuItem>
                <DropdownMenuItem disabled>Archive</DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuCheckboxItem checked>Pinned</DropdownMenuCheckboxItem>
                <DropdownMenuItem variant="destructive">Delete</DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    ),
} satisfies Meta<typeof DropdownMenu>

export default meta
type Story = StoryObj<typeof meta>

export const Opened: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Actions' }))
        await expect(await screen.findByRole('menuitem', { name: 'Edit' })).toBeVisible()
        await expect(screen.getByRole('menuitem', { name: 'Archive' })).toHaveAttribute('aria-disabled', 'true')
        await expect(screen.getByRole('menuitemcheckbox', { name: 'Pinned' })).toBeChecked()
    },
}

export const ClosedByEscape: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Actions' }))
        await screen.findByRole('menu')
        await userEvent.keyboard('{Escape}')
        await expect(screen.queryByRole('menu')).toBeNull()
    },
}
