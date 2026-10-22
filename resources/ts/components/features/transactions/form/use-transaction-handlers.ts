import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNegativeBalanceConfirm } from '@/components/shared'
import type { NegativeBalanceWarning } from '@/lib/negative-balance'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { useTransferRate } from './transfer-rate'

interface TransactionHandlersOptions {
    form: UseFormReturn<TransactionFormValues>
    rate: ReturnType<typeof useTransferRate>
    warnings: NegativeBalanceWarning[]
    dateRequired: boolean
    isSpaceTransfer: boolean
    onSubmit: (data: TransactionFormValues) => void
    onTypeChange?: (type: TransactionFormValues['type']) => void
}

export function useTransactionHandlers({
    form,
    rate,
    warnings,
    dateRequired,
    isSpaceTransfer,
    onSubmit,
    onTypeChange,
}: TransactionHandlersOptions) {
    const { t } = useTranslation('common')
    const { confirmIfNeeded, dialog } = useNegativeBalanceConfirm<TransactionFormValues>()

    const handleSubmit = form.handleSubmit((data) => {
        if (dateRequired && !data.date) {
            form.setError('date', { message: t('validation.dateRequired') })
            return
        }
        rate.commit()
        confirmIfNeeded(
            {
                ...data,
                to_amount: form.getValues('to_amount'),
                // UI-only helper; the server derives it from the amounts.
                exchange_rate: undefined,
            },
            warnings,
            onSubmit,
        )
    })

    const handleTypeChange = (value: TransactionFormValues['type']) => {
        if ((value === 'transfer_out') !== isSpaceTransfer) {
            // The destination account belongs to another space now.
            form.setValue('to_account_id', null)
        }
        form.setValue('type', value)
        onTypeChange?.(value)
    }

    return { handleSubmit, handleTypeChange, negativeBalanceDialog: dialog }
}
