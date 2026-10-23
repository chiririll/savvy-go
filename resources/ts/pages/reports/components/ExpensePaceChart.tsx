import { useMemo, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/charts/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn, formatCurrency } from '@/lib/utils'
import { useExpensePace } from '@/hooks'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import { useChartTheme } from '@/lib/chart-theme'
import type { ReportFilters } from '../types'
import { processMonthData } from './expense-pace'
import { buildChartOption } from './expense-pace-option'
import { ExpensePaceSummary } from './ExpensePaceSummary'

interface ExpensePaceChartProps {
    filters: ReportFilters
}

export function ExpensePaceChart({ filters }: ExpensePaceChartProps) {
    const { t, i18n } = useTranslation('pages')
    const { data, isLoading, error } = useExpensePace(filters)
    const theme = useChartTheme()
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>()
    const isNarrow = isNarrowChart(chartWidth)
    const [selectedMonth, setSelectedMonth] = useState(0)

    const monthsData = useMemo(() => {
        if (!data?.months?.length) return null
        return data.months.map(processMonthData)
    }, [data, i18n.language])

    useEffect(() => {
        if (monthsData?.length) {
            setSelectedMonth(monthsData.length - 1)
        }
    }, [data])

    const currentMonthData = monthsData?.[Math.min(selectedMonth, (monthsData?.length ?? 1) - 1)]
    const chartOption = useMemo(() => {
        if (!currentMonthData || !data) return null
        return buildChartOption(currentMonthData, data.currency, theme, isNarrow)
    }, [currentMonthData, data, i18n.language, theme, isNarrow])

    const currency = data?.currency

    if (error) {
        return (
            <Card>
                <CardContent className="py-8 text-center text-red-500">{t('reports.errors.expensePace')}</CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                        <CardTitle className="text-lg">{t('reports.expensePace.title')}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {currentMonthData?.hasBudget
                                ? t('reports.expensePace.subtitleBudget')
                                : t('reports.expensePace.subtitleDaily')}
                        </p>
                    </div>
                    {currentMonthData && (
                        <div className="min-w-0 sm:text-right">
                            <p className="text-sm text-muted-foreground">
                                {currentMonthData.hasBudget
                                    ? t('reports.expensePace.budgetRemaining')
                                    : t('reports.expensePace.spentSoFar')}
                            </p>
                            <p
                                className={cn(
                                    'text-xl sm:text-2xl font-bold break-words',
                                    currentMonthData.hasBudget
                                        ? currentMonthData.budgetRemaining >= 0
                                            ? 'text-green-600'
                                            : 'text-red-600'
                                        : 'text-foreground',
                                )}
                            >
                                {currentMonthData.hasBudget
                                    ? formatCurrency(currentMonthData.budgetRemaining, currency)
                                    : formatCurrency(currentMonthData.currentActual, currency)}
                            </p>
                        </div>
                    )}
                </div>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-[300px]" />
                ) : !chartOption || !monthsData ? (
                    <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                        {t('reports.expensePace.noData')}
                    </div>
                ) : (
                    <>
                        {monthsData.length > 1 && (
                            <div className="mb-4">
                                <Select
                                    value={String(selectedMonth)}
                                    onValueChange={(v) => setSelectedMonth(Number(v))}
                                >
                                    <SelectTrigger className="w-full sm:w-[180px]" aria-label={t('reports.filters.month')}>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {monthsData.map((m, i) => (
                                            <SelectItem key={i} value={String(i)}>
                                                {m.label}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </div>
                        )}

                        <div ref={chartRef} style={{ height: 300 }}>
                            {/* Mount once measured so the chart animates in a single pass */}
                            {chartWidth > 0 && <ReactECharts option={chartOption} ariaLabel={t('reports.expensePace.title')} style={{ height: 300 }} />}
                        </div>

                        {currentMonthData && <ExpensePaceSummary month={currentMonthData} currency={currency} />}
                    </>
                )}
            </CardContent>
        </Card>
    )
}
