import type { Meta, StoryObj } from '@storybook/react-vite'

import { Button } from './button'
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './card'

const meta = { title: 'ui/Card', component: Card } satisfies Meta<typeof Card>

export default meta
type Story = StoryObj<typeof meta>

export const Full: Story = {
    render: (args) => (
        <Card {...args} className="max-w-sm">
            <CardHeader>
                <CardTitle>Monthly budget</CardTitle>
                <CardDescription>Groceries and household</CardDescription>
                <CardAction><Button size="sm" variant="outline">Edit</Button></CardAction>
            </CardHeader>
            <CardContent>Spent 420 of 600</CardContent>
            <CardFooter><Button>Open</Button></CardFooter>
        </Card>
    ),
}

export const ContentOnly: Story = {
    render: (args) => (
        <Card {...args} className="max-w-sm">
            <CardContent>Just a body</CardContent>
        </Card>
    ),
}
