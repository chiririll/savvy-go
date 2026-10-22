import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { accounts, currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { AccountList } from './AccountList'

const meta = {
    title: 'features/accounts/AccountList',
    component: AccountList,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        accounts,
        emptyTitle: 'No accounts',
        emptyDescription: 'Create the first one.',
        onEdit: fn(),
        onDelete: fn(),
    },
} satisfies Meta<typeof AccountList>

export default meta
type Story = StoryObj<typeof meta>

export const Filled: Story = {}
export const Loading: Story = { args: { isLoading: true } }
export const Empty: Story = { args: { accounts: [], onCreate: fn(), createLabel: 'Create account' } }

export const EmptyReadOnly: Story = {
    args: { accounts: [], onCreate: fn(), createLabel: 'Create account', isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const Reorderable: Story = {
    args: { onReorder: fn() },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getAllByRole('button', { name: /reorder/i })).toHaveLength(accounts.length)
    },
}

// A viewer cannot reorder, even when the page offers it.
export const ReorderOffWhenReadOnly: Story = {
    args: { onReorder: fn(), isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button', { name: /reorder/i })).toBeNull()
    },
}

export const ReorderedByKeyboard: Story = {
    args: { onReorder: fn() },
    play: async ({ canvasElement, args }) => {
        const handles = within(canvasElement).getAllByRole('button', { name: /reorder/i })
        handles[0].focus()
        await userEvent.keyboard(' ')
        await userEvent.keyboard('{ArrowDown}')
        await userEvent.keyboard(' ')
        await expect(args.onReorder).toHaveBeenCalledOnce()
        const reordered = (args.onReorder as ReturnType<typeof fn>).mock.calls[0][0] as typeof accounts
        await expect(reordered.map((item) => item.id)).toEqual([2, 1, 3, 4])
    },
}
