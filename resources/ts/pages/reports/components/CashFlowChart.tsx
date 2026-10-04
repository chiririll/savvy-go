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

interface CashFlowChartProps {
    filters: ReportFilters
}

export function CashFlowChart({ filters }: CashFlowChartProps) {
    const { t, i18n: i18nInstance } = useTranslation('pages')
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
                    color: '#22c55e',
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
                    color: '#ef4444',
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
                    color: '#3b82f6',
                    width: 3,
                },
                itemStyle: {
                    color: '#3b82f6',
                    borderColor: '#fff',
                    borderWidth: 2,
                },
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0,
                        y: 0,
                        x2: 0,
                        y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(59, 130, 246, 0.15)' },
                            { offset: 1, color: 'rgba(59, 130, 246, 0)' },
                        ],
                    },
                },
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
                        color: 'rgba(34, 197, 94, 0.3)',
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
                        color: 'rgba(239, 68, 68, 0.3)',
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
                        color: '#3b82f6',
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
                trigger: 'axis',
                axisPointer: {
                    type: 'cross',
                    crossStyle: {
                        color: '#999',
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

                        html += `<div class="mt-2 pt-2 border-t border-gray-200 space-y-1 opacity-70">`
                        html += `<div class="text-xs text-gray-500 mb-1">${namePreviousPeriod}</div>`
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
                textStyle: {
                    fontSize: 12,
                    color: '#64748b',
                },
            },
            grid: {
                left: 60,
                right: 60,
                top: 20,
                bottom: 50,
            },
            xAxis: {
                type: 'category',
                data: labels,
                axisLabel: {
                    fontSize: 11,
                    color: '#64748b',
                    rotate: groupBy === 'day' && chartData.length > 15 ? 45 : 0,
                    interval: groupBy === 'day' ? Math.floor(chartData.length / 10) : 0,
                },
                axisLine: {
                    lineStyle: { color: '#e2e8f0' },
                },
                axisTick: { show: false },
            },
            yAxis: [
                // Left Y-axis for bars (income/expenses)
                {
                    type: 'value',
                    name: nameFlow,
                    position: 'left',
                    axisLabel: {
                        formatter: formatValue,
                        fontSize: 11,
                        color: '#64748b',
                    },
                    splitLine: {
                        lineStyle: { color: '#f1f5f9', type: 'dashed' },
                    },
                },
                // Right Y-axis for balance line
                {
                    type: 'value',
                    name: nameBalance,
                    position: 'right',
                    axisLabel: {
                        formatter: formatValue,
                        fontSize: 11,
                        color: '#3b82f6',
                    },
                    splitLine: { show: false },
                },
            ],
            series,
        }
    }, [data, showComparison, groupBy, hasIncome, hasExpenses, i18nInstance.language])

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
                <div className="flex items-start justify-between">
                    <div>
                        <CardTitle className="text-lg">{t('reports.cashFlowChart.title')}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {t('reports.cashFlowChart.subtitle')}
                        </p>
                    </div>
                    <div className="flex items-center gap-4">
                        {/* Grouping toggle */}
                        <div className="flex gap-1">
                            {(['day', 'week', 'month'] as CashFlowGroupBy[]).map(g => (
                                <Badge
                                    key={g}
                                    variant={groupBy === g ? 'default' : 'outline'}
                                    className="cursor-pointer"
                                    onClick={() => setGroupBy(g)}
                                >
                                    {t(`reports.groupBy.${g}`)}
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
                    <ReactECharts
                        option={chartOption}
                        style={{ height: 400 }}
                    />
                )}
            </CardContent>
        </Card>
    )
}
