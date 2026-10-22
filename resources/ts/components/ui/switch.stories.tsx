import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { Switch } from './switch'

const meta = { title: 'ui/Switch', component: Switch, args: { 'aria-label': 'Enabled' } } satisfies Meta<typeof Switch>

export default meta
type Story = StoryObj<typeof meta>

export const Off: Story = {
    play: async ({ canvasElement }) => {
        const toggle = within(canvasElement).getByRole('switch')
        await userEvent.click(toggle)
        await expect(toggle).toBeChecked()
    },
}
export const On: Story = { args: { defaultChecked: true } }
export const Disabled: Story = { args: { disabled: true } }
export const DisabledOn: Story = { args: { disabled: true, defaultChecked: true } }
