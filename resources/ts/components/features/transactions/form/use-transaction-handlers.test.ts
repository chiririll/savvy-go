// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { useForm } from 'react-hook-form'
import { describe, expect, it, vi } from 'vitest'
import type { TransactionFormValues } from '@/schemas/transactions'
import { accounts } from '@/test/fixtures'
import { useTransactionHandlers } from './use-transaction-handlers'

const confirmIfNeeded = vi.fn((values, _warnings, onConfirm) => onConfirm(values))

vi.mock('@/components/shared', () => ({
    useNegativeBalanceConfirm: () => ({ confirmIfNeeded, dialog: null }),
}))

const filled: TransactionFormValues = {
    type: 'expense',
    description: 'Groceries',
    date: '2020-10-17',
    tag_ids: [],
    payments: [{ account_id: accounts[0].id, amount: 20 }],
    change: [],
    lines: [],
    default_category_id: 1,
    default_note: null,
}

function setup(values: Partial<TransactionFormValues> = {}, options: { dateRequired?: boolean } = {}) {
    const onSubmit = vi.fn()
    const hook = renderHook(() => {
        const form = useForm<TransactionFormValues>({ defaultValues: { ...filled, ...values } })
        const handlers = useTransactionHandlers({
            form,
            warnings: [],
            dateRequired: options.dateRequired ?? false,
            accounts,
            onSubmit,
        })
        return { form, errors: form.formState.errors, ...handlers }
    })
    return { ...hook, onSubmit }
}

describe('useTransactionHandlers type change', () => {
    it('restores the category chosen earlier for a type', () => {
        const { result } = setup()

        act(() => result.current.handleTypeChange('income'))
        expect(result.current.form.getValues('default_category_id')).toBeNull()

        act(() => result.current.form.setValue('default_category_id', 7))
        act(() => result.current.handleTypeChange('expense'))
        expect(result.current.form.getValues('default_category_id')).toBe(1)

        act(() => result.current.handleTypeChange('income'))
        expect(result.current.form.getValues('default_category_id')).toBe(7)
    })

    it('clears the categories of the lines: they belong to the other type', () => {
        const lines = [
            { category_id: 1, currency_id: null, amount: 12, note: 'milk' },
            { category_id: 3, currency_id: null, amount: 8, note: null },
        ]
        const { result } = setup({ lines })

        act(() => result.current.handleTypeChange('income'))

        expect(result.current.form.getValues('lines')).toEqual([
            { ...lines[0], category_id: null },
            { ...lines[1], category_id: null },
        ])
    })

    it('ignores a transfer: its tabs are disabled until the transfer form is back', () => {
        const { result } = setup()

        act(() => result.current.handleTypeChange('transfer'))
        expect(result.current.form.getValues('type')).toBe('expense')
        expect(result.current.form.getValues('default_category_id')).toBe(1)
    })
})

