import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { ServerPagination } from './ServerPagination'

const meta = {
    title: 'shared/ServerPagination',
    component: ServerPagination,
    args: {
        onPageChange: fn(),
        meta: { current_page: 5, last_page: 12, from: 41, to: 50, total: 115 },
    },
} satisfies Meta<typeof ServerPagination>

export default meta
type Story = StoryObj<typeof meta>

export const Middle: Story = {}
export const FirstPage: Story = { args: { meta: { current_page: 1, last_page: 12, from: 1, to: 10, total: 115 } } }
export const LastPage: Story = { args: { meta: { current_page: 12, last_page: 12, from: 111, to: 115, total: 115 } } }
export const FewPages: Story = { args: { meta: { current_page: 2, last_page: 3, from: 11, to: 20, total: 25 } } }
export const WithoutCounts: Story = { args: { meta: { current_page: 2, last_page: 3 } } }
export const InfoHidden: Story = { args: { showInfo: false } }
export const WiderWindow: Story = { args: { delta: 2 } }
export const CustomLabel: Story = { args: { infoLabel: 'accounts' } }

export const SinglePage: Story = {
    args: { meta: { current_page: 1, last_page: 1, from: 1, to: 4, total: 4 } },
    play: async ({ canvasElement }) => {
        await expect(canvasElement.querySelector('nav')).toBeNull()
    },
}

export const NextPage: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByLabelText(/next/i))
        await expect(args.onPageChange).toHaveBeenCalledWith(6)
    },
}

export const NoPreviousOnFirstPage: Story = {
    args: { meta: { current_page: 1, last_page: 4 } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByLabelText(/previous/i), { pointerEventsCheck: 0 })
        await expect(args.onPageChange).not.toHaveBeenCalled()
    },
}
