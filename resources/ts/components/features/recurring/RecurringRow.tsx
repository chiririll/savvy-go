import { useTranslation } from 'react-i18next'
import { FeedRow, RowActions, type FeedGroupKey } from '@/components/shared'
import { transactionIconProps } from '@/components/features/transactions/transaction-icon'
import { useRelativeDay } from '@/hooks'
import { pendingDateClassName } from '@/lib/dates'
import i18n from '@/lib/i18n'
import { displayTransactionDescription, transactionAmountAppearance, transactionSubtitle } from '@/lib/transaction-description'
import { cn, formatCurrency } from '@/lib/utils'
import { RecurringFrequency, RecurringTransaction } from '@/types'

const FREQUENCY_ORDER: RecurringFrequency[] = ['daily', 'weekly', 'monthly', 'yearly']

/** Sections of the recurring list: one per schedule, shortest first. */
export function recurringGroup(recurring: RecurringTransaction): FeedGroupKey {
    const frequency = i18n.t(`forms:recurring.frequencies.${recurring.frequency}`)
    return {
        key: `${recurring.frequency}:${recurring.interval}`,
        title: recurring.interval === 1
            ? frequency
            : i18n.t('pages:recurring.everyInterval', { count: recurring.interval, frequency: frequency.toLowerCase() }),
    }
}

/** Schedule order, then active before paused, then the next run. */
export function compareRecurring(a: RecurringTransaction, b: RecurringTransaction): number {
    return FREQUENCY_ORDER.indexOf(a.frequency) - FREQUENCY_ORDER.indexOf(b.frequency)
        || a.interval - b.interval
        || Number(b.isActive) - Number(a.isActive)
        || a.nextRunDate.localeCompare(b.nextRunDate)
}

interface RecurringRowProps {
    recurring: RecurringTransaction
    onEdit?: (recurring: RecurringTransaction) => void
    onDelete?: (id: number) => void
    isReadOnly?: boolean
}

export function RecurringRow({ recurring, onEdit, onDelete, isReadOnly }: RecurringRowProps) {
    const { t } = useTranslation('pages')
    const relativeDay = useRelativeDay()
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const { sign, className } = transactionAmountAppearance(recurring.type)

    return (
        <FeedRow
            {...transactionIconProps(recurring.type, recurring.category)}
            title={displayTransactionDescription(recurring)}
            subtitle={transactionSubtitle(recurring)}
            amount={`${recurring.isEstimated ? '≈ ' : ''}${sign}${formatCurrency(recurring.amount, recurring.account.currency)}`}
            amountClassName={className}
            extraAmount={(
                <span className={cn('font-sans', recurring.isActive && pendingDateClassName(recurring.nextRunDate.slice(0, 10)))}>
                    {relativeDay(recurring.nextRunDate)}
                </span>
            )}
            muted={!recurring.isActive}
            onOpen={canEdit ? () => onEdit(recurring) : undefined}
            hasActions={canEdit || canDelete}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(recurring) : undefined}
                    onDelete={canDelete ? () => onDelete(recurring.id) : undefined}
                    deleteTitle={t('recurring.deleteTitle')}
                    deleteDescription={t('recurring.deleteDescription')}
                />
            )}
        />
    )
}
