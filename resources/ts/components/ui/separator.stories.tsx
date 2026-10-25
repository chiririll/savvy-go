import type { Meta, StoryObj } from '@storybook/react-vite'

import { Separator } from './separator'

const meta = { title: 'ui/Separator', component: Separator } satisfies Meta<typeof Separator>

export default meta
type Story = StoryObj<typeof meta>

export const Horizontal: Story = {}
export const Vertical: Story = {
    args: { orientation: 'vertical' },
    decorators: [(Story) => <div className="h-8"><Story /></div>],
}
