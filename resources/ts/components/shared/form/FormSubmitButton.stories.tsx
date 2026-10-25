import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { FormSubmitButton } from './FormSubmitButton'

const meta = { title: 'shared/form/FormSubmitButton', component: FormSubmitButton } satisfies Meta<typeof FormSubmitButton>

export default meta
type Story = StoryObj<typeof meta>

export const Idle: Story = {}

export const Submitting: Story = {
    args: { isSubmitting: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button')).toBeDisabled()
    },
}

export const CustomLabel: Story = { args: { label: 'Create space' } }

export const Hidden: Story = {
    args: { hidden: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

