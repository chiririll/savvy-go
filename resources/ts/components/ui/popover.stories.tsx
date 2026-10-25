import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { Button } from './button'
import { Popover, PopoverContent, PopoverTrigger } from './popover'

const meta = {
    title: 'ui/Popover',
    component: Popover,
    render: (args) => (
        <Popover {...args}>
            <PopoverTrigger asChild><Button variant="outline">Details</Button></PopoverTrigger>
            <PopoverContent aria-label="Details">Rate fetched today</PopoverContent>
        </Popover>
    ),
} satisfies Meta<typeof Popover>

export default meta
type Story = StoryObj<typeof meta>

export const Opened: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Details' }))
        await expect(await screen.findByText('Rate fetched today')).toBeVisible()
    },
}
