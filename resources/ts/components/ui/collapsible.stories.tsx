import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { Button } from './button'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from './collapsible'

const meta = {
    title: 'ui/Collapsible',
    component: Collapsible,
    render: (args) => (
        <Collapsible {...args}>
            <CollapsibleTrigger asChild><Button variant="outline">More</Button></CollapsibleTrigger>
            <CollapsibleContent>Hidden details</CollapsibleContent>
        </Collapsible>
    ),
} satisfies Meta<typeof Collapsible>

export default meta
type Story = StoryObj<typeof meta>

export const Collapsed: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText('Hidden details')).toBeNull()
    },
}

export const Expanded: Story = { args: { defaultOpen: true } }

export const Toggled: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'More' }))
        await expect(within(canvasElement).getByText('Hidden details')).toBeVisible()
    },
}
