import type { Meta, StoryObj } from '@storybook/react-vite'
import { Plus } from 'lucide-react'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Button } from './button'

const meta = {
    title: 'ui/Button',
    component: Button,
    args: { children: 'Save', onClick: fn() },
} satisfies Meta<typeof Button>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Save' }))
        await expect(args.onClick).toHaveBeenCalledOnce()
    },
}

export const Disabled: Story = {
    args: { disabled: true },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button'), { pointerEventsCheck: 0 })
        await expect(args.onClick).not.toHaveBeenCalled()
    },
}

export const WithIcon: Story = { args: { children: (<><Plus /> Add</>) } }
export const Destructive: Story = { args: { variant: 'destructive', children: 'Delete' } }
export const Outline: Story = { args: { variant: 'outline' } }
export const Secondary: Story = { args: { variant: 'secondary' } }
export const Ghost: Story = { args: { variant: 'ghost' } }
export const Link: Story = { args: { variant: 'link' } }
export const Small: Story = { args: { size: 'sm' } }
export const Large: Story = { args: { size: 'lg' } }
export const Icon: Story = { args: { size: 'icon', 'aria-label': 'Add', children: <Plus /> } }
