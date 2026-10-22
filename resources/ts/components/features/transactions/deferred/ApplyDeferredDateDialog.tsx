import { useEffect, useId, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { ResponsiveDialog } from '@/components/shared/dialogs/ResponsiveDialog'
import { useNegativeBalanceConfirm } from '@/components/shared'
import { formatDateLocal, isDateInFuture } from '@/lib/dates'
import { warningsForTransactionOutflow } from '@/lib/negative-balance'
import { Transaction } from '@/types'
import { ApplyDateChoices } from './ApplyDateChoices'
import { EstimatedAmountFields } from './EstimatedAmountFields'
import {
    amountsValid,
    buildApplyOptions,
    selectedApplyDate,
    type ApplyDateChoice,
    type ApplyOptions,
} from './apply-deferred'

interface ApplyDeferredDateDialogProps {
    transaction: Transaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onConfirm: (options: ApplyOptions) => void
    isSubmitting?: boolean
}

export function ApplyDeferredDateDialog({
    transaction,
    open,
    onOpenChange,
    onConfirm,
    isSubmitting,
}: ApplyDeferredDateDialogProps) {
    const { t } = useTranslation('pages')
    const { t: tCommon } = useTranslation('common')
    const { confirmIfNeeded, dialog: negativeBalanceDialog } = useNegativeBalanceConfirm<ApplyOptions>()
    const groupId = useId()
    const originalDate = transaction?.date ?? null
    const originalIsUsable = Boolean(originalDate) && !isDateInFuture(originalDate)
    const today = formatDateLocal()
    const [choice, setChoice] = useState<ApplyDateChoice>('today')
    const [customDate, setCustomDate] = useState(today)
    const isEstimated = Boolean(transaction?.isEstimated)
    const crossCurrencyTransfer = Boolean(
        transaction?.toAccount &&
        transaction.toAmount != null &&
        transaction.account.currency?.id !== transaction.toAccount.currency?.id,
    )
    const [amount, setAmount] = useState('')
    const [toAmount, setToAmount] = useState('')

    useEffect(() => {
        if (!open) {
            return
        }

        setChoice('today')
        setCustomDate(formatDateLocal())
        setAmount(transaction ? String(transaction.amount) : '')
        setToAmount(transaction?.toAmount != null ? String(transaction.toAmount) : '')
    }, [open, transaction?.id])

    const selectedDate = selectedApplyDate(choice, today, originalDate, customDate)
    const actualAmount = Number(amount)
    const actualToAmount = Number(toAmount)

    const canSubmit =
        Boolean(selectedDate) &&
        amountsValid(isEstimated, crossCurrencyTransfer, actualAmount, actualToAmount) &&
        !isDateInFuture(selectedDate) &&
        !(choice === 'original' && !originalIsUsable) &&
        !isSubmitting

    const handleSubmit = (event: FormEvent) => {
        event.preventDefault()
        if (!selectedDate || isDateInFuture(selectedDate)) {
            return
        }
        confirmIfNeeded(
            buildApplyOptions({
                date: selectedDate,
                estimated: isEstimated,
                crossCurrency: crossCurrencyTransfer,
                amount: actualAmount,
                toAmount: actualToAmount,
            }),
            transaction
                ? warningsForTransactionOutflow(isEstimated ? { ...transaction, amount: actualAmount } : transaction)
                : [],
            onConfirm,
        )
    }

    return (
        <>
            <ResponsiveDialog
                open={open}
                onOpenChange={onOpenChange}
                title={t('transactions.applyTitle')}
                description={t('transactions.applyDescription')}
                footer={
                    <>
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button type="submit" form={`${groupId}-form`} disabled={!canSubmit}>
                            {isSubmitting ? tCommon('actions.saving') : tCommon('actions.confirm')}
                        </Button>
                    </>
                }
            >
                <form id={`${groupId}-form`} onSubmit={handleSubmit} className="grid gap-4">
                    {isEstimated && (
                        <EstimatedAmountFields
                            idPrefix={groupId}
                            transaction={transaction}
                            crossCurrency={crossCurrencyTransfer}
                            amount={amount}
                            toAmount={toAmount}
                            onAmountChange={setAmount}
                            onToAmountChange={setToAmount}
                        />
                    )}
                    <ApplyDateChoices
                        groupId={groupId}
                        choice={choice}
                        originalDate={originalDate}
                        originalIsUsable={originalIsUsable}
                        customDate={customDate}
                        today={today}
                        onChoice={setChoice}
                        onCustomDate={setCustomDate}
                    />
                </form>
            </ResponsiveDialog>
            {negativeBalanceDialog}
        </>
    )
}
