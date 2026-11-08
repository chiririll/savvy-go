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
    // The lines as they were on each of income and expense, and on the last of them before a transfer wiped the lines.
    const linesByType = useRef<Partial<Record<TransactionFormValues['type'], TransactionFormValues['lines']>>>({})
    const lastLines = useRef<NonNullable<TransactionFormValues['lines']>>([])

    const handleTypeChange = (value: string) => {
        if (value !== 'income' && value !== 'expense' && value !== 'transfer' && value !== 'transfer_out') {
            return
        }
        const current = form.getValues('type')
        const categoryId = form.getValues('default_category_id')
        if (categoryId && (current === 'income' || current === 'expense')) {
            categoryByType.current[current] = categoryId
        }
        const lines = form.getValues('lines') ?? []
        if (current === 'income' || current === 'expense') {
            linesByType.current[current] = lines
            if (lines.length > 0) lastLines.current = lines
        }
        form.setValue('type', value)
        if (value === 'transfer' || value === 'transfer_out') {
            toTransfer(form, current, value)
            onTypeChange?.(value)
            return
        }
        // The lines come back after a transfer. The categories of the lines are of the other type: each has its own.
        const base = lines.length > 0 ? lines : lastLines.current
        const own = linesByType.current[value] ?? []
        const next = base.map((line, i) => ({ ...line, category_id: own[i]?.category_id ?? null }))
        if (next.length > 0 || lines.length > 0) form.setValue('lines', next)
        // Null lets the model pick the default category of the type.
        form.setValue('default_category_id', categoryByType.current[value] ?? null)
        onTypeChange?.(value)
    }

    return handleTypeChange
}
