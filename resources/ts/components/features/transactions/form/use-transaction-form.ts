import { useEffect, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { useFormValuesChange } from '@/hooks'
import { formatDateLocal } from '@/lib/utils'
import { getTransactionSchema, TransactionFormValues } from '@/schemas/transactions'
import { resolveOpeningTransferRate } from './transfer-rate'

interface TransactionFormOptions {
    defaultValues?: Partial<TransactionFormValues>
    onValuesChange?: (data: TransactionFormValues) => void
    originalAffectsBalance?: boolean
    open: boolean
}

// The form, its defaults and the schema options the amount rounding feeds back into.
export function useTransactionForm({
    defaultValues,
    onValuesChange,
    originalAffectsBalance,
    open,
}: TransactionFormOptions) {
    const formDefaults = useMemo(
        () => ({
            type: defaultValues?.type ?? ('expense' as const),
            account_id: defaultValues?.account_id ?? 0,
            to_account_id: defaultValues?.to_account_id ?? null,
            to_space_id: defaultValues?.to_space_id ?? null,
            category_id: defaultValues?.category_id ?? null,
            amount: defaultValues?.amount ?? 0,
            to_amount: defaultValues?.to_amount ?? null,
            is_estimated: defaultValues?.is_estimated ?? false,
            exchange_rate: resolveOpeningTransferRate(defaultValues),
            description: defaultValues?.description ?? '',
            date: defaultValues?.date !== undefined ? (defaultValues.date ?? '') : formatDateLocal(),
            items: defaultValues?.items ?? [],
            tag_ids: defaultValues?.tag_ids ?? [],
            payment_mode: defaultValues?.payment_mode ?? ('single' as const),
            extra_payments: defaultValues?.extra_payments ?? [],
            handed_over: defaultValues?.handed_over ?? null,
            change_account_id: defaultValues?.change_account_id ?? null,
            change_amount: defaultValues?.change_amount ?? null,
        }),
        [defaultValues],
    )

    const schemaOptionsRef = useRef({
        rejectFutureDate: Boolean(originalAffectsBalance),
        currencyDecimals: 2,
    })

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

    useFormValuesChange(form, onValuesChange ? (data) => onValuesChange({ ...data, exchange_rate: null }) : undefined)

    return { form, schemaOptionsRef }
}
