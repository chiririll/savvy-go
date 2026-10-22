import type { Meta, StoryObj } from '@storybook/react-vite'

import { Progress } from './progress'

const meta = {
    title: 'ui/Progress',
    component: Progress,
    args: { 'aria-label': 'Budget used' },
} satisfies Meta<typeof Progress>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = { args: { value: 0 } }
export const Half: Story = { args: { value: 50 } }
export const Full: Story = { args: { value: 100 } }
export const Unset: Story = {}
