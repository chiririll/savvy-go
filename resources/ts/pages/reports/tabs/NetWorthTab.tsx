import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn, formatCurrency } from '@/lib/utils'
import { TrendingUp, TrendingDown, Wallet } from 'lucide-react'
import { NetWorthChart } from '../components/NetWorthChart'
import { AccountsStructureChart } from '../components/AccountsStructureChart'
import { useNetWorth } from '@/hooks'
import type { ReportFilters } from '../types'

// Shrinks the headline amount with its length so it fits narrow screens (≈0.6em per glyph).
function netWorthFontSize(text: string) {
    return `min(3.75rem, ${(84 / Math.max(text.length * 0.6, 1)).toFixed(2)}vw)`
}

interface NetWorthTabProps {
    filters: ReportFilters
}

export function NetWorthTab({ filters }: NetWorthTabProps) {
    const { t } = useTranslation('pages')
    const { data, isLoading } = useNetWorth(filters)

    const isPositive = (data?.change ?? 0) >= 0
    const netWorthText = data ? formatCurrency(data.current, data.currency) : ''

    return (
        <div className="space-y-6">
            {/* Block 1 — Current Net Worth */}
            <Card>
                <CardContent className="pt-6">
                    {isLoading ? (
                        <div className="flex flex-col items-center text-center py-4">
                            <Skeleton className="size-16 rounded-2xl mb-4" />
                            <Skeleton className="h-4 w-32 mb-2" />
                            <Skeleton className="h-16 w-48 mb-4" />
                            <Skeleton className="h-8 w-64" />
                        </div>
                    ) : data ? (
                        <div className="flex flex-col items-center text-center py-4">
                            {/* Icon */}
                            <div className="flex items-center justify-center size-16 rounded-2xl bg-gradient-to-br from-blue-500 to-blue-600 text-white mb-4 shadow-lg">
                                <Wallet className="size-8" />
                            </div>

                            {/* Label */}
                            <p className="text-sm text-muted-foreground font-medium mb-2">
                                {t('reports.metrics.currentNetWorth')}
                            </p>

                            {/* Main value */}
                            <p
                                className={cn(
                                    'max-w-full font-bold tracking-tight mb-4 whitespace-nowrap',
                                    data.current >= 0 ? 'text-blue-600' : 'text-red-600'
                                )}
                                style={{ fontSize: netWorthFontSize(netWorthText) }}
                            >
                                {netWorthText}
                            </p>

                            {/* Change indicators */}
                            {filters.compareWith !== 'none' && data.previous !== null && (
                                <div className="flex flex-wrap items-center justify-center gap-2 sm:gap-4">
                                    {/* Percentage change */}
                                    <div className={cn(
                                        'flex items-center gap-1.5 px-4 py-2 rounded-full text-sm font-medium',
                                        isPositive ? 'bg-green-100 text-green-700 dark:bg-green-950/50 dark:text-green-300' : 'bg-red-100 text-red-700 dark:bg-red-950/50 dark:text-red-300'
                                    )}>
                                        {isPositive ? (
                                            <TrendingUp className="size-4" />
                                        ) : (
                                            <TrendingDown className="size-4" />
                                        )}
                                        {isPositive ? '+' : ''}{data.changePercent.toFixed(1)}%
                                    </div>

                                    {/* Absolute change */}
                                    <div className={cn(
                                        'flex items-center gap-1.5 px-4 py-2 rounded-full text-sm font-medium',
                                        isPositive ? 'bg-green-100 text-green-700 dark:bg-green-950/50 dark:text-green-300' : 'bg-red-100 text-red-700 dark:bg-red-950/50 dark:text-red-300'
                                    )}>
                                        {isPositive ? '+' : ''}{formatCurrency(data.change, data.currency)}
                                    </div>

                                    <span className="text-sm text-muted-foreground">
                                        {t('reports.vsPreviousPeriod')}
                                    </span>
                                </div>
                            )}
                        </div>
                    ) : null}
                </CardContent>
            </Card>

            {/* Block 2 — Net Worth Chart */}
            <NetWorthChart filters={filters} />

            {/* Block 3 — Accounts Breakdown */}
            <AccountsStructureChart filters={filters} />
        </div>
    )
}
