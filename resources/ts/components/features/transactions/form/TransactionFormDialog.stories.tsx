import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, categories, currencies, tags, transaction } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionFormDialog, toTransactionFormValues } from './TransactionFormDialog'

const meta = {
    title: 'features/transactions/form/TransactionFormDialog',
    component: TransactionFormDialog,
    parameters: { a11y: overlayA11y },
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/accounts', accounts),
            reply('get', '/categories', categories),
            reply('get', '/currencies', currencies),
            reply('get', '/tags', tags),
            reply('get', '/links', []),
        )
    },
} satisfies Meta<typeof TransactionFormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}
export const CreateWithDefaults: Story = {
    args: { defaultValues: { type: 'income', account_id: 2, amount: 50 } },
    play: async () => {
        await expect(await screen.findByRole('spinbutton', { name: /amount/i })).toHaveValue(50)
    },
}
export const Edit: Story = {
    args: { transaction: transaction({ status: 'confirmed' }) },
    play: async () => {
        await expect(await screen.findByDisplayValue('Weekly shop')).toBeVisible()
    },
}
export const EditPending: Story = {
    args: { transaction: transaction({ status: 'pending', date: '2099-06-01' }) },
}
export const Submitting: Story = { args: { isSubmitting: true } }
export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    args: { transaction: transaction() },
}
// The balance after the change sits in the dialog's header, out of the way of the form.
export const BalancePreviewInHeader: Story = {
    args: { transaction: transaction({ status: 'confirmed', amount: 20 }) },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect((await within(dialog).findAllByText(/120\.50/)).length).toBeGreaterThan(0)
    },
}

export const SubmittedFromTheFooter: Story = {
    args: { transaction: transaction({ status: 'confirmed' }) },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await within(dialog).findByDisplayValue('Weekly shop')
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledOnce())
    },
}

export const ChangesBecomeFormValues: Story = {
    play: async () => {
        const values = toTransactionFormValues(
            transaction({
                type: 'transfer', amount: 100, toAmount: 20000, toAccount: accounts[2], tags: [tags[0]],
                items: [{ id: 1, name: 'Fee', quantity: 2, pricePerUnit: 1, totalPrice: 2 }],
            }),
        )
        await expect(values).toMatchObject({
            type: 'transfer', amount: 100, to_amount: 20000, to_account_id: 3, tag_ids: [1],
            items: [{ name: 'Fee', quantity: 2, price_per_unit: 1 }],
        })
    },
}
