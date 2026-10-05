import { testIdControl, testIdControls } from '@/lib/test-id'
import { useState, useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { useNetWorthHistory } from '@/hooks'
import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import i18n from '@/lib/i18n'
import { defaultGroupBy } from '../types'
import { formatReportPeriodLabel } from '../utils'
import type { ReportFilters } from '../types'
import type { CashFlowGroupBy } from '@/api/reports'
import { groupByLabel } from '@/lib/labels'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import { CHART_COLORS, axisStyle, useChartTheme, verticalFade } from '@/lib/chart-theme'

interface NetWorthChartProps {
    filters: ReportFilters
}

export function NetWorthChart({ filters }: NetWorthChartProps) {
    const { t, i18n: i18nInstance } = useTranslation('pages')
    const theme = useChartTheme()
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>()
    const isNarrow = isNarrowChart(chartWidth)
    const [groupBy, setGroupBy] = useState<CashFlowGroupBy>(() => defaultGroupBy(filters))
    useEffect(() => {
        setGroupBy(defaultGroupBy(filters))
    }, [filters.periodType, filters.customStartDate, filters.customEndDate])
    const { data, isLoading } = useNetWorthHistory(filters, groupBy)

    const currency = data?.currency

    const chartOption = useMemo(() => {
        if (!data) return {}

        return {
            tooltip: {
                ...theme.tooltip,
                trigger: 'axis',
                formatter: (params: { value: number; axisValue: string }[]) => {
                    const p = params[0]
                    const netWorth = i18n.t('pages:reports.series.netWorth')
                    return `<div class="font-medium mb-1">${p.axisValue}</div>
                        <div>${netWorth}: <strong>${formatCurrency(p.value, currency)}</strong></div>`
                },
            },
            grid: {
                left: isNarrow ? 52 : 70,
                right: 20,
                top: 20,
                bottom: 30,
            },
            xAxis: axisStyle(theme, 'category', {
                data: data.dates.map((date) => formatReportPeriodLabel(date, groupBy, 'weekNum')),
                axisLabel: {
                    rotate: groupBy === 'day' ? 45 : 0,
                    interval: groupBy === 'day' ? 4 : 0,
                },
            }),
            yAxis: axisStyle(theme, 'value', {
                axisLabel: { formatter: (val: number) => formatCurrencyCompact(val, currency) },
            }),
            series: [{
                name: i18n.t('pages:reports.series.netWorth'),
                type: 'line',
                data: data.values,
                smooth: true,
                symbol: 'circle',
                symbolSize: 8,
                lineStyle: {
                    color: CHART_COLORS.balance,
                    width: 3,
                },
                itemStyle: {
                    color: CHART_COLORS.balance,
                    borderColor: theme.surface,
                    borderWidth: 2,
                },
                areaStyle: { color: verticalFade(CHART_COLORS.balance, 0.25) },
            }],
        }
    }, [data, groupBy, currency, i18nInstance.language, theme, isNarrow])

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                        <CardTitle className="text-lg">{t('reports.netWorth.chartTitle')}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {t('reports.netWorth.chartSubtitle')}
                        </p>
                    </div>
                    <div className="flex gap-1" {...testIdControls('group-by')}>
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
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-[350px]" />
                ) : !data?.values?.length ? (
                    <div className="h-[350px] flex items-center justify-center text-muted-foreground">
                        {t('reports.noData')}
                    </div>
                ) : (
                    <div ref={chartRef}>
                        <ReactECharts
                            option={chartOption}
                            style={{ height: 350 }}
                            key={groupBy}
                        />
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
