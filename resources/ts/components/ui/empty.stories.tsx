import type { Meta, StoryObj } from '@storybook/react-vite'
import { Inbox } from 'lucide-react'

import { Button } from './button'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from './empty'

const meta = { title: 'ui/Empty', component: Empty } satisfies Meta<typeof Empty>

export default meta
type Story = StoryObj<typeof meta>

export const WithAction: Story = {
    render: (args) => (
        <Empty {...args}>
            <EmptyHeader>
                <EmptyMedia variant="icon"><Inbox /></EmptyMedia>
                <EmptyTitle>No transactions yet</EmptyTitle>
                <EmptyDescription>Add the first one to see it here.</EmptyDescription>
            </EmptyHeader>
            <EmptyContent><Button>Add transaction</Button></EmptyContent>
        </Empty>
    ),
}

export const TitleOnly: Story = {
    render: (args) => (
        <Empty {...args}>
            <EmptyHeader><EmptyTitle>Nothing here</EmptyTitle></EmptyHeader>
        </Empty>
    ),
}
