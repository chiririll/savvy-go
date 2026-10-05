import { useMemo, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn, formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import { TrendingUp, TrendingDown, AlertCircle } from 'lucide-react'
import { useExpensePace } from '@/hooks'
import i18n from '@/lib/i18n'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import { CHART_COLORS, axisStyle, useChartTheme, verticalFade, withAlpha, type ChartTheme } from '@/lib/chart-theme'
import type { ReportFilters } from '../types'
import { formatExpensePaceMonthLabel } from '../utils'
import type { ExpensePaceMonth } from '@/api/reports'

interface ExpensePaceChartProps {
    filters: ReportFilters
}

interface MonthChartData {
    label: string
    budget: number | null
    hasBudget: boolean
    idealPace: number[]
    actualExpenses: number[]
    currentDay: number | null
    currentActual: number
    budgetRemaining: number
    forecastTotal: number
    forecastDiff: number
    isOverBudget: boolean
    dailyAverage: number
    days: number[]
    daysInMonth: number
    totalSpent: number
}

function processMonthData(month: ExpensePaceMonth): MonthChartData {
    const { budget, dailyExpenses, currentDay, daysInMonth, totalSpent, monthStart } = month
    const label = formatExpensePaceMonthLabel(monthStart)

    const hasBudget = budget !== null && budget > 0
    const isCurrentMonth = currentDay !== null && currentDay > 0
    const isPastMonth = currentDay === null && dailyExpenses.length > 0

    const currentActual = isCurrentMonth
        ? (dailyExpenses[currentDay - 1] ?? 0)
        : totalSpent

    let idealPace: number[] = []
    let budgetRemaining = 0

    if (hasBudget && budget) {
        const dailyBudget = budget / daysInMonth
        idealPace = Array.from({ length: daysInMonth }, (_, i) => Math.round(dailyBudget * (i + 1)))
        budgetRemaining = budget - currentActual
    }

    let forecastTotal = 0
    let dailyAverage = 0
    if (isCurrentMonth && currentActual > 0) {
        dailyAverage = currentActual / currentDay
        forecastTotal = Math.round(dailyAverage * daysInMonth)
    } else if (isPastMonth) {
        forecastTotal = totalSpent
        dailyAverage = totalSpent / daysInMonth
    }

    const forecastDiff = hasBudget && budget ? forecastTotal - budget : 0
    const isOverBudget = forecastDiff > 0

    const days = Array.from({ length: daysInMonth }, (_, i) => i + 1)
    const actualExpenses = isCurrentMonth ? dailyExpenses.slice(0, currentDay) : dailyExpenses

    return {
        label,
        budget: hasBudget ? budget : null,
        hasBudget,
        idealPace,
        actualExpenses,
        currentDay: isCurrentMonth ? currentDay : null,
        currentActual,
        budgetRemaining,
        forecastTotal,
        forecastDiff,
        isOverBudget,
        dailyAverage,
        days,
        daysInMonth,
        totalSpent,
    }
}

function buildChartOption(chartData: MonthChartData, currency: string | null, theme: ChartTheme, isNarrow: boolean) {
    const { days, idealPace, actualExpenses, currentDay, currentActual, budget, daysInMonth, hasBudget, forecastTotal } = chartData

    const maxValue = Math.max(
        hasBudget && budget ? budget * 1.1 : 0,
        forecastTotal * 1.1,
        currentActual * 1.5,
        100
    )

    const formatYAxis = (val: number) => formatCurrencyCompact(val, currency)

    const series: any[] = []
    const nameBudgetPace = i18n.t('pages:reports.series.budgetPace')
    const nameActual = i18n.t('pages:reports.series.actual')
    const nameCurrent = i18n.t('pages:reports.series.current')
    const nameBudget = i18n.t('pages:reports.series.budget')

    if (hasBudget && idealPace.length > 0) {
        series.push({
            name: nameBudgetPace,
            type: 'line',
            data: idealPace,
            smooth: true,
            symbol: 'none',
            showSymbol: false,
            lineStyle: { color: CHART_COLORS.neutral, width: 2, type: 'dashed' },
            areaStyle: { color: verticalFade(CHART_COLORS.neutral, 0.1) },
        })
    }

    series.push({
        name: nameActual,
        type: 'line',
        data: actualExpenses,
        smooth: true,
        symbol: 'none',
        showSymbol: false,
        lineStyle: { color: CHART_COLORS.expense, width: 3 },
        areaStyle: { color: verticalFade(CHART_COLORS.expense, 0.2) },
    })

    if (currentDay !== null && currentDay > 0) {
        series.push({
            name: nameCurrent,
            type: 'scatter',
            data: [[currentDay, currentActual]],
            symbol: 'circle',
            symbolSize: 12,
            itemStyle: {
                color: CHART_COLORS.expense,
                borderColor: theme.surface,
                borderWidth: 2,
                shadowColor: withAlpha(CHART_COLORS.expense, 0.4),
                shadowBlur: 8,
            },
            label: {
                show: true,
                position: 'top',
                formatter: i18n.t('pages:reports.series.today'),
                fontSize: 11,
                color: CHART_COLORS.expense,
                fontWeight: 'bold',
                distance: 8,
            },
            z: 10,
        })
    }

    if (hasBudget && budget) {
        series.push({
            name: nameBudget,
            type: 'line',
            data: Array.from({ length: daysInMonth }, () => budget),
            symbol: 'none',
            lineStyle: { color: CHART_COLORS.income, width: 2, type: 'dotted' },
            markLine: {
                silent: true,
                symbol: 'none',
                label: {
                    show: true,
                    position: isNarrow ? 'insideEndTop' : 'end',
                    formatter: i18n.t('pages:reports.series.budgetLabel', { amount: formatCurrency(budget, currency) }),
                    fontSize: 11,
                    color: CHART_COLORS.income,
                },
                data: [{ yAxis: budget }],
                lineStyle: { color: CHART_COLORS.income, type: 'dotted' },
            },
        })
    }

    return {
        tooltip: {
            ...theme.tooltip,
            trigger: 'axis',
            formatter: (params: { seriesName: string; value: number; axisValue: number }[]) => {
                const day = params[0]?.axisValue
                let html = `<div class="font-medium mb-1">${i18n.t('pages:reports.series.day', { day })}</div>`
                params.forEach(p => {
                    if (p.value !== undefined && p.seriesName !== nameCurrent) {
                        const colors: Record<string, string> = {
                            [nameActual]: CHART_COLORS.expense,
                            [nameBudgetPace]: CHART_COLORS.neutral,
                            [nameBudget]: CHART_COLORS.income,
                        }
                        const color = colors[p.seriesName] || theme.text
                        html += `<div class="flex items-center gap-2">
                            <span style="background:${color}" class="w-2 h-2 rounded-full inline-block"></span>
                            <span>${p.seriesName}: <strong>${formatCurrency(p.value, currency)}</strong></span>
                        </div>`
                    }
                })
                return html
            },
        },
        grid: { left: isNarrow ? 48 : 60, right: isNarrow ? 12 : 20, top: 40, bottom: 40 },
        xAxis: axisStyle(theme, 'category', {
            data: days,
            axisLabel: { interval: Math.floor(daysInMonth / 7) },
        }),
        yAxis: axisStyle(theme, 'value', {
            axisLabel: { formatter: formatYAxis },
            max: maxValue,
        }),
        series,
    }
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
                <CardContent className="py-8 text-center text-red-500">
                    {t('reports.errors.expensePace')}
                </CardContent>
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
                                : t('reports.expensePace.subtitleDaily')
                            }
                        </p>
                    </div>
                    {currentMonthData && (
                        <div className="min-w-0 sm:text-right">
                            <p className="text-sm text-muted-foreground">
                                {currentMonthData.hasBudget ? t('reports.expensePace.budgetRemaining') : t('reports.expensePace.spentSoFar')}
                            </p>
                            <p className={cn(
                                'text-xl sm:text-2xl font-bold break-words',
                                currentMonthData.hasBudget
                                    ? (currentMonthData.budgetRemaining >= 0 ? 'text-green-600' : 'text-red-600')
                                    : 'text-foreground'
                            )}>
                                {currentMonthData.hasBudget
                                    ? formatCurrency(currentMonthData.budgetRemaining, currency)
                                    : formatCurrency(currentMonthData.currentActual, currency)
                                }
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
                                <Select value={String(selectedMonth)} onValueChange={(v) => setSelectedMonth(Number(v))}>
                                    <SelectTrigger className="w-full sm:w-[180px]">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {monthsData.map((m, i) => (
                                            <SelectItem key={i} value={String(i)}>{m.label}</SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </div>
                        )}

                        <div ref={chartRef} style={{ height: 300 }}>
                            {/* Mount once measured so the chart animates in a single pass */}
                            {chartWidth > 0 && (
                                <ReactECharts option={chartOption} style={{ height: 300 }} />
                            )}
                        </div>

                        {currentMonthData && (
                            <div className="mt-4 pt-4 border-t">
                                {!currentMonthData.hasBudget && (
                                    <div className="flex items-center gap-2 text-sm text-amber-600 mb-3">
                                        <AlertCircle className="size-4" />
                                        <span>{t('reports.expensePace.noBudgetHint')}</span>
                                    </div>
                                )}

                                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                                    <div className="min-w-0">
                                        <p className="text-sm text-muted-foreground">
                                            {currentMonthData.currentDay ? t('reports.expensePace.projectedEnd') : t('reports.expensePace.totalSpent')}
                                        </p>
                                        <p className="text-lg font-semibold">
                                            {currentMonthData.currentDay ? formatCurrency(currentMonthData.forecastTotal, currency) : formatCurrency(currentMonthData.totalSpent, currency)}
                                        </p>
                                    </div>
                                    {currentMonthData.hasBudget && (
                                        <div className={cn(
                                            'flex min-w-0 items-start gap-2 px-3 py-2 rounded-lg',
                                            currentMonthData.isOverBudget ? 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300' : 'bg-green-50 text-green-700 dark:bg-green-950/40 dark:text-green-300'
                                        )}>
                                            {currentMonthData.isOverBudget ? <TrendingUp className="size-4 mt-0.5 shrink-0" /> : <TrendingDown className="size-4 mt-0.5 shrink-0" />}
                                            <span className="text-sm font-medium break-words min-w-0">
                                                {currentMonthData.isOverBudget
                                                    ? t('reports.expensePace.overBudget', { amount: `+${formatCurrency(Math.abs(currentMonthData.forecastDiff), currency)}` })
                                                    : t('reports.expensePace.underBudget', { amount: formatCurrency(Math.abs(currentMonthData.forecastDiff), currency) })}
                                            </span>
                                        </div>
                                    )}
                                </div>
                                {currentMonthData.currentDay ? (
                                    <p className="text-xs text-muted-foreground mt-2">
                                        {t('reports.expensePace.spentInDays', {
                                            spent: formatCurrency(currentMonthData.currentActual, currency),
                                            days: currentMonthData.currentDay,
                                            avg: formatCurrency(currentMonthData.dailyAverage, currency),
                                            monthDays: currentMonthData.daysInMonth,
                                        })}
                                    </p>
                                ) : (
                                    <p className="text-xs text-muted-foreground mt-2">
                                        {t('reports.expensePace.spentOverDays', {
                                            spent: formatCurrency(currentMonthData.totalSpent, currency),
                                            days: currentMonthData.daysInMonth,
                                            avg: formatCurrency(currentMonthData.dailyAverage, currency),
                                        })}
                                    </p>
                                )}
                            </div>
                        )}
                    </>
                )}
            </CardContent>
        </Card>
    )
}