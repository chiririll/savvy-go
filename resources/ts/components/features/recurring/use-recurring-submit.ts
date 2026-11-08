import type { FieldPath, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { buildRecurringInput } from '@/lib/recurring-template'
import { payloadErrorPath } from '@/lib/transaction-payments'
import { transferErrorPath } from '@/lib/transfer-payload'
import type { RecurringFormValues } from '@/schemas/recurring'
import type { Account, RecurringInput } from '@/types'
import { payloadErrorMessage, transferErrorMessage } from '@/components/features/transactions/form/payload-errors'

/**
 * The submit of the recurring form: the rule the API takes for what the form holds, or the message of what the
 * API would refuse, put on the field it belongs to.
 */
export function useRecurringSubmit(
    form: UseFormReturn<RecurringFormValues>,
    accounts: Account[] | undefined,
    onSubmit: (data: RecurringInput) => void,
) {
    const { t } = useTranslation('common')

    return form.handleSubmit((values) => {
        const built = buildRecurringInput(values, accounts ?? [])
        if (built.ok) {
            onSubmit(built.input)
            return
        }
        if (built.source === 'transfer') {
            form.setError(transferErrorPath(built.error) as FieldPath<RecurringFormValues>, {
                message: transferErrorMessage(t, built.error),
            })
            return
        }
        const type = values.type === 'income' ? 'income' : 'expense'
        const path = payloadErrorPath(built.error, { ...values, type }, accounts ?? [])
        form.setError(path as FieldPath<RecurringFormValues>, { message: payloadErrorMessage(t, built.error) })
    })
}
