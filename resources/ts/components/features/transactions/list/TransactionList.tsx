import { type ReactNode } from 'react'
import { FeedEmpty, FeedGroup, FeedRowSkeleton } from '@/components/shared'
import { useRelativeDay } from '@/hooks'
import { groupByDateKey } from '@/lib/dates'
import { Transaction } from '@/types'
import { TransactionRow } from './TransactionRow'

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
    isReadOnly?: boolean
    grouped?: boolean
    /** Called with a day and its transactions in the new order, as listed. Move up/down menu items show when set. */
    onReorderDay?: (date: string, shownIds: number[]) => void
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
    isReadOnly,
    grouped = true,
    onReorderDay,
}: TransactionListProps) {
    const relativeDay = useRelativeDay()
    const groups = grouped ? groupByDateKey(transactions) : []
    const canReorder = (group: { date: string | null; items: Transaction[] }) =>
        !!onReorderDay && !isReadOnly && group.date !== null && group.items.length > 1
    const move = (group: { date: string | null; items: Transaction[] }, index: number, step: -1 | 1) => {
        const ids = group.items.map((item) => item.id)
        ;[ids[index], ids[index + step]] = [ids[index + step], ids[index]]
        onReorderDay?.(group.date as string, ids)
    }
    const rowProps = (transaction: Transaction) => ({
        transaction,
        onDelete,
        onDuplicate,
        onConfirm,
        onSkip,
        onEdit,
        isReadOnly,
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
                        isReadOnly={isReadOnly}
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
                    title={relativeDay(group.date)}
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
