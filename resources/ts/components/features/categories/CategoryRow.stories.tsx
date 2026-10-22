import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { categories, category } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'

import { CategoryRow } from './CategoryRow'

const meta = {
    title: 'features/categories/CategoryRow',
    component: CategoryRow,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/categories', categories)),
    args: { category: category({ transactionsCount: 0 }), onEdit: fn(), onDelete: fn(), onSetDefault: fn() },
} satisfies Meta<typeof CategoryRow>

export default meta
type Story = StoryObj<typeof meta>

const openMenu = (canvasElement: HTMLElement) =>
    userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))

export const Expense: Story = {}
export const Income: Story = { args: { category: category({ type: 'income', icon: '💼', color: '#2563eb', transactionsCount: 0 }) } }
export const Default: Story = { args: { category: category({ isDefault: true, transactionsCount: 0 }) } }
export const BuiltInName: Story = { args: { category: category({ name: '#GROCERIES', transactionsCount: 0 }) } }
export const BuiltInNameInRussian: Story = { ...BuiltInName, globals: { locale: 'ru' } }
export const NoColor: Story = { args: { category: category({ color: null, transactionsCount: 0 }) } }
export const LongName: Story = {
    args: { category: category({ name: 'A category name far too long for the line it sits on', transactionsCount: 0 }) },
}
export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const DeleteDisabled: Story = {
    args: { deleteDisabled: true, deleteDisabledLabel: 'The last category cannot be deleted' },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await expect(await screen.findByRole('menuitem', { name: /last category/i })).toHaveAttribute('aria-disabled', 'true')
    },
}

export const Deleted: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1)
    },
}

export const MadeDefault: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /default/i }))
        await expect(args.onSetDefault).toHaveBeenCalledWith(1)
    },
}

export const DefaultCannotBeMadeDefault: Story = {
    args: { category: category({ isDefault: true, transactionsCount: 0 }) },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await screen.findByRole('menuitem', { name: /edit/i })
        await expect(screen.queryByRole('menuitem', { name: /set as default/i })).toBeNull()
    },
}

export const SettingDefault: Story = {
    args: { isSettingDefault: true },
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await expect(await screen.findByRole('menuitem', { name: /default/i })).toHaveAttribute('aria-disabled', 'true')
    },
}

// A category with transactions asks where they go before it is deleted.
export const DeleteNeedsReplacement: Story = {
    args: { category: category({ transactionsCount: 12 }) },
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        const dialog = await screen.findByRole('alertdialog')
        const confirm = within(dialog).getByRole('button', { name: /delete/i })
        await expect(confirm).toBeDisabled()
        await userEvent.click(within(dialog).getByRole('combobox'))
        await userEvent.click(await screen.findByRole('option', { name: /Transport/ }))
        await waitFor(() => expect(within(dialog).getByRole('button', { name: /delete/i })).toBeEnabled())
        await userEvent.click(within(dialog).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1, 3)
    },
}

// In a space kept in step with others, even an unused category may be deleted with a replacement.
export const SyncedUnused: Story = {
    args: { synced: true },
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        const dialog = await screen.findByRole('alertdialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1, undefined)
    },
}
