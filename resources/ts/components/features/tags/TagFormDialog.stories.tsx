import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { TagFormDialog } from './TagFormDialog'

const meta = {
    title: 'features/tags/TagFormDialog',
    component: TagFormDialog,
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
} satisfies Meta<typeof TagFormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}
export const Edit: Story = {
    args: { tag: { id: 1, name: 'holiday', transactionsCount: 4 } },
    play: async () => {
        await expect(await screen.findByDisplayValue('holiday')).toBeVisible()
    },
}
export const Submitting: Story = { args: { isSubmitting: true } }
export const ReadOnly: Story = { parameters: { app: { role: 'viewer' } } }

export const SubmittedFromTheFooter: Story = {
    args: { tag: { id: 1, name: 'holiday' } },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await expect(args.onSubmit).toHaveBeenCalledWith({ name: 'holiday' })
    },
}
