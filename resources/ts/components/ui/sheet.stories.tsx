import type { Meta, StoryObj } from '@storybook/react-vite'

import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from './sheet'

const meta = {
    title: 'ui/Sheet',
    component: Sheet,
    args: { defaultOpen: true },
    render: (args) => (
        <Sheet {...args}>
            <SheetContent side="right">
                <SheetHeader>
                    <SheetTitle>Filters</SheetTitle>
                    <SheetDescription>Narrow the list down.</SheetDescription>
                </SheetHeader>
            </SheetContent>
        </Sheet>
    ),
} satisfies Meta<typeof Sheet>

export default meta
type Story = StoryObj<typeof meta>

export const Right: Story = {}

export const Left: Story = {
    render: (args) => (
        <Sheet {...args}>
            <SheetContent side="left">
                <SheetHeader><SheetTitle>Menu</SheetTitle><SheetDescription>Navigate.</SheetDescription></SheetHeader>
            </SheetContent>
        </Sheet>
    ),
}

export const Bottom: Story = {
    render: (args) => (
        <Sheet {...args}>
            <SheetContent side="bottom">
                <SheetHeader><SheetTitle>Actions</SheetTitle><SheetDescription>Pick one.</SheetDescription></SheetHeader>
            </SheetContent>
        </Sheet>
    ),
}
