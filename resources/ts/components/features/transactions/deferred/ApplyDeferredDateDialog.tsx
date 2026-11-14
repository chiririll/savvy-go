import { useEffect, useId, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { ResponsiveDialog } from '@/components/shared/dialogs/ResponsiveDialog'
import { useNegativeBalanceConfirm } from '@/components/shared'
import { formatDateLocal, isDateInFuture } from '@/lib/format/dates'
import { Transaction } from '@/types'
import { ApplyDateChoices } from './ApplyDateChoices'
import { OccurrenceAmountFields } from './OccurrenceAmountFields'
import {
    amountsValid,
    applyWarnings,
    buildApplyOptions,
    initialAmounts,
    isOccurrence,
    selectedApplyDate,
    type AmountDraft,
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
    // The open occurrence of a recurring rule has expected amounts; the others are posted as they are.
    const askAmounts = transaction !== null && isOccurrence(transaction)
    const [draft, setDraft] = useState<AmountDraft>({ payments: [], lines: [] })

    useEffect(() => {
        if (!open) {
            return
        }

        setChoice('today')
        setCustomDate(formatDateLocal())
        setDraft(transaction ? initialAmounts(transaction) : { payments: [], lines: [] })
    }, [open, transaction?.id])

    const selectedDate = selectedApplyDate(choice, today, originalDate, customDate)

    const canSubmit =
        Boolean(selectedDate) &&
        (!transaction || amountsValid(transaction, draft)) &&
        !isDateInFuture(selectedDate) &&
        !(choice === 'original' && !originalIsUsable) &&
        !isSubmitting

    const handleSubmit = (event: FormEvent) => {
        event.preventDefault()
        if (!selectedDate || isDateInFuture(selectedDate) || !transaction) {
            return
        }
        confirmIfNeeded(
            buildApplyOptions(transaction, selectedDate, draft),
            applyWarnings(transaction, draft),
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
                    {askAmounts && transaction && (
                        <OccurrenceAmountFields
                            idPrefix={groupId}
                            transaction={transaction}
                            draft={draft}
                            onChange={setDraft}
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
