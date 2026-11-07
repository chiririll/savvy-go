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
        await expect(await screen.findByRole('textbox', { name: /^amount$/i })).toHaveValue('50')
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

// A saved transfer opens with the accounts, the amount sent (the fee taken off) and the amount received.
export const TransferBecomesFormValues: Story = {
    args: {
        transaction: transaction({
            type: 'transfer',
            status: 'confirmed',
            payments: [payment({ amount: -102 }), payment({ account: accounts[1], accountId: 2, amount: 80 })],
            lines: [line({ amount: 2, categoryId: 3, category: categories[2] })],
        }),
    },
    play: async () => {
        const amounts = await screen.findAllByRole('spinbutton', {}, { timeout: 5000 })
        await expect(amounts.slice(0, 2).map((a) => (a as HTMLInputElement).value)).toEqual(['100', '80'])
    },
}

// A transaction that allows only its details: what it will not change is shown as text, with the reason.
const debtPayment = (overrides = {}) =>
    transaction({
        type: 'debt_payment',
        status: 'confirmed',
        description: 'Back to Alex',
        payments: [payment({ amount: -40 })],
        lines: [],
        restrictions: [{ reason: 'debt', message: 'this transaction belongs to a debt' }],
        actions: { edit: false, editDetails: true, duplicate: false, delete: true, confirm: false, skip: false },
        ...overrides,
    })

export const RestrictedByADebt: Story = {
    args: { transaction: debtPayment() },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(await within(dialog).findByRole('note')).toHaveTextContent(/debt/i)
        await expect(within(dialog).getByDisplayValue('Back to Alex')).toBeEnabled()
        // The amount and the account are text, not fields.
        await expect(within(dialog).queryByRole('textbox', { name: /^amount$/i })).toBeNull()
        await expect(within(dialog).getByText(/40/)).toBeVisible()
    },
}

export const RestrictedSavesOnlyTheDetails: Story = {
    args: { transaction: debtPayment({ tags: [tags[0]] }), onSubmitDetails: fn() },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        const title = await within(dialog).findByDisplayValue('Back to Alex')
        await userEvent.clear(title)
        await userEvent.type(title, 'Paid back')
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(args.onSubmitDetails).toHaveBeenCalledOnce())
        const sent = (args.onSubmitDetails as ReturnType<typeof fn>).mock.calls[0][0]
        await expect(sent).toEqual({ description: 'Paid back', tag_ids: [tags[0].id] })
    },
}

// One side of a transfer between spaces with fee lines: the lines keep their amounts, the categories and notes change.
export const RestrictedLinesChangeCategoryAndNote: Story = {
    args: {
        transaction: transaction({
            status: 'confirmed',
            restrictions: [{ reason: 'space_transfer', message: 'one side of a transfer' }],
            actions: { edit: true, editDetails: true, duplicate: false, delete: false, confirm: false, skip: false },
            payments: [payment({ amount: -50 })],
            lines: [line({ amount: 30, note: '- bread' }), line({ seq: 1, categoryId: 3, category: categories[2], amount: 20 })],
        }),
        onSubmitDetails: fn(),
    },
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        const group = await within(dialog).findByRole('group', { name: /lines|строки/i })
        await expect(within(group).queryByRole('textbox', { name: /^amount$/i })).toBeNull()
        await expect(within(group).getAllByRole('combobox')).toHaveLength(2)
        await userEvent.click(within(dialog).getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(args.onSubmitDetails).toHaveBeenCalledOnce())
        const sent = (args.onSubmitDetails as ReturnType<typeof fn>).mock.calls[0][0]
        await expect(sent.lines).toEqual([
            { category_id: 1, currency_id: 1, amount: 30, note: '- bread' },
            { category_id: 3, currency_id: 1, amount: 20, note: null },
        ])
        await expect(sent).not.toHaveProperty('payments')
    },
}

// A shape the form cannot show whole is not refused: its details can still be changed.
export const UnshowableShapeOpensRestricted: Story = {
    args: {
        transaction: transaction({
            status: 'confirmed',
            payments: [
                payment({ amount: -50 }),
                payment({ account: accounts[1], accountId: 2, amount: -5 }),
                payment({ account: accounts[2], accountId: 3, amount: 9 }),
            ],
            type: 'transfer_out',
            lines: [],
        }),
    },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(await within(dialog).findByRole('note')).toHaveTextContent(/can't show/i)
        await expect(within(dialog).queryByRole('textbox', { name: /^amount$/i })).toBeNull()
    },
}
