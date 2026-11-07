import { useEffect, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { useFormValuesChange } from '@/hooks'
import { formatDateLocal } from '@/lib/utils'
import { getTransactionSchema, TransactionFormValues } from '@/schemas/transactions'

/** What an opener can ask for. A kind the form does not take yet (a transfer) opens an expense. */
export type TransactionFormDefaults = Omit<Partial<TransactionFormValues>, 'type'> & {
    type?: TransactionFormValues['type'] | 'transfer' | 'transfer_out'
}

interface TransactionFormOptions {
    defaultValues?: TransactionFormDefaults
    onValuesChange?: (data: TransactionFormValues) => void
    originalAffectsBalance?: boolean
    open: boolean
}

export function toFormDefaults(defaultValues?: TransactionFormDefaults): TransactionFormValues {
    return {
        type: defaultValues?.type === 'income' ? 'income' : 'expense',
        description: defaultValues?.description ?? '',
        date: defaultValues?.date !== undefined ? (defaultValues.date ?? '') : formatDateLocal(),
        tag_ids: defaultValues?.tag_ids ?? [],
        payments: defaultValues?.payments ?? [{ account_id: 0, amount: '' as unknown as number }],
        change: defaultValues?.change ?? [],
        lines: defaultValues?.lines ?? [],
        default_category_id: defaultValues?.default_category_id ?? null,
        default_note: defaultValues?.default_note ?? null,
    }
}

// The form and its defaults.
export function useTransactionForm({ defaultValues, onValuesChange, originalAffectsBalance, open }: TransactionFormOptions) {
    const formDefaults = useMemo(() => toFormDefaults(defaultValues), [defaultValues])

    const schemaOptionsRef = useRef({ rejectFutureDate: Boolean(originalAffectsBalance) })
    schemaOptionsRef.current = { rejectFutureDate: Boolean(originalAffectsBalance) }

    const form = useForm<TransactionFormValues>({
        resolver: (values, context, options) =>
            schemaResolver<TransactionFormValues>(getTransactionSchema(schemaOptionsRef.current))(
                values,
                context,
                options,
            ),
        defaultValues: formDefaults,
    })

    // Runs before the effects of the caller, so the defaults they fill in survive it.
    const formDefaultsRef = useRef(formDefaults)
    formDefaultsRef.current = formDefaults
    useEffect(() => {
        if (open) {
            form.reset(formDefaultsRef.current)
        }
    }, [open, form])

    useFormValuesChange(form, onValuesChange)

    return { form }
}
