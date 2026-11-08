import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, recurring, recurringTemplate, tags } from '@/test/fixtures'
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
        msw.use(
            reply('get', '/accounts', accounts),
            reply('get', '/categories', categories),
            reply('get', '/currencies', currencies),
            reply('get', '/tags', tags),
        )
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
    args: {
        recurring: recurring({
            template: recurringTemplate({
                type: 'transfer',
                description: 'To the card',
                payments: [
                    { account_id: 1, amount: -100 },
                    { account_id: 2, amount: 80 },
                ],
                lines: [],
            }),
        }),
    },
}

// The amount received was left open: it stays open when the rule is opened.
export const EditTransferWithTheAmountReceivedOpen: Story = {
    args: {
        recurring: recurring({
            template: recurringTemplate({
                type: 'transfer',
                description: 'To the card',
                payments: [
                    { account_id: 1, amount: -100 },
                    { account_id: 2, amount: null, estimated: true },
                ],
                lines: [],
            }),
        }),
    },
    play: async () => {
        await expect(await screen.findByPlaceholderText(/at each run/i)).toBeVisible()
    },
}

export const EditStoppedRule: Story = {
    args: { recurring: recurring({ isActive: false, error: 'unknown account', nextRunDate: null }) },
    play: async () => {
        await expect(await screen.findByText(/unknown account/i)).toBeVisible()
    },
}

// A rule the form cannot show whole (here, an account of it is gone) is not opened, so nothing is lost by saving.
export const RuleTheFormCannotShow: Story = {
    args: {
        recurring: recurring({ template: recurringTemplate({ payments: [{ account_id: 99, amount: -5 }] }) }),
    },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(await within(dialog).findByText(/cannot show this rule/i)).toBeVisible()
        await expect(within(dialog).queryByRole('button', { name: /^save$/i })).toBeNull()
    },
}

export const Submitting: Story = { args: { isSubmitting: true } }
export const ReadOnly: Story = { parameters: { app: { role: 'viewer' } }, args: { recurring: recurring() } }

export const SubmittedFromTheFooter: Story = {
    args: { recurring: recurring() },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByRole('button', { name: /^cancel$/i })).toBeVisible()
        // The accounts come from the mocked API: the form is ready when the first one is drawn.
        await waitFor(() => expect(dialog).toHaveTextContent(/cash/i))
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await expect(args.onSubmit).toHaveBeenCalledOnce()
    },
}
