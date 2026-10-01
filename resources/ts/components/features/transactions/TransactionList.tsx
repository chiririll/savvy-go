import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { FeedEmpty, FeedRowSkeleton } from '@/components/shared'
import { formatTransactionGroupHeading, groupByDateKey } from '@/lib/dates'
import { intlLocale } from '@/lib/i18n'
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
    const { t, i18n } = useTranslation('pages')
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
                <section key={group.date ?? 'undated'} className="min-w-0">
                    <h2 className="px-1.5 pb-1 text-sm font-semibold capitalize">
                        {formatTransactionGroupHeading(group.date, intlLocale(i18n.language), {
                            today: t('transactions.today'),
                            yesterday: t('transactions.yesterday'),
                            noDate: t('transactions.noDate'),
                        })}
                    </h2>
                    <div className="divide-y divide-border/60">
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
                    </div>
                </section>
            ))}
        </div>
    )
}
