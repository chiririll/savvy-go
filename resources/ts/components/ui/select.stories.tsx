import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './select'
import { overlayA11y } from '@/test/a11y'

const meta = {
    title: 'ui/Select',
    component: Select,
    parameters: { a11y: overlayA11y },
    render: (args) => (
        <Select {...args}>
            <SelectTrigger aria-label="Currency" className="w-48">
                <SelectValue placeholder="Pick one" />
            </SelectTrigger>
            <SelectContent>
                <SelectItem value="usd">USD</SelectItem>
                <SelectItem value="eur">EUR</SelectItem>
                <SelectItem value="jpy" disabled>JPY</SelectItem>
            </SelectContent>
        </Select>
    ),
} satisfies Meta<typeof Select>

export default meta
type Story = StoryObj<typeof meta>

export const Placeholder: Story = {}

export const Selected: Story = { args: { defaultValue: 'eur' } }

export const Disabled: Story = { args: { disabled: true, defaultValue: 'usd' } }

export const Opened: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('combobox'))
        await expect(await screen.findByRole('option', { name: 'USD' })).toBeVisible()
        await expect(screen.getByRole('option', { name: 'JPY' })).toHaveAttribute('aria-disabled', 'true')
    },
}

export const Picked: Story = {
    play: async ({ canvasElement }) => {
        const trigger = within(canvasElement).getByRole('combobox')
        await userEvent.click(trigger)
        await userEvent.click(await screen.findByRole('option', { name: 'EUR' }))
        await expect(trigger).toHaveTextContent('EUR')
    },
}
