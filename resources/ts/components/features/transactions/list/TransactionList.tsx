import { type ReactNode } from 'react'
import { FeedEmpty, FeedGroup, FeedRowSkeleton } from '@/components/shared'
import { useRelativeDay } from '@/hooks'
import { formatCurrency } from '@/lib/currency'
import { groupByDateKey } from '@/lib/dates'
import { isApproximateOccurrence } from '@/lib/recurring-template'
import { Transaction, type DayTotal } from '@/types'
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
    onReorder,
}: TransactionListProps) {
    const relativeDay = useRelativeDay()
    const groups = grouped ? groupByDateKey(transactions) : []
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

    return (
        <div className="space-y-5">
            {groups.map((group) => (
                <FeedGroup
                    key={group.date ?? 'undated'}
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
                </FeedGroup>
            ))}
        </div>
    )
}
