import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, waitFor, within } from 'storybook/test'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { FormWrapper } from './FormWrapper'

const meta = {
    tags: ['desktop-only'],
    title: 'shared/form/FormWrapper',
    component: FormWrapper,
    args: {
        children: (
            <div className="space-y-2">
                <Input aria-label="Name" defaultValue="Groceries" />
                <Button>Save</Button>
            </div>
        ),
    },
} satisfies Meta<typeof FormWrapper>

export default meta
type Story = StoryObj<typeof meta>

export const Editable: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('textbox')).toBeEnabled()
    },
}

// In a space the user may only read, every field is disabled.
export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await waitFor(() => expect(canvas.getByRole('textbox')).toBeDisabled())
        await expect(canvas.getByRole('button')).toBeDisabled()
    },
}

export const EditorCanWrite: Story = {
    parameters: { app: { role: 'editor' } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('textbox')).toBeEnabled()
    },
}
