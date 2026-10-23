import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { category } from '@/test/fixtures'

import { CategoryFormDialog } from './CategoryFormDialog'

const meta = {
    title: 'features/categories/CategoryFormDialog',
    component: CategoryFormDialog,
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
} satisfies Meta<typeof CategoryFormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}
export const CreateIncome: Story = { args: { defaultType: 'income' } }
export const Edit: Story = {
    args: { category: category() },
    play: async () => {
        await expect(await screen.findByDisplayValue('Groceries')).toBeVisible()
    },
}
export const EditDefault: Story = { args: { category: category({ isDefault: true }) } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const ReadOnly: Story = { parameters: { app: { role: 'viewer' } } }

export const SubmittedFromTheFooter: Story = {
    args: { category: category() },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await expect(args.onSubmit).toHaveBeenCalledWith(expect.objectContaining({ name: 'Groceries', type: 'expense' }))
    },
}
