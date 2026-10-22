import type { Meta, StoryObj } from '@storybook/react-vite'

import { Textarea } from './textarea'

const meta = {
    title: 'ui/Textarea',
    component: Textarea,
    args: { 'aria-label': 'Note' },
} satisfies Meta<typeof Textarea>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = { args: { placeholder: 'Note' } }
export const Filled: Story = { args: { defaultValue: 'Two lines\nof text' } }
export const Disabled: Story = { args: { disabled: true, defaultValue: 'Locked' } }
export const Invalid: Story = { args: { 'aria-invalid': true, defaultValue: 'bad' } }
