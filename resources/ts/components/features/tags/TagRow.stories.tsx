import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { TagRow } from './TagRow'

const meta = {
    title: 'features/tags/TagRow',
    component: TagRow,
    parameters: { a11y: overlayA11y },
    args: { tag: { id: 1, name: 'holiday', transactionsCount: 4 }, onEdit: fn(), onDelete: fn() },
} satisfies Meta<typeof TagRow>

export default meta
type Story = StoryObj<typeof meta>

export const Used: Story = {}
export const Unused: Story = { args: { tag: { id: 2, name: 'spare', transactionsCount: 0 } } }
export const CountMissing: Story = { args: { tag: { id: 3, name: 'new' } } }
export const BuiltInName: Story = { args: { tag: { id: 4, name: '#VACATION', transactionsCount: 2 } } }
export const LongName: Story = { args: { tag: { id: 5, name: 'a-tag-name-that-is-much-too-long-for-its-row', transactionsCount: 12 } } }
export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('#holiday'))
        await expect(args.onEdit).toHaveBeenCalledWith(args.tag)
    },
}

export const Deleted: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1)
    },
}
