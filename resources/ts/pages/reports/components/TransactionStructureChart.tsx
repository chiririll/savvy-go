import { testIdControl, testIdControls } from '@/lib/test-id'
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { PieChart, BarChart3, LayoutGrid } from 'lucide-react'
import { useTransactionReportByCategory } from '@/hooks'
import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import i18n from '@/lib/i18n'
import { localizeDefaultName } from '@/lib/localized-name'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import type { ReportFilters } from '../types'
import { axisStyle, legendTextStyle, useChartTheme } from '@/lib/chart-theme'
import type { ReportTransactionType } from '@/api/reports'

type ViewMode = 'donut' | 'bar' | 'treemap'

const CHART_HEIGHT = 350
// Narrow screens: legend moves below the donut, so the chart needs more height.
const NARROW_CHART_HEIGHT = 460
const NARROW_DONUT_CENTER_Y = 160
const DONUT_CENTER_X = 0.35

/**
 * Treemap label sized to the tile it sits in. The tile's side is estimated from its
 * share of the chart area (tiles are roughly square): tiny tiles get no label, small
 * ones just the name, larger ones add the amount and percentage.
 */
function treemapLabel(
    item: { name: string; value: number },
    total: number,
    currency: string | null | undefined,
    chartWidth: number,
) {
    const side = Math.sqrt((item.value / (total || 1)) * chartWidth * CHART_HEIGHT)
    if (side < 44) return { show: false }

    const font = Math.max(9, Math.min(14, Math.round(side / 7)))
    const width = Math.max(side - 14, 20)
    const text = (extra: Record<string, unknown> = {}) => ({
        fontSize: font,
        color: '#fff',
        lineHeight: font + 6,
        width,
        overflow: 'truncate',
        ...extra,
    })
    const lines = [`{name|${item.name}}`]
    if (side >= 76) lines.push(`{value|${formatCurrency(item.value, currency)}}`)
    if (side >= 96) lines.push(`{percent|${Math.round((item.value / (total || 1)) * 100)}%}`)

    return {
        show: true,
        formatter: lines.join('\n'),
        rich: {
            name: text({ fontWeight: 'bold' }),
            value: text(),
            percent: text({ fontSize: Math.max(9, font - 2), color: 'rgba(255,255,255,0.8)' }),
        },
    }
}

interface TransactionStructureChartProps {
    filters: ReportFilters
    type: ReportTransactionType
}

