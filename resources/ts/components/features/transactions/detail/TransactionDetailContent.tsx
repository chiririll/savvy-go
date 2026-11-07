import { AlertTriangle, Lock, Repeat } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { categoryIcons } from '@/components/shared/category-icons'
import { Markdown } from '@/components/shared/markdown/Markdown'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { useRelativeDay } from '@/hooks/use-relative-day'
import { formatCurrency } from '@/lib/currency'
import { localizeDefaultName } from '@/lib/localized-name'
import { lineCurrency } from '@/lib/transaction-money'
import { isChangePayment, orderedLines } from '@/lib/transaction-view'
import { cn } from '@/lib/utils'
import { useCurrenciesStore } from '@/stores/currencies'
import type { Transaction } from '@/types'
import { restrictionText } from '../form/restriction-text'
import { transactionIconProps } from '../list/transaction-icon'

const signed = (amount: number) => (amount > 0 ? '+' : '')

function Section({ title, children }: { title: string; children: React.ReactNode }) {
    return (
        <section className="space-y-2 border-t pt-4 first:border-t-0 first:pt-0">
            <h3 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{title}</h3>
            {children}
        </section>
    )
}

function Header({ transaction, onNavigate }: { transaction: Transaction; onNavigate: () => void }) {
    const { t } = useTranslation('pages')
    const relativeDay = useRelativeDay()
    const categories = [
        ...new Map(transaction.lines.flatMap((l) => (l.category ? [[l.category.id, l.category]] : []))).values(),
    ]
    const icon = transactionIconProps(transaction.type, categories)

    return (
        <div className="flex items-center gap-3">
            <div
                className={cn('flex size-10 shrink-0 items-center justify-center rounded-full', icon.iconClassName)}
                style={icon.iconColor ? { backgroundColor: icon.iconColor } : undefined}
            >
                {icon.icon}
            </div>
            <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-sm text-muted-foreground">
                <span>{relativeDay(transaction.date, { weekday: true })}</span>
                {transaction.status !== 'confirmed' && (
                    <Badge variant={transaction.status === 'skipped' ? 'secondary' : 'outline'}>
                        {transaction.status === 'skipped' ? t('transactions.status.skipped') : t('transactions.status.pending')}
                    </Badge>
                )}
                {transaction.recurringTransactionId !== null && (
                    <Badge variant="outline" asChild>
                        <Link to="/recurring" onClick={onNavigate}>
                            <Repeat className="size-3" />
                            {t('transactions.detail.recurring')}
                        </Link>
                    </Badge>
                )}
            </div>
        </div>
    )
}

function PaymentRow({ payment }: { payment: Transaction['payments'][number] }) {
    return (
        <li className="flex min-w-0 justify-between gap-4">
            <span className="truncate">{payment.account?.name ?? payment.accountId}</span>
            <span className={cn('shrink-0 font-medium tabular-nums', payment.amount > 0 && 'text-success')}>
                {signed(payment.amount)}
                {formatCurrency(payment.amount, payment.account?.currency)}
            </span>
        </li>
    )
}

/** The payments, then the change below them, set apart under its own label. */
function Payments({ transaction }: { transaction: Transaction }) {
    const { t } = useTranslation('pages')
    const change = transaction.payments.filter((p) => isChangePayment(transaction, p))
    const paid = transaction.payments.filter((p) => !isChangePayment(transaction, p))

    return (
        <Section title={t('transactions.detail.payments')}>
            <ul className="space-y-1.5 text-sm">
                {paid.map((payment, index) => (
                    <PaymentRow key={`${payment.accountId}-${index}`} payment={payment} />
                ))}
            </ul>
            {change.length > 0 && (
                <div className="space-y-1.5 border-t border-dashed pt-2">
                    <h4 className="text-xs text-muted-foreground">{t('transactions.detail.change')}</h4>
                    <ul className="space-y-1.5 text-sm">
                        {change.map((payment, index) => (
                            <PaymentRow key={`${payment.accountId}-${index}`} payment={payment} />
                        ))}
                    </ul>
                </div>
            )}
        </Section>
    )
}

function Lines({ transaction }: { transaction: Transaction }) {
    const { t } = useTranslation('pages')
    if (transaction.lines.length === 0) {
        return null
    }

    return (
        <Section title={t('transactions.detail.lines')}>
            <ul className="space-y-3 text-sm">
                {orderedLines(transaction.lines).map((line) => (
                    <li key={line.seq} className="space-y-1">
                        <div className="flex min-w-0 justify-between gap-4">
                            <span className="flex min-w-0 items-center gap-2">
                                <span className="shrink-0">{categoryIcons(line.category ? [line.category] : [])}</span>
                                <span className="truncate">
                                    {line.category
                                        ? localizeDefaultName(line.category.name)
                                        : t('transactions.detail.noCategory')}
                                </span>
                            </span>
                            <span className="shrink-0 font-medium tabular-nums">
                                {formatCurrency(line.amount, lineCurrency(transaction, line) ?? undefined)}
                            </span>
                        </div>
                        {line.note && <Markdown className="text-muted-foreground">{line.note}</Markdown>}
                    </li>
                ))}
            </ul>
        </Section>
    )
}

function Notices({ transaction }: { transaction: Transaction }) {
    const { t } = useTranslation()
    const base = Object.values(useCurrenciesStore((s) => s.byCode)).find((c) => c.isBase)
    const difference = transaction.difference

    return (
        <>
            {difference !== null && difference !== 0 && (
                <Alert role="note">
                    <AlertTriangle />
                    <AlertDescription>
                        {t('pages:transactions.detail.difference', {
                            amount: formatCurrency(Math.abs(difference), base),
                        })}
                    </AlertDescription>
                </Alert>
            )}
            {transaction.restrictions.length > 0 && (
                <Alert role="note">
                    <Lock />
                    <AlertDescription>
                        {transaction.restrictions.map(({ reason }) => (
                            <p key={reason}>{restrictionText(t as (key: string) => string, reason)}</p>
                        ))}
                    </AlertDescription>
                </Alert>
            )}
        </>
    )
}

/** What the detail dialog shows: header, payments, lines with their rendered notes, tags and notices. */
export function TransactionDetailContent({
    transaction,
    onNavigate,
}: {
    transaction: Transaction
    /** Called when the content links away, so the dialog can close. */
    onNavigate: () => void
}) {
    const { t } = useTranslation('pages')

    return (
        <div className="space-y-4">
            <Header transaction={transaction} onNavigate={onNavigate} />
            <Payments transaction={transaction} />
            <Lines transaction={transaction} />
            {transaction.tags.length > 0 && (
                <Section title={t('transactions.detail.tags')}>
                    <div className="flex flex-wrap gap-1.5">
                        {transaction.tags.map((tag) => (
                            <Badge key={tag.id} variant="secondary">
                                #{tag.name}
                            </Badge>
                        ))}
                    </div>
                </Section>
            )}
            <Notices transaction={transaction} />
        </div>
    )
}
