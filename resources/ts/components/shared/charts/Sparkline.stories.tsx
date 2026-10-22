import type { Meta, StoryObj } from '@storybook/react-vite'

import { Sparkline } from './Sparkline'

const meta = {
    title: 'shared/charts/Sparkline',
    component: Sparkline,
    args: { data: [3, 5, 4, 8, 6, 9, 7] },
} satisfies Meta<typeof Sparkline>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}
export const Success: Story = { args: { color: 'success' } }
export const Danger: Story = { args: { color: 'danger', data: [9, 8, 6, 7, 4, 3, 1] } }
export const Flat: Story = { args: { data: [5, 5, 5, 5] } }
export const NotEnoughData: Story = { args: { data: [5] } }
export const NoData: Story = { args: { data: [] } }
export const Negative: Story = { args: { data: [-5, -2, -8, -1] } }
export const Large: Story = { args: { width: 320, height: 80, strokeWidth: 3 } }
