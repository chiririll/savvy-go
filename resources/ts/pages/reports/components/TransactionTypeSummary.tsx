import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn, formatCurrency } from '@/lib/utils'
import { TrendingUp, TrendingDown, Calendar, CalendarDays } from 'lucide-react'
import { useTransactionReportSummary } from '@/hooks'
import type { ReportFilters } from '../types'
import type { ReportTransactionType } from '@/api/reports'

const TONE = {
    income: {
        total: 'text-green-600',
        day: 'bg-green-100 text-green-600 dark:bg-green-950/50 dark:text-green-400',
        week: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400',
        up: 'text-green-600',
        down: 'text-red-600',
    },
    expense: {
        total: 'text-red-600',
        day: 'bg-orange-100 text-orange-600 dark:bg-orange-950/50 dark:text-orange-400',
        week: 'bg-purple-100 text-purple-600 dark:bg-purple-950/50 dark:text-purple-400',
        up: 'text-red-600',
        down: 'text-green-600',
    },
} as const

interface TransactionTypeSummaryProps {
    filters: ReportFilters
    type: ReportTransactionType
}

export function TransactionTypeSummary({ filters, type }: TransactionTypeSummaryProps) {
    const { t } = useTranslation('pages')
    const { data, isLoading } = useTransactionReportSummary(filters, type)
    const tone = TONE[type]
    const metricLabel = type === 'income' ? t('reports.metrics.totalIncome') : t('reports.metrics.totalExpenses')

    const percentChange = data?.previous
        ? ((data.total - data.previous) / data.previous) * 100
        : 0
    const absoluteChange = data ? data.total - (data.previous || 0) : 0
    const isIncrease = percentChange > 0

    return (
        <Card>
            <CardContent className="pt-6">
                {isLoading ? (
                    <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-6">
                        <div className="space-y-2">
                            <Skeleton className="h-4 w-24" />
                            <Skeleton className="h-12 w-40" />
                            <Skeleton className="h-4 w-48" />
                        </div>
                        <div className="flex flex-col gap-3 sm:flex-row sm:gap-6">
                            <Skeleton className="h-20 w-full sm:w-36" />
                            <Skeleton className="h-20 w-full sm:w-36" />
                        </div>
                    </div>
                ) : data ? (
                    <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-6">
                        <div className="min-w-0 space-y-2">
                            <p className="text-sm text-muted-foreground font-medium">
                                {metricLabel}
                            </p>
                            <p className={cn('text-3xl sm:text-5xl font-bold tracking-tight break-words', tone.total)}>
                                {formatCurrency(data.total, data.currency)}
                            </p>

                            {filters.compareWith !== 'none' && data.previous !== null && (
                                <div className="flex flex-wrap items-center gap-x-3 gap-y-1 pt-1">
                                    <span className={cn(
                                        'flex items-center gap-1 text-sm font-medium',
                                        isIncrease ? tone.up : tone.down
                                    )}>
                                        {isIncrease ? (
                                            <TrendingUp className="size-4" />
                                        ) : (
                                            <TrendingDown className="size-4" />
                                        )}
                                        {Math.abs(percentChange).toFixed(1)}%
                                    </span>
                                    <span className="text-sm text-muted-foreground">
                                        {t('reports.vsPreviousAmount', {
                                            amount: `${isIncrease ? '+' : ''}${formatCurrency(absoluteChange, data.currency)}`,
                                        })}
                                    </span>
                                </div>
                            )}
                        </div>

                        <div className="flex flex-col gap-3 sm:flex-row sm:gap-6">
                            <div className="flex min-w-0 items-center gap-3 px-4 py-3 sm:px-5 sm:py-4 bg-muted/50 rounded-xl">
                                <div className={cn('flex shrink-0 items-center justify-center size-10 rounded-lg', tone.day)}>
                                    <Calendar className="size-5" />
                                </div>
                                <div className="min-w-0">
                                    <p className="text-xs text-muted-foreground">{t('reports.metrics.avgPerDay')}</p>
                                    <p className="text-xl font-semibold break-words">{formatCurrency(data.avgPerDay, data.currency)}</p>
                                    {filters.compareWith !== 'none' && data.prevAvgPerDay !== null && (
                                        <p className="text-xs text-muted-foreground">
                                            {t('reports.vsAmount', { amount: formatCurrency(data.prevAvgPerDay, data.currency) })}
                                        </p>
                                    )}
                                </div>
                            </div>

                            <div className="flex min-w-0 items-center gap-3 px-4 py-3 sm:px-5 sm:py-4 bg-muted/50 rounded-xl">
                                <div className={cn('flex shrink-0 items-center justify-center size-10 rounded-lg', tone.week)}>
                                    <CalendarDays className="size-5" />
                                </div>
                                <div className="min-w-0">
                                    <p className="text-xs text-muted-foreground">{t('reports.metrics.avgPerWeek')}</p>
                                    <p className="text-xl font-semibold break-words">{formatCurrency(data.avgPerWeek, data.currency)}</p>
                                    {filters.compareWith !== 'none' && data.prevAvgPerWeek !== null && (
                                        <p className="text-xs text-muted-foreground">
                                            {t('reports.vsAmount', { amount: formatCurrency(data.prevAvgPerWeek, data.currency) })}
                                        </p>
                                    )}
                                </div>
                            </div>
                        </div>
                    </div>
                ) : null}
            </CardContent>
        </Card>
    )
}
