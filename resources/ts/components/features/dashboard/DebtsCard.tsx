import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight, Banknote, HandCoins, Plus, TrendingDown, TrendingUp } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { formatCurrency, cn } from '@/lib/utils'
import type { Debt, DebtSummary } from '@/types'

function SummaryTiles({ summary }: { summary: DebtSummary }) {
    const { t } = useTranslation('pages')
    const net = summary.netDebt >= 0

    return (
        <div className="grid min-w-0 gap-4 sm:grid-cols-3">
            <div className="flex items-center gap-3 p-3 rounded-lg bg-red-50 dark:bg-red-950/20">
                <div className="p-2 rounded-lg bg-red-100 dark:bg-red-900/30">
                    <TrendingDown className="size-4 text-red-600" />
                </div>
                <div>
                    <p className="text-xs text-muted-foreground">{t('debts.types.i_owe')}</p>
                    <p className="font-mono font-semibold text-red-600 break-all">
                        {formatCurrency(summary.totalIOwe, summary.currency)}
                    </p>
                </div>
            </div>
            <div className="flex items-center gap-3 p-3 rounded-lg bg-green-50 dark:bg-green-950/20">
                <div className="p-2 rounded-lg bg-green-100 dark:bg-green-900/30">
                    <TrendingUp className="size-4 text-green-600" />
                </div>
                <div>
                    <p className="text-xs text-muted-foreground">{t('debts.types.owed_to_me')}</p>
                    <p className="font-mono font-semibold text-green-600 break-all">
                        {formatCurrency(summary.totalOwedToMe, summary.currency)}
                    </p>
                </div>
            </div>
            <div
                className={cn(
                    'flex items-center gap-3 p-3 rounded-lg',
                    net ? 'bg-green-50 dark:bg-green-950/20' : 'bg-red-50 dark:bg-red-950/20',
                )}
            >
                <div
                    className={cn(
                        'p-2 rounded-lg',
                        net ? 'bg-green-100 dark:bg-green-900/30' : 'bg-red-100 dark:bg-red-900/30',
                    )}
                >
                    {net ? (
                        <HandCoins className="size-4 text-green-600" />
                    ) : (
                        <Banknote className="size-4 text-red-600" />
                    )}
                </div>
                <div>
                    <p className="text-xs text-muted-foreground">{t('debts.netPosition')}</p>
                    <p className={`font-mono font-semibold break-all ${net ? 'text-green-600' : 'text-red-600'}`}>
                        {formatCurrency(Math.abs(summary.netDebt), summary.currency)}
                    </p>
                </div>
            </div>
        </div>
    )
}

function DebtTile({ debt }: { debt: Debt }) {
    const { t } = useTranslation('pages')
    const iOwe = debt.debtType === 'i_owe'

    return (
        <Link to={`/debts?edit=${debt.id}`} className="block p-3 rounded-lg border hover:bg-muted/50 transition-colors">
            <div className="flex items-center gap-2 mb-2">
                {iOwe ? <Banknote className="size-4 text-red-600" /> : <HandCoins className="size-4 text-green-600" />}
                <p className="font-medium text-sm truncate">{debt.name}</p>
            </div>
            <Progress value={debt.paymentProgress} aria-label={debt.name} className="h-1.5 mb-2" />
            <div className="flex justify-between text-xs">
                <span className="text-muted-foreground">
                    {t('dashboard.percentPaid', { percent: debt.paymentProgress.toFixed(0) })}
                </span>
                <span className={iOwe ? 'text-red-600' : 'text-green-600'}>
                    {formatCurrency(debt.currentBalance, debt.currency)}
                </span>
            </div>
            {debt.counterparty && (
                <p className="text-xs text-muted-foreground mt-1 truncate">
                    {iOwe
                        ? t('dashboard.toCounterparty', { name: debt.counterparty })
                        : t('dashboard.fromCounterparty', { name: debt.counterparty })}
                </p>
            )}
        </Link>
    )
}

export function DebtsCard({ summary, debts }: { summary?: DebtSummary; debts: Debt[] }) {
    const { t } = useTranslation('pages')
    const { t: tCommon } = useTranslation('common')

    return (
        <Card className="min-w-0">
            <CardHeader className="flex min-w-0 flex-row items-center justify-between gap-2">
                <CardTitle className="flex min-w-0 items-center gap-2">
                    <HandCoins className="size-5 shrink-0" />
                    {t('dashboard.debts')}
                </CardTitle>
                <Button variant="ghost" size="sm" asChild>
                    <Link to="/debts">
                        {tCommon('actions.viewAll')}
                        <ArrowRight className="ml-1 size-4" />
                    </Link>
                </Button>
            </CardHeader>
            <CardContent className="min-w-0">
                {summary && (summary.totalIOwe > 0 || summary.totalOwedToMe > 0) ? (
                    <div className="min-w-0 space-y-4">
                        <SummaryTiles summary={summary} />
                        {debts.length > 0 && (
                            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                                {debts.map((debt) => (
                                    <DebtTile key={debt.id} debt={debt} />
                                ))}
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="text-center py-8">
                        <HandCoins className="size-12 mx-auto text-muted-foreground/50 mb-3" />
                        <p className="text-muted-foreground mb-3">{t('dashboard.noDebts')}</p>
                        <Button asChild size="sm">
                            <Link to="/debts?create=1">
                                <Plus className="size-4 mr-1" />
                                {t('dashboard.addDebt')}
                            </Link>
                        </Button>
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
