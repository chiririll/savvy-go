import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, waitFor } from 'storybook/test'

import { InfiniteScrollSentinel } from './InfiniteScrollSentinel'

const meta = {
    title: 'shared/table/InfiniteScrollSentinel',
    component: InfiniteScrollSentinel,
    args: { enabled: true, onVisible: fn() },
} satisfies Meta<typeof InfiniteScrollSentinel>

export default meta
type Story = StoryObj<typeof meta>

export const InView: Story = {
    play: async ({ args }) => {
        await waitFor(() => expect(args.onVisible).toHaveBeenCalledOnce())
    },
}

export const Disabled: Story = {
    args: { enabled: false },
    play: async ({ args }) => {
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(args.onVisible).not.toHaveBeenCalled()
    },
}

// Far below the fold: nothing is loaded until the reader scrolls there.
export const FarAway: Story = {
    args: { rootMargin: '0px' },
    render: (args) => (
        <>
            <div className="h-[3000px]" />
            <InfiniteScrollSentinel {...args} />
        </>
    ),
    play: async ({ args }) => {
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(args.onVisible).not.toHaveBeenCalled()
    },
}
