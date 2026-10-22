import type { Meta, StoryObj } from '@storybook/react-vite'

import { Skeleton } from './skeleton'

const meta = {
    title: 'ui/Skeleton',
    component: Skeleton,
    args: { className: 'h-4 w-48' },
} satisfies Meta<typeof Skeleton>

export default meta
type Story = StoryObj<typeof meta>

export const Line: Story = {}
export const Circle: Story = { args: { className: 'size-10 rounded-full' } }