export function TransactionStructureChart({ filters, type }: TransactionStructureChartProps) {
    const { t, i18n: i18nInstance } = useTranslation('pages')
    const theme = useChartTheme()
    const copy = type === 'income'
        ? {
            title: t('reports.incomeStructure.title'),
            subtitle: t('reports.incomeStructure.subtitle'),
            noData: t('reports.incomeStructure.noData'),
        }
        : {
            title: t('reports.expensesStructure.title'),
            subtitle: t('reports.expensesStructure.subtitle'),
            noData: t('reports.expensesStructure.noData'),
        }
    const [viewMode, setViewMode] = useState<ViewMode>('donut')
    const { data, isLoading } = useTransactionReportByCategory(filters, type)
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>()

    const chartData = useMemo(() => {
        if (!data?.items) return []
        return data.items.map((item) => ({
            name: localizeDefaultName(item.name),
            value: item.value,
            color: item.color,
        }))
    }, [data, i18nInstance.language])

    const total = data?.total || 0
    const currency = data?.currency

    const donutOption = useMemo(() => {
        // Inner hole diameter in px: radius 50%..75% of half the smaller side.
        const width = chartWidth || 600
        const isNarrow = isNarrowChart(chartWidth)
        const height = isNarrow ? NARROW_CHART_HEIGHT : CHART_HEIGHT
        const hole = Math.min(width, height) * 0.75 * 0.5
        const totalText = formatCurrency(total, currency)
        const totalFont = Math.max(11, Math.min(24, (hole * 0.85) / (totalText.length * 0.6)))
        const labelFont = Math.max(10, Math.min(12, totalFont * 0.6))
        const legendFont = width < 480 ? 11 : 12
        const centerX = width * (isNarrow ? 0.5 : DONUT_CENTER_X)
        const centerY = isNarrow ? NARROW_DONUT_CENTER_Y : height / 2

        return {
        tooltip: {
            ...theme.tooltip,
            trigger: 'item',
            formatter: (params: { name: string; value: number; percent: number }) => {
                return `<div class="font-medium">${params.name}</div>
                    <div>${formatCurrency(params.value, currency)} (${params.percent.toFixed(1)}%)</div>`
            },
        },
        legend: {
            ...(isNarrow
                ? { orient: 'horizontal', left: 12, right: 12, top: NARROW_DONUT_CENTER_Y * 2 - 4 }
                : { orient: 'vertical', right: 20, top: 'center' }),
            textStyle: legendTextStyle(theme, legendFont),
            formatter: (name: string) => {
                const item = chartData.find((entry) => entry.name === name)
                if (item) {
                    const percent = ((item.value / total) * 100).toFixed(1)
                    return `${name}  ${percent}%`
                }
                return name
            },
        },
        series: [{
            type: 'pie',
            radius: ['50%', '75%'],
            center: isNarrow ? ['50%', NARROW_DONUT_CENTER_Y] : ['35%', '50%'],
            avoidLabelOverlap: true,
            itemStyle: {
                borderRadius: 6,
                borderColor: theme.surface,
                borderWidth: 2,
            },
            label: {
                show: false,
            },
            emphasis: {
                label: {
                    show: true,
                    fontSize: 14,
                    fontWeight: 'bold',
                    formatter: '{b}\n{d}%',
                },
                itemStyle: {
                    shadowBlur: 10,
                    shadowOffsetX: 0,
                    shadowColor: 'rgba(0, 0, 0, 0.2)',
                },
            },
            data: chartData.map((item) => ({
                name: item.name,
                value: item.value,
                itemStyle: { color: item.color },
            })),
        }],
        graphic: [{
            type: 'text',
            x: centerX,
            y: centerY - totalFont * 0.6,
            silent: true,
            style: {
                text: totalText,
                textAlign: 'center',
                textVerticalAlign: 'middle',
                fontSize: totalFont,
                fontWeight: 'bold',
                fill: theme.textStrong,
            },
        }, {
            type: 'text',
            x: centerX,
            y: centerY + labelFont * 1.2,
            silent: true,
            style: {
                text: i18n.t('pages:reports.series.total'),
                textAlign: 'center',
                textVerticalAlign: 'middle',
                fontSize: labelFont,
                fill: theme.text,
            },
        }],
        }
    }, [chartData, total, currency, i18nInstance.language, theme, chartWidth])

    const barOption = useMemo(() => {
        const isNarrow = isNarrowChart(chartWidth)
        const sortedData = [...chartData].sort((a, b) => b.value - a.value)

        return {
            tooltip: {
                ...theme.tooltip,
                trigger: 'axis',
                axisPointer: {
                    type: 'shadow',
                },
                formatter: (params: { name: string; value: number }[]) => {
                    const item = params[0]
                    const percent = ((item.value / total) * 100).toFixed(1)
                    return `<div class="font-medium">${item.name}</div>
                        <div>${formatCurrency(item.value, currency)} (${percent}%)</div>`
                },
            },
            grid: {
                left: isNarrow ? 96 : type === 'income' ? 140 : 120,
                right: isNarrow ? 72 : 60,
                top: 20,
                bottom: 20,
            },
            xAxis: axisStyle(theme, 'value', {
                axisLabel: { formatter: (val: number) => formatCurrencyCompact(val, currency) },
            }),
            yAxis: axisStyle(theme, 'category', {
                data: sortedData.map((item) => item.name),
                axisLabel: isNarrow
                    ? { fontSize: 11, color: theme.textStrong, width: 88, overflow: 'truncate', rotate: 0 }
                    : { fontSize: 12, color: theme.textStrong, rotate: 0 },
                axisLine: { show: false },
            }),
            series: [{
                type: 'bar',
                data: sortedData.map((item) => ({
                    value: item.value,
                    itemStyle: {
                        color: item.color,
                        borderRadius: [0, 4, 4, 0],
                    },
                })),
                barWidth: 20,
                label: {
                    show: true,
                    position: 'right',
                    formatter: (params: { value: number }) => formatCurrency(params.value, currency),
                    fontSize: isNarrow ? 10 : 11,
                    color: theme.text,
                },
            }],
        }
    }, [chartData, total, currency, type, theme, chartWidth])

    const treemapOption = useMemo(() => ({
        tooltip: {
            ...theme.tooltip,
            formatter: (params: { name: string; value: number }) => {
                const percent = ((params.value / total) * 100).toFixed(1)
                return `<div class="font-medium">${params.name}</div>
                    <div>${formatCurrency(params.value, currency)} (${percent}%)</div>`
            },
        },
        series: [{
            type: 'treemap',
            width: '100%',
            height: '100%',
            roam: false,
            nodeClick: false,
            breadcrumb: { show: false },
            label: { show: true, position: 'inside', overflow: 'truncate' },
            upperLabel: { show: false },
            itemStyle: {
                borderColor: theme.surface,
                borderWidth: 2,
                gapWidth: 2,
            },
            levels: [{
                itemStyle: {
                    borderColor: theme.surface,
                    borderWidth: 3,
                    gapWidth: 3,
                },
            }],
            data: chartData.map((item) => ({
                name: item.name,
                value: item.value,
                itemStyle: {
                    color: item.color,
                },
                label: treemapLabel(item, total, currency, chartWidth || 600),
            })),
        }],
    }), [chartData, total, currency, i18nInstance.language, theme, chartWidth])

    const getOption = () => {
        switch (viewMode) {
            case 'donut':
                return donutOption
            case 'bar':
                return barOption
            case 'treemap':
                return treemapOption
        }
    }

    const viewModes: { value: ViewMode; label: string; icon: React.ReactNode }[] = [
        { value: 'donut', label: t('reports.views.donut'), icon: <PieChart className="size-3.5" /> },
        { value: 'bar', label: t('reports.views.bar'), icon: <BarChart3 className="size-3.5" /> },
        { value: 'treemap', label: t('reports.views.treemap'), icon: <LayoutGrid className="size-3.5" /> },
    ]

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div>
                        <CardTitle className="text-lg">{copy.title}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {copy.subtitle}
                        </p>
                    </div>
                    <div className="flex flex-wrap gap-1" {...testIdControls('view')}>
                        {viewModes.map((mode) => (
                            <Badge
                                key={mode.value}
                                {...testIdControl(mode.value, viewMode === mode.value)}
                                variant={viewMode === mode.value ? 'default' : 'outline'}
                                className="cursor-pointer gap-1.5"
                                onClick={() => setViewMode(mode.value)}
                            >
                                {mode.icon}
                                {mode.label}
                            </Badge>
                        ))}
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-[350px]" />
                ) : chartData.length === 0 ? (
                    <div className="h-[350px] flex items-center justify-center text-muted-foreground">
                        {copy.noData}
                    </div>
                ) : (
                    <div ref={chartRef}>
                        <ReactECharts
                            option={getOption()}
                            style={{ height: viewMode === 'donut' && isNarrowChart(chartWidth) ? NARROW_CHART_HEIGHT : CHART_HEIGHT }}
                            key={viewMode}
                        />
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
