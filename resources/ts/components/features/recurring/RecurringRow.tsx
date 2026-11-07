import { useTranslation } from 'react-i18next'
import { FeedRow, FeedStatusBadge, RowActions, type FeedGroupKey } from '@/components/shared'
import { transactionIconProps } from '@/components/features/transactions/list/transaction-icon'
import { TransactionAmount } from '@/components/features/transactions/list/TransactionAmount'
import { useAccounts, useCategories, useRelativeDay } from '@/hooks'
import { pendingDateClassName } from '@/lib/dates'
import i18n from '@/lib/i18n'
import { frequencyLabel } from '@/lib/labels'
import { isApproximateRule, templateAsTransaction } from '@/lib/recurring-template'
import {
    displayTransactionDescription,
    transactionAmountAppearance,
    transactionSubtitle,
} from '@/lib/transaction-description'
import { rowAmount, transferSides } from '@/lib/transaction-money'
import { cn } from '@/lib/utils'
import type { Category, RecurringFrequency, RecurringTransaction } from '@/types'

const FREQUENCY_ORDER: RecurringFrequency[] = ['daily', 'weekly', 'monthly', 'yearly']

/** Sections of the recurring list: one per schedule, shortest first. */
export function recurringGroup(recurring: RecurringTransaction): FeedGroupKey {
    const { frequency, interval } = recurring.schedule
    const label = frequencyLabel(i18n.t, frequency)
    return {
        key: `${frequency}:${interval}`,
        title:
            interval === 1
                ? label
                : i18n.t('pages:recurring.everyInterval', { count: interval, frequency: label.toLowerCase() }),
    }
}

/** Schedule order, then active before paused, then the next run (a rule with none last). */
export function compareRecurring(a: RecurringTransaction, b: RecurringTransaction): number {
    return (
        FREQUENCY_ORDER.indexOf(a.schedule.frequency) - FREQUENCY_ORDER.indexOf(b.schedule.frequency) ||
        a.schedule.interval - b.schedule.interval ||
        Number(b.isActive) - Number(a.isActive) ||
        (a.nextRunDate ?? '9999-12-31').localeCompare(b.nextRunDate ?? '9999-12-31')
    )
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
    const { data: accounts } = useAccounts()
    const { data: categories } = useCategories()
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const { template, nextRunDate } = recurring
    const { sign, className } = transactionAmountAppearance(template.type)

    // What the rule makes, read as the transaction it will be. Until the accounts load (or if one is gone) the row
    // has only the title the rule was saved with.
    const made = accounts ? templateAsTransaction(recurring, accounts, categories) : null
    const shape = made ?? { type: template.type, description: template.description ?? null, payments: [], lines: [] }
    const amount = made ? (template.type === 'transfer' ? transferSides(made).sent : rowAmount(made)) : null
    const rowCategories = [
        ...new Map(shape.lines.flatMap((l) => (l.category ? [[l.category.id, l.category] as [number, Category]] : []))).values(),
    ]

    return (
        <FeedRow
            {...transactionIconProps(template.type, rowCategories)}
            title={displayTransactionDescription(shape)}
            badge={recurring.error ? <FeedStatusBadge variant="outline">{t('recurring.stopped')}</FeedStatusBadge> : undefined}
            subtitle={recurring.error ?? transactionSubtitle(shape)}
            amount={<TransactionAmount amount={amount} sign={sign} approximate={isApproximateRule(template)} />}
            amountClassName={className}
            extraAmount={
                nextRunDate ? (
                    <span className={cn('font-sans', recurring.isActive && pendingDateClassName(nextRunDate.slice(0, 10)))}>
                        {relativeDay(nextRunDate)}
                    </span>
                ) : undefined
            }
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
