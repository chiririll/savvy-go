import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { budget, categories, tags } from '@/test/fixtures'
import { currencyApi, reply } from '@/test/api-mocks'

import { BudgetFormDialog } from './BudgetFormDialog'

const meta = {
    title: 'features/budgets/BudgetFormDialog',
    component: BudgetFormDialog,
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
    beforeEach: ({ msw }) => {
        msw.use(...currencyApi(), reply('get', '/categories', categories), reply('get', '/tags', tags))
    },
} satisfies Meta<typeof BudgetFormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}
export const Edit: Story = {
    args: { budget: budget() },
    play: async () => {
        await expect(await screen.findByDisplayValue('Groceries')).toBeVisible()
    },
}
export const EditAllExpenses: Story = { args: { budget: budget({ isGlobal: true, categories: [] }) } }
export const Submitting: Story = { args: { isSubmitting: true } }
export const ReadOnly: Story = { parameters: { app: { role: 'viewer' } }, args: { budget: budget() } }

export const SubmittedFromTheFooter: Story = {
    args: { budget: budget() },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await expect(args.onSubmit).toHaveBeenCalledOnce()
    },
}
