import { useState, type ReactNode } from 'react'
import { FeedEmpty, FeedGroup, FeedRowSkeleton } from '@/components/shared'
import { useRelativeDay } from '@/hooks'
import { formatCurrency } from '@/lib/currency'
import { groupByDateKey } from '@/lib/dates'
import { isApproximateOccurrence } from '@/lib/recurring-template'
import type { Account, ReconciliationDay, Transaction, DayTotal } from '@/types'
import { ReconciledLine } from './ReconciledLine'
import { ReconciliationDetailsDialog } from './ReconciliationDetailsDialog'
import { mergeReconciliationDays } from './reconciliation-days'
import { TransactionRow } from './TransactionRow'

const MAX_DAY_TOTALS = 2
const NO_RULES: ReadonlySet<number> = new Set()

function dayTotalsLabel(day: DayTotal | undefined): string | null {
    if (!day || day.totals.length === 0) {
        return null
    }

    const shown = day.totals.slice(0, MAX_DAY_TOTALS)
    const hidden = day.totals.length - shown.length
    const parts = shown.map(({ currency, amount }) => `${amount > 0 ? '+' : ''}${formatCurrency(amount, currency)}`)
    return (hidden > 0 ? [...parts, '…'] : parts).join(' · ')
}

interface TransactionListProps {
    transactions: Transaction[]
    isLoading?: boolean
    emptyTitle: string
    emptyDescription: string
    emptyAction?: ReactNode
    onCreate?: () => void
    createLabel?: string
    onDelete: (id: number) => void
    onDuplicate: (id: number) => void
    onConfirm?: (transaction: Transaction) => void
    onSkip?: (id: number) => void
    onEdit?: (transaction: Transaction) => void
    /** What a click on a row does (the view dialog); the form when left out. */
    onOpen?: (transaction: Transaction) => void
    isReadOnly?: boolean
    /** The ids of the rules whose amount is a guess: their pending occurrences show `≈` (see `useApproximateRuleIds`). */
    approximateRuleIds?: ReadonlySet<number>
    grouped?: boolean
    /** What each day netted, from the server: it covers the whole day, not just the rows loaded. */
    dayTotals?: DayTotal[]
    /** The reconciliation points by day: a line closes each day that has some, also a day with no transactions. */
    reconciliationDays?: ReconciliationDay[]
    /** The days whose points may be shown; by default the span of the loaded transactions. */
    reconciliationRange?: { start: string; end: string }
    /**
     * The direction of the list. A reconciliation is the balance at the END of its day, so its line follows the
     * day's rows when the list runs oldest first, and comes before them when it runs newest first.
     */
    sortDirection?: 'asc' | 'desc'
    /** The accounts, for the names and currencies in the details of a reconciliation. */
    accounts?: readonly Account[]
    /** Code of the base currency, in which the sum of a day's differences is shown. */
    baseCurrency?: string
    onDeleteReconciliation?: (id: number) => void
    /** Called with a day and its transactions in the new order, as listed. Move up/down menu items show when set. */
    onReorder?: (date: string, shownIds: number[]) => void
}

export function TransactionList({
    transactions,
    isLoading,
    emptyTitle,
    emptyDescription,
    emptyAction,
    onCreate,
    createLabel,
    onDelete,
    onDuplicate,
    onConfirm,
    onSkip,
    onEdit,
    onOpen,
    isReadOnly,
    approximateRuleIds = NO_RULES,
    grouped = true,
    dayTotals = [],
    reconciliationDays = [],
    reconciliationRange,
    sortDirection = 'desc',
    accounts,
    baseCurrency,
    onDeleteReconciliation,
    onReorder,
}: TransactionListProps) {
    const relativeDay = useRelativeDay()
    const [detailsDate, setDetailsDate] = useState<string | null>(null)
    const groups = grouped
        ? mergeReconciliationDays(
              groupByDateKey(transactions),
              reconciliationDays,
              sortDirection === 'desc',
              reconciliationRange,
          )
        : []
    const detailsDay = reconciliationDays.find((day) => day.date === detailsDate) ?? null
    const canReorder = (group: { date: string | null; items: Transaction[] }) =>
        !!onReorder && !isReadOnly && group.date !== null && group.items.length > 1
    const move = (group: { date: string | null; items: Transaction[] }, index: number, step: -1 | 1) => {
        const ids = group.items.map((item) => item.id)
        ;[ids[index], ids[index + step]] = [ids[index + step], ids[index]]
        onReorder?.(group.date as string, ids)
    }
    const rowProps = (transaction: Transaction) => ({
        transaction,
        onDelete,
        onDuplicate,
        onConfirm,
        onSkip,
        onEdit,
        onOpen,
        isReadOnly,
        approximate: isApproximateOccurrence(transaction, approximateRuleIds),
    })

    if (isLoading) {
        return (
            <div className="space-y-6">
                <FeedRowSkeleton showHeading />
                <FeedRowSkeleton showHeading />
            </div>
        )
    }

    if (transactions.length === 0) {
        return (
            <FeedEmpty
                title={emptyTitle}
                description={emptyDescription}
                action={emptyAction}
                onCreate={onCreate}
                createLabel={createLabel}
                isReadOnly={isReadOnly}
            />
        )
    }

    if (!grouped) {
        return (
            <div className="divide-y divide-border/60">
                {transactions.map((transaction) => (
                    <TransactionRow
                        key={transaction.id}
                        transaction={transaction}
                        onDelete={onDelete}
                        onDuplicate={onDuplicate}
                        onConfirm={onConfirm}
                        onSkip={onSkip}
                        onEdit={onEdit}
                        onOpen={onOpen}
                        isReadOnly={isReadOnly}
                        approximate={isApproximateOccurrence(transaction, approximateRuleIds)}
                        showDate
                    />
                ))}
            </div>
        )
    }

    const reconciledLine = (group: (typeof groups)[number]) =>
        group.reconciliation && (
            <ReconciledLine
                day={group.reconciliation}
                baseCurrency={baseCurrency}
                onOpenDetails={() => setDetailsDate(group.reconciliation?.date ?? null)}
            />
        )

    return (
        <div className="space-y-5">
            {groups.map((group) => (
                <div key={group.date ?? 'undated'} className="space-y-1">
                    {/* Newest first: the end of the day is the top of its group, above the heading. */}
                    {sortDirection === 'desc' && reconciledLine(group)}
                    {group.items.length > 0 && (
                        <FeedGroup
                            title={relativeDay(group.date, { weekday: true })}
                            aside={dayTotalsLabel(dayTotals.find((day) => day.date === group.date))}
                        >
                            {group.items.map((transaction, index) => (
                                <TransactionRow
                                    key={transaction.id}
                                    {...rowProps(transaction)}
                                    onMoveUp={canReorder(group) && index > 0 ? () => move(group, index, -1) : undefined}
                                    onMoveDown={
                                        canReorder(group) && index < group.items.length - 1
                                            ? () => move(group, index, 1)
                                            : undefined
                                    }
                                />
                            ))}
                            {sortDirection === 'asc' && reconciledLine(group)}
                        </FeedGroup>
                    )}
                    {group.items.length === 0 && sortDirection === 'asc' && reconciledLine(group)}
                </div>
            ))}
            <ReconciliationDetailsDialog
                day={detailsDay}
                accounts={accounts}
                dateLabel={relativeDay(detailsDate)}
                onOpenChange={(open) => !open && setDetailsDate(null)}
                onDelete={isReadOnly ? undefined : onDeleteReconciliation}
            />
        </div>
    )
}
