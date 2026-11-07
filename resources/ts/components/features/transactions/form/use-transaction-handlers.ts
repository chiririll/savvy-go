import { useRef } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { useNegativeBalanceConfirm } from '@/components/shared'
import type { NegativeBalanceWarning } from '@/lib/negative-balance'
import { buildTransactionInput, type PayloadError } from '@/lib/transaction-payload'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account, TransactionInput } from '@/types'

/** The field the form shows each way the payload can fail on. */
const PAYLOAD_ERROR_FIELDS: Record<PayloadError, 'payments.0.account_id' | 'payments.0.amount' | 'change'> = {
    unknownAccount: 'payments.0.account_id',
    duplicateAccount: 'payments.0.account_id',
    amountZero: 'payments.0.amount',
    netSign: 'change',
    lineCurrency: 'payments.0.account_id',
}

function payloadErrorMessage(t: TFunction, error: PayloadError): string {
    switch (error) {
        case 'unknownAccount':
            return t('validation.selectAccount')
        case 'duplicateAccount':
            return t('validation.accountUsedTwice')
        case 'amountZero':
            return t('validation.amountPositive')
        case 'netSign':
            return t('validation.changeTooLarge')
        case 'lineCurrency':
            return t('validation.lineCurrency')
    }
}

interface TransactionHandlersOptions {
    form: UseFormReturn<TransactionFormValues>
    warnings: NegativeBalanceWarning[]
    dateRequired: boolean
    accounts?: Account[]
    onSubmit: (data: TransactionInput) => void
    onTypeChange?: (type: TransactionFormValues['type']) => void
}

export function useTransactionHandlers({
    form,
    warnings,
    dateRequired,
    accounts,
    onSubmit,
    onTypeChange,
}: TransactionHandlersOptions) {
    const { t } = useTranslation('common')
    const { confirmIfNeeded, dialog } = useNegativeBalanceConfirm<TransactionInput>()

    const handleSubmit = form.handleSubmit((data) => {
        if (dateRequired && !data.date) {
            form.setError('date', { message: t('validation.dateRequired') })
            return
        }
        const built = buildTransactionInput(data, accounts ?? [])
        if (!built.ok) {
            form.setError(PAYLOAD_ERROR_FIELDS[built.error], { message: payloadErrorMessage(t, built.error) })
            return
        }
        confirmIfNeeded(built.input, warnings, onSubmit)
    })

    // Categories are per income/expense type: keep each, so a slip onto another tab loses nothing.
    const categoryByType = useRef<Partial<Record<TransactionFormValues['type'], number>>>({})

    const handleTypeChange = (value: string) => {
        // Transfers are not in this form yet; their tabs are disabled.
        if (value !== 'income' && value !== 'expense') {
            return
        }
        const current = form.getValues('type')
        const categoryId = form.getValues('default_category_id')
        if (categoryId) {
            categoryByType.current[current] = categoryId
        }
        form.setValue('type', value)
        // The categories of the lines are of the old type too.
        const lines = form.getValues('lines') ?? []
        if (lines.some((line) => line.category_id)) {
            form.setValue('lines', lines.map((line) => ({ ...line, category_id: null })))
        }
        // Null lets the model pick the default category of the type.
        form.setValue('default_category_id', categoryByType.current[value] ?? null)
        onTypeChange?.(value)
    }

    return { handleSubmit, handleTypeChange, negativeBalanceDialog: dialog }
}
