// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { useForm } from 'react-hook-form'
import { describe, expect, it, vi } from 'vitest'
import type { TransactionFormValues } from '@/schemas/transactions'
import { useTransactionHandlers } from './use-transaction-handlers'

vi.mock('@/components/shared', () => ({
    useNegativeBalanceConfirm: () => ({ confirmIfNeeded: vi.fn(), dialog: null }),
}))

function setup() {
    return renderHook(() => {
        const form = useForm<TransactionFormValues>({
            defaultValues: { type: 'expense', category_id: 1 } as TransactionFormValues,
        })
        const handlers = useTransactionHandlers({
            form,
            rate: {} as never,
            warnings: [],
            dateRequired: false,
            isSpaceTransfer: false,
            onSubmit: vi.fn(),
        })
        return { form, ...handlers }
    })
}

describe('useTransactionHandlers type change', () => {
    it('restores the category chosen earlier for a type', () => {
        const { result } = setup()

        act(() => result.current.handleTypeChange('income'))
        expect(result.current.form.getValues('category_id')).toBeNull()

        act(() => result.current.form.setValue('category_id', 7))
        act(() => result.current.handleTypeChange('expense'))
        expect(result.current.form.getValues('category_id')).toBe(1)

        act(() => result.current.handleTypeChange('income'))
        expect(result.current.form.getValues('category_id')).toBe(7)
    })

    it('keeps the category across a visit to transfer', () => {
        const { result } = setup()

        act(() => result.current.handleTypeChange('transfer'))
        act(() => result.current.handleTypeChange('expense'))
        expect(result.current.form.getValues('category_id')).toBe(1)
    })
})