describe('useTransactionHandlers submit', () => {
    it('saves the lines, and what they leave uncovered in the default category', async () => {
        const lines = [{ category_id: 3, currency_id: null, amount: 12, note: null }]
        const { result, onSubmit } = setup({ lines })

        await act(() => result.current.handleSubmit())

        expect(onSubmit).toHaveBeenCalledWith(
            expect.objectContaining({
                lines: [
                    { category_id: 3, currency_id: accounts[0].currency!.id, amount: 12 },
                    { category_id: 1, currency_id: accounts[0].currency!.id, amount: 8 },
                ],
            }),
        )
    })

    it('does not ask for the default category when the lines say where all of it went', async () => {
        const lines = [{ category_id: 3, currency_id: null, amount: 20, note: null }]
        const { result, onSubmit } = setup({ lines, default_category_id: null })

        await act(() => result.current.handleSubmit())

        expect(onSubmit).toHaveBeenCalledWith(
            expect.objectContaining({ lines: [{ category_id: 3, currency_id: accounts[0].currency!.id, amount: 20 }] }),
        )
    })

    it('hands the API payload over, with the amount signed', async () => {
        const { result, onSubmit } = setup()

        await act(() => result.current.handleSubmit())

        expect(onSubmit).toHaveBeenCalledWith({
            type: 'expense',
            description: 'Groceries',
            date: '2020-10-17',
            payments: [{ account_id: accounts[0].id, amount: -20 }],
            lines: [{ category_id: 1, currency_id: accounts[0].currency!.id, amount: 20 }],
            tag_ids: [],
        })
    })

    it('does not save an empty form', async () => {
        const { result, onSubmit } = setup({ payments: [{ account_id: 0, amount: 0 }] })

        await act(() => result.current.handleSubmit())

        expect(onSubmit).not.toHaveBeenCalled()
    })

    it('asks for a date when the transaction already moved balances', async () => {
        const { result, onSubmit } = setup({ date: '' }, { dateRequired: true })

        await act(() => result.current.handleSubmit())

        expect(onSubmit).not.toHaveBeenCalled()
        expect(result.current.errors.date).toBeDefined()
    })

    it('puts an error the form cannot catch on the field it belongs to', async () => {
        const { result, onSubmit } = setup({
            payments: [{ account_id: accounts[0].id, amount: 5 }],
            change: [{ account_id: accounts[1].id, amount: 10 }],
        })

        await act(() => result.current.handleSubmit())

        expect(onSubmit).not.toHaveBeenCalled()
        expect(result.current.errors.change).toBeDefined()
    })

    it('puts the message of a change that is too large on the amount of the change', async () => {
        const { result } = setup({
            payments: [{ account_id: accounts[0].id, amount: 5 }],
            change: [{ account_id: accounts[1].id, amount: 10 }],
        })

        await act(() => result.current.handleSubmit())

        expect(result.current.errors.change?.[0]?.amount).toBeDefined()
    })

    it('puts the message of an account used twice on the second row that uses it', async () => {
        const { result, onSubmit } = setup({
            payments: [
                { account_id: accounts[0].id, amount: 5 },
                { account_id: accounts[1].id, amount: 5 },
            ],
            change: [{ account_id: accounts[1].id, amount: 1 }],
        })

        await act(() => result.current.handleSubmit())

        expect(onSubmit).not.toHaveBeenCalled()
        expect(result.current.errors.change?.[0]?.account_id).toBeDefined()
        expect(result.current.errors.payments).toBeUndefined()
    })

    it('saves several payments and the change, each with its sign', async () => {
        const { result, onSubmit } = setup({
            payments: [
                { account_id: accounts[0].id, amount: 30 },
                { account_id: accounts[1].id, amount: 8 },
            ],
            change: [{ account_id: accounts[2].id, amount: 100 }],
        })

        await act(() => result.current.handleSubmit())

        expect(onSubmit).toHaveBeenCalledWith(
            expect.objectContaining({
                payments: [
                    { account_id: accounts[0].id, amount: -30 },
                    { account_id: accounts[1].id, amount: -8 },
                    { account_id: accounts[2].id, amount: 100 },
                ],
            }),
        )
    })

    it('asks about every account that goes negative before saving several payments', async () => {
        const warnings = [
            { accountName: 'Cash', currentBalance: 10, resultingBalance: -5, currency: accounts[0].currency },
            { accountName: 'Euro card', currentBalance: 2, resultingBalance: -1, currency: accounts[1].currency },
        ]
        const onSubmit = vi.fn()
        const { result } = renderHook(() => {
            const form = useForm<TransactionFormValues>({
                defaultValues: {
                    ...filled,
                    payments: [
                        { account_id: accounts[0].id, amount: 15 },
                        { account_id: accounts[1].id, amount: 3 },
                    ],
                },
            })
            return useTransactionHandlers({ form, warnings, dateRequired: false, accounts, onSubmit })
        })

        await act(() => result.current.handleSubmit())

        expect(confirmIfNeeded).toHaveBeenLastCalledWith(expect.anything(), warnings, onSubmit)
    })
})
