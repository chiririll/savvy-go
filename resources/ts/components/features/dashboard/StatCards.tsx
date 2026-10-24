import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowDownLeft, ArrowUpRight, Clock, TrendingDown, TrendingUp, Wallet } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { cn, formatCurrency } from '@/lib/utils'
import type { usePendingSummary, useTotalBalance } from '@/hooks'

type PendingSummary = NonNullable<ReturnType<typeof usePendingSummary>['data']>
type Currency = NonNullable<ReturnType<typeof useTotalBalance>['data']>['currency']

function StatCard({ title, icon, children }: { title: string; icon: ReactNode; children: ReactNode }) {
    return (
        <Card className="min-w-0">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
                {icon}
            </CardHeader>
            <CardContent className="min-w-0">{children}</CardContent>
        </Card>
    )
}

/** The arrow follows the sign of the change; the colour follows whether that direction is good. */
function ChangeBadge({ change, lowerIsBetter = false }: { change: number | null; lowerIsBetter?: boolean }) {
    if (change === null) return null
    const good = lowerIsBetter ? change <= 0 : change >= 0

    return (
        <div className={cn('flex items-center gap-1 text-xs mt-1', good ? 'text-success' : 'text-destructive')}>
            {change >= 0 ? <TrendingUp className="size-3" /> : <TrendingDown className="size-3" />}
            <span>{Math.abs(change).toFixed(1)}%</span>
        </div>
    )
}

interface DashboardStatsProps {
    currency?: Currency
    totalBalance: number
    periodIncome: number
    periodExpense: number
    balanceChange: number | null
    incomeChange: number | null
    expenseChange: number | null
    pendingSummary?: PendingSummary
}

export function DashboardStats({
    currency,
    totalBalance,
    periodIncome,
    periodExpense,
    balanceChange,
    incomeChange,
    expenseChange,
    pendingSummary,
}: DashboardStatsProps) {
    const { t } = useTranslation('pages')
    const pendingBalance = pendingSummary?.balance ?? 0
    const pendingCount = pendingSummary?.transactionsCount ?? 0

    return (
        <div className="grid min-w-0 gap-4 md:grid-cols-2 xl:grid-cols-4">
            <StatCard title={t('dashboard.totalBalance')} icon={<Wallet className="size-4 text-muted-foreground" />}>
                <div className="text-2xl font-bold font-mono break-all">{formatCurrency(totalBalance, currency)}</div>
                <ChangeBadge change={balanceChange} />
            </StatCard>

            <StatCard title={t('dashboard.income')} icon={<ArrowDownLeft className="size-4 text-success" />}>
                <div className="text-2xl font-bold font-mono text-success break-all">
                    +{formatCurrency(periodIncome, currency)}
                </div>
                <ChangeBadge change={incomeChange} />
            </StatCard>

            <StatCard title={t('dashboard.expenses')} icon={<ArrowUpRight className="size-4 text-destructive" />}>
                <div className="text-2xl font-bold font-mono text-destructive break-all">
                    -{formatCurrency(periodExpense, currency)}
                </div>
                <ChangeBadge change={expenseChange} lowerIsBetter />
            </StatCard>

            <StatCard title={t('dashboard.pending')} icon={<Clock className="size-4 text-muted-foreground" />}>
                <Link to="/transactions?status=pending" className="block min-w-0">
                    <div
                        className={cn(
                            'text-2xl font-bold font-mono break-all',
                            pendingBalance > 0 && 'text-success',
                            pendingBalance < 0 && 'text-destructive',
                        )}
                    >
                        {formatCurrency(pendingBalance, pendingSummary?.currency ?? currency)}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                        {pendingCount > 0
                            ? t('dashboard.pendingCount', { count: pendingCount })
                            : t('dashboard.pendingEmpty')}
                    </p>
                </Link>
            </StatCard>
        </div>
    )
}
