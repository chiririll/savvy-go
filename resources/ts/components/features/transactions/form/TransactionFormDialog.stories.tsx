import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { toFormValues } from '@/lib/transaction-form-values'
import { accounts, categories, currencies, line, payment, tags, transaction } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { TransactionFormDialog } from './TransactionFormDialog'

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
    args: { defaultValues: { type: 'income', payments: [{ account_id: 2, amount: 50 }] } },
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
// Editing a posted transaction: the balance already has it, so the preview takes it back first.
export const BalancePreviewOfAnEdit: Story = {
    args: { transaction: transaction({ status: 'confirmed' }) },
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

export const SavedTransactionBecomesFormValues: Story = {
    play: async () => {
        const values = toFormValues(
            transaction({
                type: 'income',
                payments: [payment({ amount: 42.5 })],
                lines: [line({ categoryId: 2, note: '- salary' })],
                tags: [tags[0]],
            }),
        )
        await expect(values).toMatchObject({
            type: 'income',
            payments: [{ account_id: 1, amount: 42.5 }],
            default_category_id: 2,
            default_note: '- salary',
            tag_ids: [1],
        })
    },
}

// A payment in two lines becomes lines: the first with its note, and no default category.
export const LinesBecomeFormValues: Story = {
    play: async () => {
        const values = toFormValues(
            transaction({ lines: [line({ amount: 30, note: '- milk' }), line({ seq: 1, categoryId: 3, amount: 12.5 })] }),
        )
        await expect(values).toMatchObject({
            payments: [{ account_id: 1, amount: 42.5 }],
            lines: [
                { category_id: 1, amount: 30, note: '- milk' },
                { category_id: 3, amount: 12.5, note: null },
            ],
            default_category_id: null,
        })
    },
}

// Payments of the type's sign are the payments of the form; the ones of the opposite sign are the change.
export const PaymentsAndChangeBecomeFormValues: Story = {
    play: async () => {
        const values = toFormValues(
            transaction({
                payments: [
                    payment({ amount: -30 }),
                    payment({ account: accounts[1], accountId: 2, amount: -8 }),
                    payment({ account: accounts[2], accountId: 3, amount: 300 }),
                ],
                lines: [line({ amount: 30 }), line({ seq: 1, categoryId: 3, amount: 10, currencyId: 2 })],
            }),
        )
        await expect(values).toMatchObject({
            payments: [
                { account_id: 1, amount: 30 },
                { account_id: 2, amount: 8 },
            ],
            change: [{ account_id: 3, amount: 300 }],
        })
    },
}
