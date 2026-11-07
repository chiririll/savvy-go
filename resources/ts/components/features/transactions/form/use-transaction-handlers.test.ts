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

    it('ignores a transfer: its tabs are disabled until the transfer form is back', () => {
        const { result } = setup()

        act(() => result.current.handleTypeChange('transfer'))
        expect(result.current.form.getValues('type')).toBe('expense')
        expect(result.current.form.getValues('default_category_id')).toBe(1)
    })
})

describe('useTransactionHandlers submit', () => {
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
})
