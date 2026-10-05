import { testIdControl, testIdControls } from '@/lib/test-id'
import { useState, useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { useCashFlowOverTime } from '@/hooks'
import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import i18n from '@/lib/i18n'
import { defaultGroupBy } from '../types'
import { formatReportPeriodLabel } from '../utils'
import type { ReportFilters } from '../types'
import type { CashFlowGroupBy } from '@/api/reports'
import { groupByLabel } from '@/lib/labels'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import { CHART_COLORS, axisStyle, legendTextStyle, useChartTheme, verticalFade, withAlpha } from '@/lib/chart-theme'

interface CashFlowChartProps {
    filters: ReportFilters
}

export function CashFlowChart({ filters }: CashFlowChartProps) {
    const { t, i18n: i18nInstance } = useTranslation('pages')
    const theme = useChartTheme()
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>()
    const isNarrow = isNarrowChart(chartWidth)
    const [groupBy, setGroupBy] = useState<CashFlowGroupBy>(() => defaultGroupBy(filters))
    useEffect(() => {
        setGroupBy(defaultGroupBy(filters))
    }, [filters.periodType, filters.customStartDate, filters.customEndDate])
    const { data, isLoading, error } = useCashFlowOverTime(filters, groupBy)

    const showComparison = filters.compareWith !== 'none'

    const { hasIncome, hasExpenses, noDataMessage } = useMemo(() => {
        if (!data?.items?.length) {
            return { hasIncome: false, hasExpenses: false, noDataMessage: t('reports.noData') }
        }

        const totalIncome = data.items.reduce((sum, d) => sum + d.income, 0)
        const totalExpenses = data.items.reduce((sum, d) => sum + d.expenses, 0)

        const hasIncome = totalIncome > 0
        const hasExpenses = totalExpenses > 0

        let noDataMessage = null
        if (!hasIncome && !hasExpenses) {
            noDataMessage = t('reports.cashFlowChart.noIncomeOrExpenses')
        } else if (!hasIncome) {
            noDataMessage = t('reports.cashFlowChart.noIncome')
        } else if (!hasExpenses) {
            noDataMessage = t('reports.cashFlowChart.noExpenses')
        }

        return { hasIncome, hasExpenses, noDataMessage }
    }, [data, t])

    const chartOption = useMemo(() => {
        if (!data?.items?.length || !hasIncome || !hasExpenses) return null

        const chartData = data.items
        const currency = data.currency

        const labels = chartData.map(d => formatReportPeriodLabel(d.date, groupBy))
        const incomeData = chartData.map(d => d.income)
        const expensesData = chartData.map(d => -d.expenses) // Negative for downward bars
        const balanceData = chartData.map(d => d.balance)

        const nameIncome = i18n.t('pages:reports.series.income')
        const nameExpenses = i18n.t('pages:reports.series.expenses')
        const nameBalance = i18n.t('pages:reports.series.balance')
        const namePrevIncome = i18n.t('pages:reports.series.prevIncome')
        const namePrevExpenses = i18n.t('pages:reports.series.prevExpenses')
        const namePrevBalance = i18n.t('pages:reports.series.prevBalance')
        const nameFlow = i18n.t('pages:reports.series.flow')
        const namePreviousPeriod = i18n.t('pages:reports.series.previousPeriod')

        const series: any[] = [
            // Income bars (green, upward)
            {
                name: nameIncome,
                type: 'bar',
                stack: 'current',
                data: incomeData,
                itemStyle: {
                    color: CHART_COLORS.income,
                    borderRadius: [4, 4, 0, 0],
                },
                barMaxWidth: 24,
            },
            // Expenses bars (red, downward)
            {
                name: nameExpenses,
                type: 'bar',
                stack: 'current',
                data: expensesData,
                itemStyle: {
                    color: CHART_COLORS.expense,
                    borderRadius: [0, 0, 4, 4],
                },
                barMaxWidth: 24,
            },
            // Cumulative balance line
            {
                name: nameBalance,
                type: 'line',
                yAxisIndex: 1,
                data: balanceData,
                smooth: true,
                symbol: 'circle',
                symbolSize: 6,
                lineStyle: {
                    color: CHART_COLORS.balance,
                    width: 3,
                },
                itemStyle: {
                    color: CHART_COLORS.balance,
                    borderColor: theme.surface,
                    borderWidth: 2,
                },
                areaStyle: { color: verticalFade(CHART_COLORS.balance, 0.15) },
            },
        ]

        // Add comparison data if enabled
        if (showComparison) {
            const prevIncomeData = chartData.map(d => d.prevIncome || 0)
            const prevExpensesData = chartData.map(d => -(d.prevExpenses || 0))
            const prevBalanceData = chartData.map(d => d.prevBalance || 0)

            series.push(
                // Previous income (semi-transparent)
                {
                    name: namePrevIncome,
                    type: 'bar',
                    stack: 'previous',
                    data: prevIncomeData,
                    itemStyle: {
                        color: withAlpha(CHART_COLORS.income, 0.3),
                        borderRadius: [4, 4, 0, 0],
                    },
                    barMaxWidth: 24,
                    barGap: '-100%',
                },
                // Previous expenses (semi-transparent)
                {
                    name: namePrevExpenses,
                    type: 'bar',
                    stack: 'previous',
                    data: prevExpensesData,
                    itemStyle: {
                        color: withAlpha(CHART_COLORS.expense, 0.3),
                        borderRadius: [0, 0, 4, 4],
                    },
                    barMaxWidth: 24,
                },
                // Previous balance line (dashed)
                {
                    name: namePrevBalance,
                    type: 'line',
                    yAxisIndex: 1,
                    data: prevBalanceData,
                    smooth: true,
                    symbol: 'none',
                    lineStyle: {
                        color: CHART_COLORS.balance,
                        width: 2,
                        type: 'dashed',
                        opacity: 0.5,
                    },
                }
            )
        }

        const formatValue = (val: number) => formatCurrencyCompact(val, currency)

        return {
            tooltip: {
                ...theme.tooltip,
                trigger: 'axis',
                axisPointer: {
                    type: 'cross',
                    crossStyle: {
                        color: theme.text,
                    },
                },
                formatter: (params: any[]) => {
                    const label = params[0]?.axisValue || ''
                    let html = `<div class="font-medium mb-2">${label}</div>`

                    // Current period
                    const income = params.find((p: any) => p.seriesName === nameIncome)?.value || 0
                    const expenses = Math.abs(params.find((p: any) => p.seriesName === nameExpenses)?.value || 0)
                    const balance = params.find((p: any) => p.seriesName === nameBalance)?.value || 0

                    html += `<div class="space-y-1">`
                    html += `<div class="flex items-center gap-2">
                        <span class="w-2 h-2 rounded-full bg-green-500"></span>
                        <span>${nameIncome}: <strong>${formatCurrency(income, currency)}</strong></span>
                    </div>`
                    html += `<div class="flex items-center gap-2">
                        <span class="w-2 h-2 rounded-full bg-red-500"></span>
                        <span>${nameExpenses}: <strong>${formatCurrency(expenses, currency)}</strong></span>
                    </div>`
                    html += `<div class="flex items-center gap-2">
                        <span class="w-2 h-2 rounded-full bg-blue-500"></span>
                        <span>${nameBalance}: <strong>${formatCurrency(balance, currency)}</strong></span>
                    </div>`
                    html += `</div>`

                    if (showComparison) {
                        const prevIncome = params.find((p: any) => p.seriesName === namePrevIncome)?.value || 0
                        const prevExpenses = Math.abs(params.find((p: any) => p.seriesName === namePrevExpenses)?.value || 0)
                        const prevBalance = params.find((p: any) => p.seriesName === namePrevBalance)?.value || 0

                        html += `<div class="mt-2 pt-2 space-y-1 opacity-70" style="border-top:1px solid ${theme.axisLine}">`
                        html += `<div class="text-xs mb-1" style="color:${theme.text}">${namePreviousPeriod}</div>`
                        html += `<div class="flex items-center gap-2 text-sm">
                            <span>${nameIncome}: ${formatCurrency(prevIncome, currency)}</span>
                        </div>`
                        html += `<div class="flex items-center gap-2 text-sm">
                            <span>${nameExpenses}: ${formatCurrency(prevExpenses, currency)}</span>
                        </div>`
                        html += `<div class="flex items-center gap-2 text-sm">
                            <span>${nameBalance}: ${formatCurrency(prevBalance, currency)}</span>
                        </div>`
                        html += `</div>`
                    }

                    return html
                },
            },
            legend: {
                data: [nameIncome, nameExpenses, nameBalance],
                bottom: 0,
                itemGap: 16,
                textStyle: legendTextStyle(theme),
            },
            grid: {
                left: isNarrow ? 44 : 60,
                right: isNarrow ? 44 : 60,
                top: 20,
                bottom: 96,
            },
            xAxis: axisStyle(theme, 'category', {
                data: labels,
                axisLabel: {
                    interval: groupBy === 'day' ? Math.floor(chartData.length / 10) : 0,
                },
            }),
            yAxis: [
                // Left Y-axis for bars (income/expenses)
                axisStyle(theme, 'value', {
                    name: isNarrow ? undefined : nameFlow,
                    nameTextStyle: { color: theme.text },
                    position: 'left',
                    axisLabel: { formatter: formatValue },
                }),
                // Right Y-axis for balance line
                axisStyle(theme, 'value', {
                    name: isNarrow ? undefined : nameBalance,
                    nameTextStyle: { color: CHART_COLORS.balance },
                    position: 'right',
                    axisLabel: { formatter: formatValue, color: CHART_COLORS.balance },
                    splitLine: { show: false },
                }),
            ],
            series,
        }
    }, [data, showComparison, groupBy, hasIncome, hasExpenses, i18nInstance.language, theme, isNarrow])

    if (error) {
        return (
            <Card>
                <CardContent className="py-8 text-center text-red-500">
                    {t('reports.errors.cashflowChart')}
                </CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div>
                        <CardTitle className="text-lg">{t('reports.cashFlowChart.title')}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {t('reports.cashFlowChart.subtitle')}
                        </p>
                    </div>
                    <div className="flex items-center gap-4">
                        {/* Grouping toggle */}
                        <div className="flex flex-wrap gap-1" {...testIdControls('group-by')}>
                            {(['day', 'week', 'month'] as CashFlowGroupBy[]).map(g => (
                                <Badge
                                    key={g}
                                    {...testIdControl(g, groupBy === g)}
                                    variant={groupBy === g ? 'default' : 'outline'}
                                    className="cursor-pointer"
                                    onClick={() => setGroupBy(g)}
                                >
                                    {groupByLabel(t, g)}
                                </Badge>
                            ))}
                        </div>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-[400px]" />
                ) : noDataMessage ? (
                    <div className="h-[400px] flex items-center justify-center text-muted-foreground">
                        {noDataMessage}
                    </div>
                ) : (
                    <div ref={chartRef}>
                        <ReactECharts
                            option={chartOption}
                            style={{ height: 400 }}
                        />
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
