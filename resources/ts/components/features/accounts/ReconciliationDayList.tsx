import { ChevronDown, TriangleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { FeedEmpty, FeedRowSkeleton } from '@/components/shared'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { formatCurrency, formatCurrencyCompact } from '@/lib/currency'
import { parseDateKey } from '@/lib/dates'
import { cn } from '@/lib/utils'
import { reconciledLabel, signedAmount, type ReconciledTone } from '@/components/features/transactions/list/reconciliation-days'
import type { Account, ReconciliationDay, ReconciliationPoint } from '@/types'

const TONE: Record<ReconciledTone, string> = {
    calm: 'text-muted-foreground',
    positive: 'text-success',
    negative: 'text-destructive',
}

const toneOf = (difference: number): ReconciledTone => (difference > 0 ? 'positive' : difference < 0 ? 'negative' : 'calm')

interface ReconciliationDayListProps {
    days: readonly ReconciliationDay[]
    accounts: readonly Account[]
    baseCurrency?: string
    isLoading?: boolean
}

/** "=": nothing differs. The words are for screen readers; they are too long for a narrow row. */
function Matches() {
    const { t } = useTranslation('pages')
    return <span aria-label={t('transactions.reconciled.matches')}>=</span>
}

function PointRow({ point, account }: { point: ReconciliationPoint; account?: Account }) {
    const { t } = useTranslation('pages')
    const money = (value: number) => formatCurrency(value, account?.currency)
    const { difference } = point
    return (
        <li className="px-3 py-2.5">
            <div className="flex items-baseline justify-between gap-3">
                <span className="min-w-0 truncate font-medium">{account?.name ?? `#${point.accountId}`}</span>
                <span className={cn('shrink-0 text-sm font-medium tabular-nums', TONE[toneOf(difference)])}>
                    {difference === 0 ? <Matches /> : signedAmount(difference, money)}
                </span>
            </div>
            <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
                {difference === 0 ? (
                    money(point.actualBalance)
                ) : (
                    <>
                        <span className="sr-only">{t('transactions.reconciled.expected')} </span>
                        {money(point.expectedBalance)}
                        <span aria-hidden="true"> → </span>
                        <span className="sr-only"> {t('transactions.reconciled.actual')} </span>
                        {money(point.actualBalance)}
                    </>
                )}
            </p>
            {point.note && <p className="mt-0.5 text-xs text-muted-foreground">{point.note}</p>}
        </li>
    )
}

function DayGroup({
    day,
    accounts,
    baseCurrency,
}: {
    day: ReconciliationDay
    accounts: readonly Account[]
    baseCurrency?: string
}) {
    const { t, i18n } = useTranslation('pages')
    const { amount, tone } = reconciledLabel(day, (value) => formatCurrencyCompact(value, baseCurrency))
    const date = parseDateKey(day.date.slice(0, 10)).toLocaleDateString(i18n.language, {
        weekday: 'short',
        day: 'numeric',
        month: 'long',
        year: 'numeric',
    })
    return (
        <Collapsible asChild>
            <li className="rounded-lg border">
                <CollapsibleTrigger className="group flex w-full items-center gap-3 px-3 py-3 text-left">
                    <ChevronDown className="size-4 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-180" />
                    <span className="min-w-0 flex-1">
                        <span className="block truncate font-medium">{date}</span>
                        <span className="block text-xs text-muted-foreground">
                            {t('accounts.reconciliations.accountsCount', { count: day.points.length })}
                        </span>
                    </span>
                    <span className={cn('flex shrink-0 items-center gap-1 text-sm font-medium tabular-nums', TONE[tone])}>
                        {amount ?? <Matches />}
                        {day.unconverted && (
                            <TriangleAlert className="size-3.5 text-warning" aria-label={t('transactions.reconciled.partial')} />
                        )}
                    </span>
                </CollapsibleTrigger>
                <CollapsibleContent>
                    <ul className="divide-y border-t">
                        {day.points.map((point) => (
                            <PointRow key={point.id} point={point} account={accounts.find((a) => a.id === point.accountId)} />
                        ))}
                    </ul>
                </CollapsibleContent>
            </li>
        </Collapsible>
    )
}

/** Every reconciliation day, newest first, each a fold that opens to its points. */
export function ReconciliationDayList({ days, accounts, baseCurrency, isLoading }: ReconciliationDayListProps) {
    const { t } = useTranslation('pages')
    if (isLoading) return <FeedRowSkeleton />
    if (days.length === 0) {
        return <FeedEmpty title={t('accounts.reconciliations.emptyTitle')} description={t('accounts.reconciliations.emptyDescription')} />
    }
    return (
        <ul className="space-y-2">
            {days.map((day) => (
                <DayGroup key={day.date} day={day} accounts={accounts} baseCurrency={baseCurrency} />
            ))}
        </ul>
    )
}
