import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { TransactionsListFooter } from './TransactionsListFooter'

const meta = {
    title: 'features/transactions/list/TransactionsListFooter',
    component: TransactionsListFooter,
    args: {
        loadedCount: 20,
        total: 87,
        lastPage: 5,
        hasNextPage: true,
        isFetchingNextPage: false,
        isFetchNextPageError: false,
        onLoadMore: fn(),
    },
} satisfies Meta<typeof TransactionsListFooter>

export default meta
type Story = StoryObj<typeof meta>

export const MoreToLoad: Story = {}
export const Loading: Story = {
    args: { isFetchingNextPage: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('status')).toBeVisible()
    },
}
export const EndOfList: Story = { args: { loadedCount: 87, hasNextPage: false } }
export const SinglePage: Story = {
    args: { loadedCount: 5, total: 5, lastPage: 1, hasNextPage: false },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText(/end/i)).toBeNull()
    },
}
export const CountUnknown: Story = { args: { total: undefined } }

export const FailedToLoadMore: Story = {
    args: { isFetchNextPageError: true },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /try again|retry/i }))
        await expect(args.onLoadMore).toHaveBeenCalled()
    },
}

// The sentinel loads the next page by itself once it scrolls into view.
export const LoadsWhenScrolledIn: Story = {
    play: async ({ args }) => {
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(args.onLoadMore).toHaveBeenCalledOnce()
    },
}

export const DoesNotLoadWhileFetching: Story = {
    args: { isFetchingNextPage: true },
    play: async ({ args }) => {
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(args.onLoadMore).not.toHaveBeenCalled()
    },
}

export const DoesNotLoadAfterAnError: Story = {
    args: { isFetchNextPageError: true },
    play: async ({ args }) => {
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(args.onLoadMore).not.toHaveBeenCalled()
    },
}
