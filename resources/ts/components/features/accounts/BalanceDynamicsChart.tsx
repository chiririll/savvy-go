import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from 'echarts-for-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useBalanceHistory } from '@/hooks'
import { useTheme } from '@/hooks/use-theme'
import { cn, formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import { CHART_COLORS } from '@/constants'
import type { BalanceHistorySeries } from '@/api/accounts'

interface BalanceDynamicsChartProps {
    startDate: string
    endDate: string
    className?: string
}

const EMPTY_HIDDEN = new Set<string>()

function seriesKey(series: BalanceHistorySeries): string {
    return series.type === 'total' ? 'total' : `account-${series.id}`
}

function seriesNativeValues(series: BalanceHistorySeries): number[] {
    return series.native_data ?? series.data
}

function lastAmount(series: BalanceHistorySeries): number {
    const values = seriesNativeValues(series)
    return values[values.length - 1] ?? 0
}

export function BalanceDynamicsChart({ startDate, endDate, className }: BalanceDynamicsChartProps) {
    const { t } = useTranslation('pages')
    const { theme } = useTheme()
    const { data: historyData, isLoading } = useBalanceHistory({
        start_date: startDate,
        end_date: endDate,
    })
    const periodKey = `${startDate}|${endDate}`
    const [hiddenByPeriod, setHiddenByPeriod] = useState<Record<string, Set<string>>>({})
    const hiddenKeys = hiddenByPeriod[periodKey] ?? EMPTY_HIDDEN

    const labeledSeries = useMemo(() => {
        if (!historyData) return []

        return historyData.series.map((series, index) => ({
            ...series,
            key: seriesKey(series),
            label: series.type === 'total' ? t('reports.series.total') : series.name,
            color: series.type === 'total' ? '#6366f1' : CHART_COLORS[index % CHART_COLORS.length],
        }))
    }, [historyData, t])

    const toggleSeries = (key: string) => {
        setHiddenByPeriod((current) => {
            const next = new Set(current[periodKey] ?? EMPTY_HIDDEN)
            if (next.has(key)) {
                next.delete(key)
            } else {
                next.add(key)
            }
            return { ...current, [periodKey]: next }
        })
    }

    const chartOption = useMemo(() => {
        if (!historyData || labeledSeries.length === 0) return {}

        const isDark = theme === 'dark'
        const daysDiff = historyData.dates.length
        const visibleLabeled = labeledSeries.filter((series) => !hiddenKeys.has(series.key))
        const visibleCurrencies = new Set(
            visibleLabeled.map((series) => series.currency).filter((code): code is string => Boolean(code))
        )
        const axisCurrency = visibleCurrencies.size === 1 ? [...visibleCurrencies][0] : historyData.currency

        const formatLabel = (date: string) => {
            const parsed = new Date(date)
            if (daysDiff > 90) {
                return `${String(parsed.getMonth() + 1).padStart(2, '0')}.${parsed.getFullYear().toString().slice(2)}`
            }
            return `${parsed.getDate()}.${String(parsed.getMonth() + 1).padStart(2, '0')}`
        }

        return {
            animationDurationUpdate: 0,
            legend: {
                show: false,
                data: labeledSeries.map((series) => series.label),
                selected: Object.fromEntries(
                    labeledSeries.map((series) => [series.label, !hiddenKeys.has(series.key)]),
                ),
            },
            tooltip: {
                trigger: 'axis',
                confine: true,
                backgroundColor: isDark ? '#1f2937' : '#ffffff',
                borderColor: isDark ? '#374151' : '#e5e7eb',
                textStyle: { color: isDark ? '#f3f4f6' : '#1f2937' },
                formatter: (params: { seriesName: string; seriesIndex: number; dataIndex: number; color: string }[]) => {
                    const label = historyData.dates[params[0]?.dataIndex]
                        ? formatLabel(historyData.dates[params[0].dataIndex])
                        : ''
                    let html = `<div class="font-medium mb-2">${label}</div>`
                    params.forEach((param) => {
                        const series = labeledSeries[param.seriesIndex]
                            ?? labeledSeries.find((item) => item.label === param.seriesName)
                        if (!series || hiddenKeys.has(series.key)) return
                        const value = seriesNativeValues(series)[param.dataIndex] ?? 0
                        html += `<div class="flex items-center gap-2">
                            <span class="w-2 h-2 rounded-full" style="background:${param.color}"></span>
                            <span>${param.seriesName}: <strong>${formatCurrency(value, series.currency)}</strong></span>
                        </div>`
                    })
                    return html
                },
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '8%',
                top: '10%',
                containLabel: true,
            },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: historyData.dates.map(formatLabel),
                axisLine: { lineStyle: { color: isDark ? '#374151' : '#e5e7eb' } },
                axisLabel: {
                    color: isDark ? '#9ca3af' : '#6b7280',
                    interval: daysDiff > 60 ? Math.floor(daysDiff / 10) : 'auto',
                },
            },
            yAxis: {
                type: 'value',
                axisLine: { show: false },
                splitLine: { lineStyle: { color: isDark ? '#374151' : '#e5e7eb' } },
                axisLabel: {
                    color: isDark ? '#9ca3af' : '#6b7280',
                    formatter: (value: number) => formatCurrencyCompact(value, axisCurrency, { showSymbol: false }),
                },
            },
            series: labeledSeries.map((series) => ({
                id: series.key,
                name: series.label,
                type: 'line',
                smooth: true,
                data: series.data,
                lineStyle: {
                    color: series.color,
                    width: series.type === 'total' ? 3 : 2,
                },
                itemStyle: { color: series.color },
                areaStyle: series.type === 'total'
                    ? {
                        color: {
                            type: 'linear',
                            x: 0,
                            y: 0,
                            x2: 0,
                            y2: 1,
                            colorStops: [
                                { offset: 0, color: 'rgba(99, 102, 241, 0.2)' },
                                { offset: 1, color: 'rgba(99, 102, 241, 0.02)' },
                            ],
                        },
                    }
                    : undefined,
                emphasis: { focus: 'series' },
            })),
        }
    }, [historyData, labeledSeries, hiddenKeys, theme])

    const hasData = Boolean(historyData && historyData.series.length > 0 && historyData.dates.length > 0)

    return (
        <Card className={cn('min-w-0', className)}>
            <CardHeader>
                <CardTitle>{t('dashboard.balanceDynamics')}</CardTitle>
            </CardHeader>
            <CardContent className="min-w-0">
                {isLoading ? (
                    <Skeleton className="h-[250px] sm:h-[300px]" />
                ) : hasData ? (
                    <div className="min-w-0 space-y-3 overflow-x-auto overscroll-x-contain">
                        <ReactECharts
                            key={periodKey}
                            option={chartOption}
                            style={{ height: '250px', width: '100%' }}
                            className="min-w-0 max-w-full sm:[&]:!h-[300px]"
                            opts={{ renderer: 'svg' }}
                        />
                        <div className="flex min-w-0 flex-wrap justify-center gap-x-3 gap-y-1.5">
                            {labeledSeries.map((series) => {
                                const hidden = hiddenKeys.has(series.key)
                                return (
                                    <button
                                        key={series.key}
                                        type="button"
                                        onClick={() => toggleSeries(series.key)}
                                        aria-pressed={!hidden}
                                        className={cn(
                                            'inline-flex max-w-full min-w-0 items-center gap-1.5 overflow-hidden rounded-md px-1.5 py-0.5 text-xs transition-opacity hover:bg-muted',
                                            hidden && 'opacity-40 line-through',
                                        )}
                                    >
                                        <span
                                            className="size-2 shrink-0 rounded-sm"
                                            style={{ backgroundColor: series.color }}
                                        />
                                        <span className="min-w-0 truncate">{series.label}</span>
                                        <span className="shrink-0 font-mono text-muted-foreground">
                                            {formatCurrency(lastAmount(series), series.currency)}
                                        </span>
                                    </button>
                                )
                            })}
                        </div>
                    </div>
                ) : (
                    <div className="flex items-center justify-center h-[250px] sm:h-[300px] text-muted-foreground">
                        {t('dashboard.noDataPeriod')}
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
