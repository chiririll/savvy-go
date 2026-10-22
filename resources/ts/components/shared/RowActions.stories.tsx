import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { DropdownMenuItem } from '@/components/ui/dropdown-menu'

import { RowActions } from './RowActions'
import { overlayA11y } from '@/test/a11y'

const meta = {
    title: 'shared/RowActions',
    component: RowActions,
    parameters: { a11y: overlayA11y },
    args: { onEdit: fn(), onDelete: fn(), deleteTitle: 'Delete account?', deleteDescription: 'Its history goes too.' },
} satisfies Meta<typeof RowActions>

export default meta
type Story = StoryObj<typeof meta>

const openMenu = (canvasElement: HTMLElement) => userEvent.click(within(canvasElement).getByRole('button'))

export const Closed: Story = {}

export const Menu: Story = {
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await expect(await screen.findByRole('menuitem', { name: /edit/i })).toBeVisible()
        await expect(screen.getByRole('menuitem', { name: /delete/i })).toBeVisible()
    },
}

export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await screen.findByRole('menuitem', { name: /edit/i })
        await expect(screen.queryByRole('menuitem', { name: /delete/i })).toBeNull()
    },
}

export const DeleteDisabled: Story = {
    args: { deleteDisabled: true, deleteDisabledLabel: 'In use, cannot delete' },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await expect(await screen.findByRole('menuitem', { name: 'In use, cannot delete' })).toHaveAttribute('aria-disabled', 'true')
    },
}

export const ExtraItems: Story = {
    args: { leading: <DropdownMenuItem>Pin</DropdownMenuItem>, children: <DropdownMenuItem>Archive</DropdownMenuItem> },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await expect(await screen.findByRole('menuitem', { name: 'Pin' })).toBeVisible()
        await expect(screen.getByRole('menuitem', { name: 'Archive' })).toBeVisible()
    },
}

export const EditClicked: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /edit/i }))
        await expect(args.onEdit).toHaveBeenCalledOnce()
    },
}

export const DeleteConfirmed: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        const dialog = await screen.findByRole('alertdialog')
        await expect(within(dialog).getByText('Its history goes too.')).toBeVisible()
        await userEvent.click(within(dialog).getByRole('button', { name: /delete/i }))
        await waitFor(() => expect(args.onDelete).toHaveBeenCalledOnce())
    },
}

export const DeleteCancelled: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        const dialog = await screen.findByRole('alertdialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /cancel/i }))
        await expect(args.onDelete).not.toHaveBeenCalled()
    },
}
