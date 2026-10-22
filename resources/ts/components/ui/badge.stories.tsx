import type { Meta, StoryObj } from '@storybook/react-vite'

import { Badge } from './badge'

const meta = { title: 'ui/Badge', component: Badge, args: { children: 'Active' } } satisfies Meta<typeof Badge>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}
export const Secondary: Story = { args: { variant: 'secondary' } }
export const Destructive: Story = { args: { variant: 'destructive', children: 'Overdue' } }
export const Outline: Story = { args: { variant: 'outline' } }
export const LongText: Story = {
    args: { children: 'A very long label that should not break the layout of the row it sits in' },
    decorators: [(Story) => <div className="w-48"><Story /></div>],
}
