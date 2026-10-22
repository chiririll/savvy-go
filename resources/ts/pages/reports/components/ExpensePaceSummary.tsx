import { useTranslation } from 'react-i18next'
import { TrendingUp, TrendingDown, AlertCircle } from 'lucide-react'
import { cn, formatCurrency } from '@/lib/utils'
import type { MonthChartData } from './expense-pace'

interface ExpensePaceSummaryProps {
    month: MonthChartData
    currency?: string | null
}

function BudgetVerdict({ month, currency }: ExpensePaceSummaryProps) {
    const { t } = useTranslation('pages')
    const amount = formatCurrency(Math.abs(month.forecastDiff), currency)

    return (
        <div
            className={cn(
                'flex min-w-0 items-start gap-2 px-3 py-2 rounded-lg',
                month.isOverBudget
                    ? 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300'
                    : 'bg-green-50 text-green-700 dark:bg-green-950/40 dark:text-green-300',
            )}
        >
            {month.isOverBudget ? (
                <TrendingUp className="size-4 mt-0.5 shrink-0" />
            ) : (
                <TrendingDown className="size-4 mt-0.5 shrink-0" />
            )}
            <span className="text-sm font-medium break-words min-w-0">
                {month.isOverBudget
                    ? t('reports.expensePace.overBudget', { amount: `+${amount}` })
                    : t('reports.expensePace.underBudget', { amount })}
            </span>
        </div>
    )
}

export function ExpensePaceSummary({ month, currency }: ExpensePaceSummaryProps) {
    const { t } = useTranslation('pages')

    return (
        <div className="mt-4 pt-4 border-t">
            {!month.hasBudget && (
                <div className="flex items-center gap-2 text-sm text-amber-600 mb-3">
                    <AlertCircle className="size-4" />
                    <span>{t('reports.expensePace.noBudgetHint')}</span>
                </div>
            )}

            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="min-w-0">
                    <p className="text-sm text-muted-foreground">
                        {month.currentDay ? t('reports.expensePace.projectedEnd') : t('reports.expensePace.totalSpent')}
                    </p>
                    <p className="text-lg font-semibold">
                        {formatCurrency(month.currentDay ? month.forecastTotal : month.totalSpent, currency)}
                    </p>
                </div>
                {month.hasBudget && <BudgetVerdict month={month} currency={currency} />}
            </div>
            <p className="text-xs text-muted-foreground mt-2">
                {month.currentDay
                    ? t('reports.expensePace.spentInDays', {
                          spent: formatCurrency(month.currentActual, currency),
                          days: month.currentDay,
                          avg: formatCurrency(month.dailyAverage, currency),
                          monthDays: month.daysInMonth,
                      })
                    : t('reports.expensePace.spentOverDays', {
                          spent: formatCurrency(month.totalSpent, currency),
                          days: month.daysInMonth,
                          avg: formatCurrency(month.dailyAverage, currency),
                      })}
            </p>
        </div>
    )
}
