import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { account } from '@/test/fixtures'
import { currencyApi } from '@/test/api-mocks'

import { AccountFormDialog } from './AccountFormDialog'

const meta = {
    title: 'features/accounts/AccountFormDialog',
    component: AccountFormDialog,
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
    beforeEach: ({ msw }) => msw.use(...currencyApi()),
} satisfies Meta<typeof AccountFormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}

export const Edit: Story = {
    args: { account: account({ name: 'Euro card', type: 'bank' }) },
    play: async () => {
        await expect(await screen.findByRole('heading', { name: /edit/i })).toBeVisible()
        await expect(screen.getByDisplayValue('Euro card')).toBeVisible()
    },
}

// An account that came from a debt is edited as a bank account.
export const EditDebtAccount: Story = { args: { account: account({ type: 'debt' }) } }

export const Submitting: Story = { args: { isSubmitting: true } }
export const Closed: Story = { args: { open: false } }
export const ReadOnly: Story = { parameters: { app: { role: 'viewer' } } }

export const Cancelled: Story = {
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: /cancel/i }))
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}

export const SubmittedFromTheFooter: Story = {
    args: { account: account({ name: 'Cash', currency: { ...account().currency!, id: 1 } }) },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await expect(args.onSubmit).toHaveBeenCalledOnce()
    },
}
