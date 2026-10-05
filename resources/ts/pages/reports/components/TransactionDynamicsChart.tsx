import { testIdControl, testIdControls } from '@/lib/test-id'
import { useState, useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { LineChart, BarChart3, ChevronDown } from 'lucide-react'
import { useTransactionReportDynamics } from '@/hooks'
import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import { defaultGroupBy } from '../types'
import { dynamicsSeriesName, formatReportPeriodLabel } from '../utils'
import type { ReportFilters } from '../types'
import type { CashFlowGroupBy, ReportTransactionType } from '@/api/reports'
import { groupByLabel } from '@/lib/labels'
import { axisStyle, legendTextStyle, useChartTheme } from '@/lib/chart-theme'

type ChartType = 'line' | 'bar'

interface SeriesConfig {
    id: number
    name: string
    color: string
    enabled: boolean
}

interface TransactionDynamicsChartProps {
    filters: ReportFilters
    type: ReportTransactionType
}

export function TransactionDynamicsChart({ filters, type }: TransactionDynamicsChartProps) {
    const { t, i18n } = useTranslation('pages')
    const theme = useChartTheme()
    const copy = type === 'income'
        ? {
            title: t('reports.incomeDynamics.title'),
            subtitle: t('reports.incomeDynamics.subtitle'),
        }
        : {
            title: t('reports.expensesDynamics.title'),
            subtitle: t('reports.expensesDynamics.subtitle'),
        }
    const seriesLabel = type === 'income' ? t('reports.series.sources') : t('reports.filters.categories')
    const [chartType, setChartType] = useState<ChartType>('line')
    const [groupBy, setGroupBy] = useState<CashFlowGroupBy>(() => defaultGroupBy(filters))
    const [series, setSeries] = useState<SeriesConfig[]>([])

    useEffect(() => {
        setGroupBy(defaultGroupBy(filters))
    }, [filters.periodType, filters.customStartDate, filters.customEndDate])

    const { data, isLoading } = useTransactionReportDynamics(filters, type, groupBy)

    useEffect(() => {
        if (data?.datasets) {
            setSeries(data.datasets.map((dataset, index) => ({
                id: dataset.id,
                name: dynamicsSeriesName(dataset.id, dataset.name, t),
                color: dataset.color,
                enabled: index === 0,
            })))
        }
    }, [data?.datasets, t, i18n.language])

    const toggleSeries = (id: number) => {
        setSeries((current) => current.map((item) =>
            item.id === id ? { ...item, enabled: !item.enabled } : item
        ))
    }

    const enabledSeries = series.filter((item) => item.enabled)
    const currency = data?.currency

    const chartData = useMemo(() => {
        if (!data) return { labels: [], datasets: [] }

        return {
            labels: data.dates.map((date) => formatReportPeriodLabel(date, groupBy)),
            datasets: data.datasets
                .filter((dataset) => enabledSeries.some((item) => item.id === dataset.id))
                .map((dataset) => ({
                    id: dataset.id,
                    name: dynamicsSeriesName(dataset.id, dataset.name, t),
                    color: dataset.color,
                    data: dataset.data,
                })),
        }
    }, [data, enabledSeries, groupBy, t, i18n.language])

    const chartOption = useMemo(() => {
        const chartSeries = chartData.datasets.map((dataset) => ({
            name: dataset.name,
            type: chartType,
            data: dataset.data,
            smooth: chartType === 'line',
            symbol: chartType === 'line' ? 'circle' : undefined,
            symbolSize: chartType === 'line' ? 6 : undefined,
            lineStyle: chartType === 'line' ? {
                color: dataset.color,
                width: 2,
            } : undefined,
            itemStyle: {
                color: dataset.color,
                borderRadius: chartType === 'bar' ? [4, 4, 0, 0] : undefined,
            },
            areaStyle: chartType === 'line' && chartData.datasets.length === 1 ? {
                color: {
                    type: 'linear',
                    x: 0,
                    y: 0,
                    x2: 0,
                    y2: 1,
                    colorStops: [
                        { offset: 0, color: `${dataset.color}30` },
                        { offset: 1, color: `${dataset.color}05` },
                    ],
                },
            } : undefined,
            barMaxWidth: 20,
        }))

        return {
            tooltip: {
                ...theme.tooltip,
                trigger: 'axis',
                axisPointer: {
                    type: chartType === 'bar' ? 'shadow' : 'cross',
                },
                formatter: (params: { seriesName: string; value: number; color: string; axisValue: string }[]) => {
                    const label = params[0]?.axisValue || ''
                    let html = `<div class="font-medium mb-2">${label}</div>`
                    params.forEach((param) => {
                        html += `<div class="flex items-center gap-2">
                            <span class="w-2 h-2 rounded-full" style="background:${param.color}"></span>
                            <span>${param.seriesName}: <strong>${formatCurrency(param.value, currency)}</strong></span>
                        </div>`
                    })
                    return html
                },
            },
            legend: chartData.datasets.length > 1 ? {
                data: chartData.datasets.map((dataset) => dataset.name),
                bottom: 0,
                textStyle: legendTextStyle(theme),
            } : undefined,
            grid: {
                left: 60,
                right: 20,
                top: 20,
                bottom: chartData.datasets.length > 1 ? 50 : 30,
            },
            xAxis: axisStyle(theme, 'category', {
                data: chartData.labels,
                axisLabel: {
                    rotate: groupBy === 'day' ? 45 : 0,
                    interval: groupBy === 'day' ? 4 : 0,
                },
            }),
            yAxis: axisStyle(theme, 'value', {
                axisLabel: { formatter: (val: number) => formatCurrencyCompact(val, currency) },
            }),
            series: chartSeries,
        }
    }, [chartData, chartType, groupBy, currency, theme])

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                    <div>
                        <CardTitle className="text-lg">{copy.title}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {copy.subtitle}
                        </p>
                    </div>
                    <div className="flex items-center gap-3">
                        <div className="flex gap-1" {...testIdControls('chart-type')}>
                            <Badge
                                {...testIdControl('line', chartType === 'line')}
                                variant={chartType === 'line' ? 'default' : 'outline'}
                                className="cursor-pointer gap-1.5"
                                onClick={() => setChartType('line')}
                            >
                                <LineChart className="size-3.5" />
                                {t('reports.views.line')}
                            </Badge>
                            <Badge
                                {...testIdControl('bar', chartType === 'bar')}
                                variant={chartType === 'bar' ? 'default' : 'outline'}
                                className="cursor-pointer gap-1.5"
                                onClick={() => setChartType('bar')}
                            >
                                <BarChart3 className="size-3.5" />
                                {t('reports.views.bar')}
                            </Badge>
                        </div>

                        <div className="flex gap-1" {...testIdControls('group-by')}>
                            {(['day', 'week', 'month'] as CashFlowGroupBy[]).map((group) => (
                                <Badge
                                    key={group}
                                    {...testIdControl(group, groupBy === group)}
                                    variant={groupBy === group ? 'default' : 'outline'}
                                    className="cursor-pointer"
                                    onClick={() => setGroupBy(group)}
                                >
                                    {groupByLabel(t, group)}
                                </Badge>
                            ))}
                        </div>

                        <Popover>
                            <PopoverTrigger asChild>
                                <Button variant="outline" size="sm" className="h-7 gap-1">
                                    {seriesLabel}
                                    <Badge variant="secondary" className="ml-1 px-1.5 text-xs">
                                        {enabledSeries.length}
                                    </Badge>
                                    <ChevronDown className="size-3" />
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent className="w-56 p-2" align="end">
                                <div className="space-y-1">
                                    {series.map((item) => (
                                        <div
                                            key={item.id}
                                            className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-muted cursor-pointer"
                                            onClick={() => toggleSeries(item.id)}
                                        >
                                            <Checkbox
                                                checked={item.enabled}
                                                className="pointer-events-none"
                                            />
                                            <span
                                                className="w-2.5 h-2.5 rounded-full"
                                                style={{ backgroundColor: item.color }}
                                            />
                                            <Label className="text-sm cursor-pointer flex-1">
                                                {item.name}
                                            </Label>
                                        </div>
                                    ))}
                                </div>
                            </PopoverContent>
                        </Popover>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-[350px]" />
                ) : chartData.labels.length === 0 ? (
                    <div className="h-[350px] flex items-center justify-center text-muted-foreground">
                        {t('reports.noData')}
                    </div>
                ) : (
                    <ReactECharts
                        option={chartOption}
                        style={{ height: 350 }}
                        key={`${chartType}-${groupBy}`}
                    />
                )}
            </CardContent>
        </Card>
    )
}
