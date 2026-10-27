import { useRef } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { useNegativeBalanceConfirm } from '@/components/shared'
import type { NegativeBalanceWarning } from '@/lib/negative-balance'
import { buildPaymentLegs, type PaymentError } from '@/lib/payment-input'
import type { PaymentLeg } from '@/lib/split-payment'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import type { useTransferRate } from './transfer-rate'

/** The field the form shows each way a payment can fail on. */
const PAYMENT_ERROR_FIELDS: Record<PaymentError, keyof TransactionFormValues> = {
    needTwoParts: 'extra_payments',
    amountPositive: 'amount',
    unknownAccount: 'account_id',
}

function paymentErrorMessage(t: TFunction, error: PaymentError): string {
    switch (error) {
        case 'needTwoParts':
            return t('validation.paymentNeedsTwoAccounts')
        case 'amountPositive':
            return t('validation.amountPositive')
        case 'unknownAccount':
            return t('validation.selectAccount')
    }
}

/** The form values without the fields that only describe a split payment. */
function withoutPayment(data: TransactionFormValues): TransactionFormValues {
    const { extra_payments: _extra, changes: _changes, ...rest } = data
    return rest
}

interface TransactionHandlersOptions {
    form: UseFormReturn<TransactionFormValues>
    rate: ReturnType<typeof useTransferRate>
    warnings: NegativeBalanceWarning[]
    dateRequired: boolean
    isSpaceTransfer: boolean
    /** The expense is a split payment: it is saved as legs, through onSubmitPayment. */
    isSplit?: boolean
    accounts?: Account[]
    onSubmit: (data: TransactionFormValues) => void
    onSubmitPayment?: (legs: PaymentLeg[]) => void
    onTypeChange?: (type: TransactionFormValues['type']) => void
}

export function useTransactionHandlers({
    form,
    rate,
    warnings,
    dateRequired,
    isSpaceTransfer,
    isSplit,
    accounts,
    onSubmit,
    onSubmitPayment,
    onTypeChange,
}: TransactionHandlersOptions) {
    const { t } = useTranslation('common')
    const { confirmIfNeeded, dialog } = useNegativeBalanceConfirm<TransactionFormValues>()

    const handleSubmit = form.handleSubmit((data) => {
        if (dateRequired && !data.date) {
            form.setError('date', { message: t('validation.dateRequired') })
            return
        }
        if (isSplit && onSubmitPayment) {
            const built = buildPaymentLegs(data, accounts ?? [])
            if (!built.ok) {
                form.setError(PAYMENT_ERROR_FIELDS[built.error], { message: paymentErrorMessage(t, built.error) })
                return
            }
            onSubmitPayment(built.legs)
            return
        }
        rate.commit()
        confirmIfNeeded(
            {
                ...withoutPayment(data),
                to_amount: form.getValues('to_amount'),
                // UI-only helper; the server derives it from the amounts.
                exchange_rate: undefined,
            },
            warnings,
            onSubmit,
        )
    })

    // Categories are per income/expense type: keep each, so a slip onto another tab loses nothing.
    const categoryByType = useRef<Partial<Record<TransactionFormValues['type'], number>>>({})

    const handleTypeChange = (value: TransactionFormValues['type']) => {
        if ((value === 'transfer_out') !== isSpaceTransfer) {
            // The destination account belongs to another space now.
            form.setValue('to_account_id', null)
        }
        const current = form.getValues('type')
        const categoryId = form.getValues('category_id')
        if (categoryId && (current === 'income' || current === 'expense')) {
            categoryByType.current[current] = categoryId
        }
        form.setValue('type', value)
        if (value !== 'expense') {
            form.setValue('extra_payments', [])
            form.setValue('changes', [])
            form.clearErrors(['extra_payments', 'changes'])
        }
        // Null lets the party defaults pick the type's default category.
        form.setValue('category_id', categoryByType.current[value] ?? null)
        onTypeChange?.(value)
    }

    return { handleSubmit, handleTypeChange, negativeBalanceDialog: dialog }
}
