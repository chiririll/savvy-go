import { useTranslation } from 'react-i18next'
import { Transaction } from '@/types'
import { isDateOverdue } from '@/lib/dates'
import { UpcomingPendingCard } from './UpcomingPendingCard'

interface UpcomingPendingStripProps {
    transactions: Transaction[]
    isReadOnly?: boolean
    onConfirm: (transaction: Transaction) => void
    onSkip: (id: number) => void
}

export function UpcomingPendingStrip({
    transactions,
    isReadOnly,
    onConfirm,
    onSkip,
}: UpcomingPendingStripProps) {
    const { t } = useTranslation('pages')

    if (transactions.length === 0) {
        return null
    }

    const overdue = transactions.filter((transaction) => isDateOverdue(transaction.date))
    const upcoming = transactions.filter((transaction) => !isDateOverdue(transaction.date))
    const sections = [
        { key: 'overdue', title: t('transactions.overdueTitle'), items: overdue },
        { key: 'upcoming', title: t('transactions.soonTitle'), items: upcoming },
    ].filter((section) => section.items.length > 0)

    const renderCards = (items: Transaction[]) =>
        items.map((transaction) => (
            <UpcomingPendingCard
                key={transaction.id}
                transaction={transaction}
                isReadOnly={isReadOnly}
                onConfirm={onConfirm}
                onSkip={onSkip}
            />
        ))

    return (
        <>
            <div className="min-w-0 lg:hidden">
                <p className="text-sm font-medium mb-2">{t('transactions.upcomingTitle')}</p>
                <div className="overflow-x-auto overscroll-x-contain pb-3">
                    <div className="flex w-max gap-2">{renderCards(transactions)}</div>
                </div>
            </div>

            <div className="hidden min-w-0 space-y-3 lg:sticky lg:top-24 lg:block lg:max-h-[calc(100vh-10rem)] lg:self-start lg:overflow-y-auto">
                {sections.map((section) => (
                    <div key={section.key} className="min-w-0">
                        <p className="text-sm font-medium mb-2">{section.title}</p>
                        <div className="flex flex-col gap-2">{renderCards(section.items)}</div>
                    </div>
                ))}
            </div>
        </>
    )
}
