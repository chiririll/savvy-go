import { testIdControl, testIdControls } from '@/lib/test-id'
import { useState, useMemo, useRef, useLayoutEffect } from 'react'
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
import type { ReportFilters } from '../types'
import { axisStyle, legendTextStyle, useChartTheme } from '@/lib/chart-theme'
import type { ReportTransactionType } from '@/api/reports'

type ViewMode = 'donut' | 'bar' | 'treemap'

const CHART_HEIGHT = 350
const DONUT_CENTER_X = 0.35

function useElementWidth<T extends HTMLElement>(ready: boolean) {
    const ref = useRef<T>(null)
    const [width, setWidth] = useState(0)
    useLayoutEffect(() => {
        const el = ref.current
        if (!el) return
        setWidth(el.clientWidth)
        const observer = new ResizeObserver(([entry]) => setWidth(Math.round(entry.contentRect.width)))
        observer.observe(el)
        return () => observer.disconnect()
    }, [ready])
    return [ref, width] as const
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
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>(!isLoading && !!data?.items?.length)

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
        const hole = Math.min(width, CHART_HEIGHT) * 0.75 * 0.5
        const totalText = formatCurrency(total, currency)
        const totalFont = Math.max(11, Math.min(24, (hole * 0.85) / (totalText.length * 0.6)))
        const labelFont = Math.max(10, Math.min(12, totalFont * 0.6))
        const legendFont = width < 480 ? 11 : 12
        const centerX = width * DONUT_CENTER_X
        const centerY = CHART_HEIGHT / 2

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
            orient: 'vertical',
            right: 20,
            top: 'center',
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
            center: ['35%', '50%'],
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
                left: type === 'income' ? 140 : 120,
                right: 60,
                top: 20,
                bottom: 20,
            },
            xAxis: axisStyle(theme, 'value', {
                axisLabel: { formatter: (val: number) => formatCurrencyCompact(val, currency) },
            }),
            yAxis: axisStyle(theme, 'category', {
                data: sortedData.map((item) => item.name),
                axisLabel: { fontSize: 12, color: theme.textStrong },
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
                    fontSize: 11,
                    color: theme.text,
                },
            }],
        }
    }, [chartData, total, currency, type, theme])

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
            label: {
                show: true,
                formatter: (params: { name: string; value: number }) => {
                    const percent = ((params.value / total) * 100).toFixed(0)
                    return `{name|${params.name}}\n{value|${formatCurrency(params.value, currency)}}\n{percent|${percent}%}`
                },
                rich: {
                    name: {
                        fontSize: 13,
                        fontWeight: 'bold',
                        color: '#fff',
                        lineHeight: 20,
                    },
                    value: {
                        fontSize: 14,
                        color: '#fff',
                        lineHeight: 22,
                    },
                    percent: {
                        fontSize: 11,
                        color: 'rgba(255,255,255,0.8)',
                    },
                },
                position: 'inside',
            },
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
            })),
        }],
    }), [chartData, total, currency, i18nInstance.language, theme])

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
                <div className="flex items-start justify-between">
                    <div>
                        <CardTitle className="text-lg">{copy.title}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {copy.subtitle}
                        </p>
                    </div>
                    <div className="flex gap-1" {...testIdControls('view')}>
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
                            style={{ height: CHART_HEIGHT }}
                            key={viewMode}
                        />
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
