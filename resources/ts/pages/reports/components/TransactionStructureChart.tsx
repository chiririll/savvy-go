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
import type { ReportFilters } from '../types'
import type { ReportTransactionType } from '@/api/reports'

type ViewMode = 'donut' | 'bar' | 'treemap'

interface TransactionStructureChartProps {
    filters: ReportFilters
    type: ReportTransactionType
}

export function TransactionStructureChart({ filters, type }: TransactionStructureChartProps) {
    const { t, i18n: i18nInstance } = useTranslation('pages')
    const copyKey = type === 'income' ? 'incomeStructure' : 'expensesStructure'
    const [viewMode, setViewMode] = useState<ViewMode>('donut')
    const { data, isLoading } = useTransactionReportByCategory(filters, type)

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

    const donutOption = useMemo(() => ({
        tooltip: {
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
            textStyle: {
                fontSize: 12,
                color: '#64748b',
            },
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
                borderColor: '#fff',
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
            left: '35%',
            top: '45%',
            style: {
                text: formatCurrency(total, currency),
                textAlign: 'center',
                fontSize: 24,
                fontWeight: 'bold',
                fill: '#1e293b',
            },
        }, {
            type: 'text',
            left: '35%',
            top: '55%',
            style: {
                text: i18n.t('pages:reports.series.total'),
                textAlign: 'center',
                fontSize: 12,
                fill: '#64748b',
            },
        }],
    }), [chartData, total, currency, i18nInstance.language])

    const barOption = useMemo(() => {
        const sortedData = [...chartData].sort((a, b) => b.value - a.value)

        return {
            tooltip: {
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
            xAxis: {
                type: 'value',
                axisLabel: {
                    formatter: (val: number) => formatCurrencyCompact(val, currency),
                    fontSize: 11,
                    color: '#64748b',
                },
                splitLine: {
                    lineStyle: { color: '#f1f5f9', type: 'dashed' },
                },
            },
            yAxis: {
                type: 'category',
                data: sortedData.map((item) => item.name),
                axisLabel: {
                    fontSize: 12,
                    color: '#334155',
                },
                axisLine: { show: false },
                axisTick: { show: false },
            },
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
                    color: '#64748b',
                },
            }],
        }
    }, [chartData, total, currency, type])

    const treemapOption = useMemo(() => ({
        tooltip: {
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
                borderColor: '#fff',
                borderWidth: 2,
                gapWidth: 2,
            },
            levels: [{
                itemStyle: {
                    borderColor: '#fff',
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
    }), [chartData, total, currency, i18nInstance.language])

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
                        <CardTitle className="text-lg">{t(`reports.${copyKey}.title`)}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {t(`reports.${copyKey}.subtitle`)}
                        </p>
                    </div>
                    <div className="flex gap-1">
                        {viewModes.map((mode) => (
                            <Badge
                                key={mode.value}
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
                        {t(`reports.${copyKey}.noData`)}
                    </div>
                ) : (
                    <ReactECharts
                        option={getOption()}
                        style={{ height: 350 }}
                        key={viewMode}
                    />
                )}
            </CardContent>
        </Card>
    )
}
