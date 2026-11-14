import type { FieldPath, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNegativeBalanceConfirm } from '@/components/shared'
import type { NegativeBalanceWarning } from '@/lib/transaction/negative-balance'
import { payloadErrorPath } from '@/lib/transaction/transaction-payments'
import { buildTransactionInput } from '@/lib/transaction/transaction-payload'
import { isTransferType } from '@/lib/transaction/transaction-type'
import { buildSpaceTransferInput, buildTransferInput, transferErrorPath } from '@/lib/transaction/transfer-payload'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account, SpaceTransferInput, TransactionInput } from '@/types'
import { payloadErrorMessage, transferErrorMessage } from './payload-errors'
import { useTypeChange } from './use-type-change'

interface TransactionHandlersOptions {
    form: UseFormReturn<TransactionFormValues>
    warnings: NegativeBalanceWarning[]
    dateRequired: boolean
    accounts?: Account[]
    onSubmit: (data: TransactionInput) => void
    /** A transfer to a linked space goes through the transfers API, not as a transaction. */
    onSubmitSpaceTransfer?: (data: SpaceTransferInput) => void
    onTypeChange?: (type: TransactionFormValues['type']) => void
}

export function useTransactionHandlers({
    form,
    warnings,
    dateRequired,
    accounts,
    onSubmit,
    onSubmitSpaceTransfer,
    onTypeChange,
}: TransactionHandlersOptions) {
    const { t } = useTranslation('common')
    const { confirmIfNeeded, dialog } = useNegativeBalanceConfirm<TransactionInput | SpaceTransferInput>()

    const handleSubmit = form.handleSubmit((data) => {
        // The transfers API stores a transfer to another space under a date.
        if ((dateRequired || data.type === 'transfer_out') && !data.date) {
            form.setError('date', { message: t('validation.dateRequired') })
            return
        }
        if (isTransferType(data.type)) {
            const values = { ...data, type: data.type as 'transfer' | 'transfer_out' }
            const built =
                data.type === 'transfer_out'
                    ? buildSpaceTransferInput(values, accounts ?? [])
                    : buildTransferInput(values, accounts ?? [])
            if (!built.ok) {
                form.setError(transferErrorPath(built.error) as FieldPath<TransactionFormValues>, {
                    message: transferErrorMessage(t, built.error),
                })
                return
            }
            const send = (data.type === 'transfer_out' ? onSubmitSpaceTransfer : onSubmit) as (input: unknown) => void
            confirmIfNeeded(built.input, warnings, send ?? (() => {}))
            return
        }
        if (data.type !== 'income' && data.type !== 'expense') {
            return
        }
        const built = buildTransactionInput({ ...data, type: data.type }, accounts ?? [])
        if (!built.ok) {
            form.setError(payloadErrorPath(built.error, { ...data, type: data.type }, accounts ?? []) as FieldPath<TransactionFormValues>, {
                message: payloadErrorMessage(t, built.error),
            })
            return
        }
        confirmIfNeeded(built.input, warnings, onSubmit as (input: TransactionInput | SpaceTransferInput) => void)
    })

    const handleTypeChange = useTypeChange(form, onTypeChange)

    return { handleSubmit, handleTypeChange, negativeBalanceDialog: dialog }
}
