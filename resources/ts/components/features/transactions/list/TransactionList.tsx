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
}: TransactionListProps) {
    const relativeDay = useRelativeDay()
    const groups = grouped ? groupByDateKey(transactions) : []

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
                    {group.items.map((transaction) => (
                        <TransactionRow
                            key={transaction.id}
                            transaction={transaction}
                            onDelete={onDelete}
                            onDuplicate={onDuplicate}
                            onConfirm={onConfirm}
                            onSkip={onSkip}
                            onEdit={onEdit}
                            isReadOnly={isReadOnly}
                        />
                    ))}
                </FeedGroup>
            ))}
        </div>
    )
}
