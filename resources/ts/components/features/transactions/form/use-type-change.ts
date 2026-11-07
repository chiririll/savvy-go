import { useRef } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import type { TransactionFormValues } from '@/schemas/transactions'

/** A transfer moves money between two accounts: it has no lines or change, and one payment that sends. */
function toTransfer(form: UseFormReturn<TransactionFormValues>, from: TransactionFormValues['type'], to: 'transfer' | 'transfer_out') {
    form.setValue('lines', [])
    form.setValue('change', [])
    form.setValue('payments', form.getValues('payments').slice(0, 1))
    // The other space has its own account ids, and takes no fee.
    if (from !== to && (from === 'transfer_out' || to === 'transfer_out')) {
        form.setValue('to_account_id', null)
        form.setValue('to_space_id', null)
        form.setValue('to_amount', null)
        form.setValue('fee', null)
    }
}

/**
 * What switching the type of the transaction does to the form: a transfer has no lines or change and one payment,
 * and each of income and expense keeps its own category, so a slip onto another tab loses nothing.
 */
export function useTypeChange(
    form: UseFormReturn<TransactionFormValues>,
    onTypeChange?: (type: TransactionFormValues['type']) => void,
) {
    const categoryByType = useRef<Partial<Record<TransactionFormValues['type'], number>>>({})

    const handleTypeChange = (value: string) => {
        if (value !== 'income' && value !== 'expense' && value !== 'transfer' && value !== 'transfer_out') {
            return
        }
        const current = form.getValues('type')
        const categoryId = form.getValues('default_category_id')
        if (categoryId && (current === 'income' || current === 'expense')) {
            categoryByType.current[current] = categoryId
        }
        form.setValue('type', value)
        if (value === 'transfer' || value === 'transfer_out') {
            toTransfer(form, current, value)
            onTypeChange?.(value)
            return
        }
        // The categories of the lines are of the old type too.
        const lines = form.getValues('lines') ?? []
        if (lines.some((line) => line.category_id)) {
            form.setValue('lines', lines.map((line) => ({ ...line, category_id: null })))
        }
        // Null lets the model pick the default category of the type.
        form.setValue('default_category_id', categoryByType.current[value] ?? null)
        onTypeChange?.(value)
    }

    return handleTypeChange
}
