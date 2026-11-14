import { useEffect, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/ui/form-resolver'
import { useFormValuesChange } from '@/hooks'
import { formatDateLocal } from '@/lib/utils'
import { getTransactionSchema, TransactionFormValues } from '@/schemas/transactions'

/** What an opener can ask for. */
export type TransactionFormDefaults = Partial<TransactionFormValues>

interface TransactionFormOptions {
    defaultValues?: TransactionFormDefaults
    onValuesChange?: (data: TransactionFormValues) => void
    originalAffectsBalance?: boolean
    open: boolean
}

/**
 * A new income or expense starts with one line that takes all of the payment: see `TransactionLines`. The amount is
 * set here, not by an effect: the reset that applies these defaults would wipe an amount an effect had set.
 */
function startingLines(type: string, defaultValues?: TransactionFormDefaults): TransactionFormValues['lines'] {
    if (type !== 'income' && type !== 'expense') {
        return []
    }
    const payments = defaultValues?.payments
    const amount = payments?.length === 1 && !defaultValues?.change?.length ? payments[0].amount : ''
    return [{ category_id: defaultValues?.default_category_id ?? null, currency_id: null, amount: amount as number, note: defaultValues?.default_note ?? '' }]
}

export function toFormDefaults(defaultValues?: TransactionFormDefaults): TransactionFormValues {
    const type = defaultValues?.type ?? 'expense'
    const startLines = defaultValues?.lines ?? startingLines(type, defaultValues)
    return {
        type,
        description: defaultValues?.description ?? '',
        date: defaultValues?.date !== undefined ? (defaultValues.date ?? '') : formatDateLocal(),
        tag_ids: defaultValues?.tag_ids ?? [],
        payments: defaultValues?.payments ?? [{ account_id: 0, amount: '' as unknown as number }],
        change: defaultValues?.change ?? [],
        lines: startLines,
        default_category_id: defaultValues?.default_category_id ?? null,
        default_note: defaultValues?.default_note ?? null,
        to_account_id: defaultValues?.to_account_id ?? null,
        to_amount: defaultValues?.to_amount ?? null,
        exchange_rate: defaultValues?.exchange_rate ?? null,
        to_space_id: defaultValues?.to_space_id ?? null,
        fee: defaultValues?.fee ?? null,
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
