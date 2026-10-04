import { ArrowDownLeft, ArrowLeftRight, ArrowUpRight, Banknote, Check, HandCoins, SkipForward } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { formatCurrency } from '@/lib/utils'
import { parseDateKey, pendingDateClassName } from '@/lib/dates'
import { intlLocale } from '@/lib/i18n'
import { displayTransactionDescription, transactionAmountAppearance } from '@/lib/transaction-description'
import { Transaction } from '@/types'
import { SkipTransactionAlert } from './SkipTransactionAlert'

const TYPE_ICONS: Record<Transaction['type'], typeof ArrowDownLeft> = {
    income: ArrowDownLeft,
    expense: ArrowUpRight,
    transfer: ArrowLeftRight,
    debt_payment: Banknote,
    debt_collection: HandCoins,
    debt_lend: HandCoins,
    debt_borrow: Banknote,
}

interface UpcomingPendingCardProps {
    transaction: Transaction
    isReadOnly?: boolean
    onConfirm: (transaction: Transaction) => void
    onSkip: (id: number) => void
}

export function UpcomingPendingCard({
    transaction,
    isReadOnly,
    onConfirm,
    onSkip,
}: UpcomingPendingCardProps) {
    const { t } = useTranslation(['common', 'pages'])
    const Icon = TYPE_ICONS[transaction.type]
    const { sign, className } = transactionAmountAppearance(transaction.type)
    const dateLabel = transaction.date
        ? parseDateKey(transaction.date).toLocaleDateString(intlLocale(), {
            month: 'short',
            day: 'numeric',
        })
        : t('pages:transactions.noDate')

    return (
        <div className="grid w-72 shrink-0 lg:w-fullgrid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_auto_auto] gap-x-3 gap-y-2 rounded-lg border px-3 py-2.5">
            <div className="flex min-w-0 items-center gap-1.5">
                <Icon className={`size-3.5 shrink-0 ${className}`} />
                <p className="font-medium text-sm truncate">
                    {displayTransactionDescription(transaction)}
                </p>
            </div>
            <p className={`justify-self-end self-center text-xs ${pendingDateClassName(transaction.date)}`}>
                {dateLabel}
            </p>
            <p className={`row-span-2 self-start pt-0.5 font-mono text-base font-medium truncate ${className}`}>
                {transaction.isEstimated && '≈ '}{sign}{formatCurrency(transaction.amount, transaction.account.currency)}
            </p>
            {!isReadOnly && (
                <div className="col-start-2 row-start-3 flex items-end justify-end gap-1">
                    {transaction.actions.skip && (
                        <SkipTransactionAlert
                            onConfirm={() => onSkip(transaction.id)}
                            trigger={
                                <Button
                                    size="icon-sm"
                                    variant="ghost"
                                    className="size-8"
                                    aria-label={t('actions.skip')}
                                    title={t('actions.skip')}
                                >
                                    <SkipForward className="size-4" />
                                </Button>
                            }
                        />
                    )}
                    <Button
                        size="sm"
                        variant="outline"
                        title={t('actions.confirm')}
                        aria-label={t('actions.confirm')}
                        onClick={() => onConfirm(transaction)}
                    >
                        <Check className="size-4" />
                        {t('actions.confirm')}
                    </Button>
                </div>
            )}
        </div>
    )
}
