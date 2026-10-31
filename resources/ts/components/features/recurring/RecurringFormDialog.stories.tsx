import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { accounts, categories, currencies, recurring, tags } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { RecurringFormDialog } from './RecurringFormDialog'

const meta = {
    title: 'features/recurring/RecurringFormDialog',
    component: RecurringFormDialog,
    parameters: { a11y: overlayA11y },
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/accounts', accounts), reply('get', '/categories', categories), reply('get', '/tags', tags))
    },
} satisfies Meta<typeof RecurringFormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}
export const Edit: Story = {
    args: { recurring: recurring() },
    play: async () => {
        await expect(await screen.findByDisplayValue('Rent')).toBeVisible()
    },
}
export const EditTransfer: Story = {
    args: { recurring: recurring({ type: 'transfer', category: null, toAccount: accounts[1], toAmount: 920 }) },
}
export const Submitting: Story = { args: { isSubmitting: true } }
export const ReadOnly: Story = { parameters: { app: { role: 'viewer' } }, args: { recurring: recurring() } }

export const SubmittedFromTheFooter: Story = {
    args: { recurring: recurring() },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByRole('button', { name: /^cancel$/i })).toBeVisible()
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await expect(args.onSubmit).toHaveBeenCalledOnce()
    },
}
