import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { Checkbox } from './checkbox'

const meta = {
    title: 'ui/Checkbox',
    component: Checkbox,
    args: { 'aria-label': 'Accept' },
} satisfies Meta<typeof Checkbox>

export default meta
type Story = StoryObj<typeof meta>

export const Unchecked: Story = {
    play: async ({ canvasElement }) => {
        const box = within(canvasElement).getByRole('checkbox')
        await userEvent.click(box)
        await expect(box).toBeChecked()
        await userEvent.click(box)
        await expect(box).not.toBeChecked()
    },
}
export const Checked: Story = { args: { defaultChecked: true } }
export const Indeterminate: Story = { args: { checked: 'indeterminate' } }
export const Disabled: Story = { args: { disabled: true } }
export const DisabledChecked: Story = { args: { disabled: true, defaultChecked: true } }
export const Invalid: Story = { args: { 'aria-invalid': true } }
