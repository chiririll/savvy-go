import { useEffect, useId, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { ResponsiveDialog } from '@/components/shared/ResponsiveDialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useNegativeBalanceConfirm } from '@/components/shared'
import { formatDateLocal, isDateInFuture, isDateOverdue, parseDateKey } from '@/lib/dates'
import { intlLocale } from '@/lib/i18n'
import { warningsForTransactionOutflow } from '@/lib/negative-balance'
import { Transaction } from '@/types'
import type { ConfirmTransactionOptions } from '@/api'

type ApplyDateChoice = 'today' | 'original' | 'other'

interface ApplyDeferredDateDialogProps {
    transaction: Transaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onConfirm: (options: ConfirmTransactionOptions & { date: string }) => void
    isSubmitting?: boolean
}

function formatStoredDate(date: string): string {
    return parseDateKey(date).toLocaleDateString(intlLocale())
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
    const { confirmIfNeeded, dialog: negativeBalanceDialog } = useNegativeBalanceConfirm<ConfirmTransactionOptions & { date: string }>()
    const groupId = useId()
    const originalDate = transaction?.date ?? null
    const originalIsUsable = Boolean(originalDate) && !isDateInFuture(originalDate)
    const today = formatDateLocal()
    const [choice, setChoice] = useState<ApplyDateChoice>('today')
    const [customDate, setCustomDate] = useState(today)
    const isEstimated = Boolean(transaction?.isEstimated)
    const crossCurrencyTransfer = Boolean(
        transaction?.toAccount
        && transaction.toAmount != null
        && transaction.account.currency?.id !== transaction.toAccount.currency?.id,
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

    const selectedDate = choice === 'today'
        ? today
        : choice === 'original'
            ? originalDate
            : customDate

    const actualAmount = Number(amount)
    const actualToAmount = Number(toAmount)
    const amountsValid = !isEstimated
        || (actualAmount > 0 && (!crossCurrencyTransfer || actualToAmount > 0))

    const canSubmit = Boolean(selectedDate)
        && amountsValid
        && !isDateInFuture(selectedDate)
        && !(choice === 'original' && !originalIsUsable)
        && !isSubmitting

    const handleSubmit = (event: FormEvent) => {
        event.preventDefault()
        if (!selectedDate || isDateInFuture(selectedDate)) {
            return
        }
        const options: ConfirmTransactionOptions & { date: string } = { date: selectedDate }
        if (isEstimated) {
            options.amount = actualAmount
            if (crossCurrencyTransfer) {
                options.to_amount = actualToAmount
            }
        }
        confirmIfNeeded(
            options,
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
                        <div className="grid gap-3 rounded-md border border-dashed p-3">
                            <p className="text-xs text-muted-foreground">{t('transactions.applyEstimatedDescription')}</p>
                            <div className="grid gap-1.5">
                                <Label htmlFor={`${groupId}-amount`}>
                                    {t('transactions.applyAmount')}
                                    {transaction?.account.currency?.symbol ? ` (${transaction.account.currency.symbol})` : ''}
                                </Label>
                                <Input
                                    id={`${groupId}-amount`}
                                    type="number"
                                    inputMode="decimal"
                                    step="any"
                                    min="0"
                                    value={amount}
                                    onChange={(event) => setAmount(event.target.value)}
                                    required
                                />
                            </div>
                            {crossCurrencyTransfer && (
                                <div className="grid gap-1.5">
                                    <Label htmlFor={`${groupId}-to-amount`}>
                                        {t('transactions.applyToAmount')}
                                        {transaction?.toAccount?.currency?.symbol ? ` (${transaction.toAccount.currency.symbol})` : ''}
                                    </Label>
                                    <Input
                                        id={`${groupId}-to-amount`}
                                        type="number"
                                        inputMode="decimal"
                                        step="any"
                                        min="0"
                                        value={toAmount}
                                        onChange={(event) => setToAmount(event.target.value)}
                                        required
                                    />
                                </div>
                            )}
                        </div>
                    )}

                    <fieldset className="grid gap-3">
                        <legend className="sr-only">{t('transactions.applyDescription')}</legend>

                        <label className="flex items-start gap-3 rounded-md border p-3 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring">
                            <input
                                type="radio"
                                name={groupId}
                                value="today"
                                checked={choice === 'today'}
                                onChange={() => setChoice('today')}
                                className="mt-1"
                            />
                            <span>
                                <span className="block text-sm font-medium">{t('transactions.applyToday')}</span>
                                <span className="text-xs text-muted-foreground">
                                    {formatStoredDate(formatDateLocal())}
                                </span>
                            </span>
                        </label>

                        {originalIsUsable ? (
                            <label className="flex items-start gap-3 rounded-md border p-3 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring">
                                <input
                                    type="radio"
                                    name={groupId}
                                    value="original"
                                    checked={choice === 'original'}
                                    onChange={() => setChoice('original')}
                                    className="mt-1"
                                />
                                <span>
                                    <span className="block text-sm font-medium">
                                        {isDateOverdue(originalDate)
                                            ? t('transactions.applyOverdue')
                                            : t('transactions.applyOriginal')}
                                    </span>
                                    <span className="text-xs text-muted-foreground">
                                        {originalDate ? formatStoredDate(originalDate) : null}
                                    </span>
                                </span>
                            </label>
                        ) : null}

                        <label className="flex items-start gap-3 rounded-md border p-3 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring">
                            <input
                                type="radio"
                                name={groupId}
                                value="other"
                                checked={choice === 'other'}
                                onChange={() => setChoice('other')}
                                className="mt-1"
                            />
                            <span className="grid min-w-0 flex-1 gap-2">
                                <span className="text-sm font-medium">{t('transactions.applyOther')}</span>
                                {choice === 'other' && (
                                    <div className="grid gap-1.5">
                                        <Label htmlFor={`${groupId}-custom`}>{tCommon('fields.date')}</Label>
                                        <Input
                                            id={`${groupId}-custom`}
                                            type="date"
                                            value={customDate}
                                            onChange={(event) => setCustomDate(event.target.value)}
                                            max={today}
                                            required
                                        />
                                    </div>
                                )}
                            </span>
                        </label>
                    </fieldset>
                </form>
            </ResponsiveDialog>
            {negativeBalanceDialog}
        </>
    )
}
