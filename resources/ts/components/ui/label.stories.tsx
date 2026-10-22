import type { Meta, StoryObj } from '@storybook/react-vite'

import { Input } from './input'
import { Label } from './label'

const meta = { title: 'ui/Label', component: Label } satisfies Meta<typeof Label>

export default meta
type Story = StoryObj<typeof meta>

export const ForInput: Story = {
    render: () => (
        <div className="grid gap-2">
            <Label htmlFor="name">Name</Label>
            <Input id="name" />
        </div>
    ),
}
