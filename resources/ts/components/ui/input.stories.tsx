import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { Input } from './input'

const meta = {
    title: 'ui/Input',
    component: Input,
    args: { 'aria-label': 'Name' },
} satisfies Meta<typeof Input>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {
    args: { placeholder: 'Name' },
    play: async ({ canvasElement }) => {
        const input = within(canvasElement).getByRole('textbox')
        await userEvent.type(input, 'Groceries')
        await expect(input).toHaveValue('Groceries')
    },
}
export const Filled: Story = { args: { defaultValue: 'Groceries' } }
export const Disabled: Story = { args: { disabled: true, defaultValue: 'Locked' } }
export const ReadOnly: Story = { args: { readOnly: true, defaultValue: 'Read only' } }
export const Invalid: Story = { args: { 'aria-invalid': true, defaultValue: 'bad' } }
export const Password: Story = { args: { type: 'password', defaultValue: 'secret' } }
export const Numeric: Story = { args: { type: 'number', defaultValue: 12.5, step: 0.01 } }
export const Overflowing: Story = {
    args: { defaultValue: 'A value much longer than the field can show at once, to see how it is cut' },
    decorators: [(Story) => <div className="w-48"><Story /></div>],
}
