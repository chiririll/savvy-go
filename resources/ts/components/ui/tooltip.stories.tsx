import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { Button } from './button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from './tooltip'

const meta = {
    title: 'ui/Tooltip',
    component: Tooltip,
    render: (args) => (
        <TooltipProvider>
            <Tooltip {...args}>
                <TooltipTrigger asChild><Button variant="outline">Hover me</Button></TooltipTrigger>
                <TooltipContent>Shows the full amount</TooltipContent>
            </Tooltip>
        </TooltipProvider>
    ),
} satisfies Meta<typeof Tooltip>

export default meta
type Story = StoryObj<typeof meta>

export const Hidden: Story = {}

export const Shown: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.hover(within(canvasElement).getByRole('button'))
        await expect(await screen.findByRole('tooltip')).toHaveTextContent('Shows the full amount')
    },
}

export const ShownByFocus: Story = {
    play: async ({ canvasElement }) => {
        within(canvasElement).getByRole('button').focus()
        await expect(await screen.findByRole('tooltip')).toBeVisible()
    },
}
