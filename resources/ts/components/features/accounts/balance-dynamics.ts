import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import { axisStyle, verticalFade, type ChartTheme } from '@/lib/chart-theme'
import type { BalanceHistorySeries } from '@/api/accounts'

export const TOTAL_COLOR = '#6366f1'

export interface LabeledSeries extends BalanceHistorySeries {
    key: string
    label: string
    color: string
}

export function seriesKey(series: BalanceHistorySeries): string {
    return series.type === 'total' ? 'total' : `account-${series.id}`
}

export function seriesNativeValues(series: BalanceHistorySeries): number[] {
    return series.native_data ?? series.data
}

export function lastAmount(series: BalanceHistorySeries): number {
    const values = seriesNativeValues(series)
    return values[values.length - 1] ?? 0
}

/** Long ranges are labelled by month and year, short ones by day and month. */
export function dateLabeler(pointCount: number) {
    return (date: string) => {
        const parsed = new Date(date)
        const month = String(parsed.getMonth() + 1).padStart(2, '0')
        if (pointCount > 90) return `${month}.${parsed.getFullYear().toString().slice(2)}`
        return `${parsed.getDate()}.${month}`
    }
}

/** One currency among the visible series is shown on the axis; mixed ones fall back to the base. */
export function axisCurrencyOf(visible: LabeledSeries[], baseCurrency: string | null): string | null {
    const currencies = new Set(visible.map((series) => series.currency).filter((code): code is string => Boolean(code)))
    return currencies.size === 1 ? [...currencies][0] : baseCurrency
}

export interface BalanceOptionInput {
    dates: string[]
    series: LabeledSeries[]
    hiddenKeys: ReadonlySet<string>
    baseCurrency: string | null
    theme: ChartTheme
}

export function balanceOption({ dates, series, hiddenKeys, baseCurrency, theme }: BalanceOptionInput) {
    const pointCount = dates.length
    const formatLabel = dateLabeler(pointCount)
    const axisCurrency = axisCurrencyOf(
        series.filter((item) => !hiddenKeys.has(item.key)),
        baseCurrency,
    )

    return {
        animationDurationUpdate: 0,
        legend: {
            show: false,
            data: series.map((item) => item.label),
            selected: Object.fromEntries(series.map((item) => [item.label, !hiddenKeys.has(item.key)])),
        },
        tooltip: {
            trigger: 'axis',
            confine: true,
            ...theme.tooltip,
            formatter: (params: { seriesName: string; seriesIndex: number; dataIndex: number; color: string }[]) => {
                const date = dates[params[0]?.dataIndex]
                let html = `<div class="font-medium mb-2">${date ? formatLabel(date) : ''}</div>`
                params.forEach((param) => {
                    const item =
                        series[param.seriesIndex] ?? series.find((candidate) => candidate.label === param.seriesName)
                    if (!item || hiddenKeys.has(item.key)) return
                    const value = seriesNativeValues(item)[param.dataIndex] ?? 0
                    html += `<div class="flex items-center gap-2">
                        <span class="w-2 h-2 rounded-full" style="background:${param.color}"></span>
                        <span>${param.seriesName}: <strong>${formatCurrency(value, item.currency)}</strong></span>
                    </div>`
                })
                return html
            },
        },
        grid: { left: '3%', right: '4%', bottom: '8%', top: '10%', containLabel: true },
        xAxis: axisStyle(theme, 'category', {
            boundaryGap: false,
            data: dates.map(formatLabel),
            axisLabel: { fontSize: 12, interval: pointCount > 60 ? Math.floor(pointCount / 10) : 'auto' },
        }),
        yAxis: axisStyle(theme, 'value', {
            axisLine: { show: false },
            axisLabel: {
                fontSize: 12,
                formatter: (value: number) => formatCurrencyCompact(value, axisCurrency, { showSymbol: false }),
            },
        }),
        series: series.map((item) => ({
            id: item.key,
            name: item.label,
            type: 'line',
            smooth: true,
            data: item.data,
            lineStyle: { color: item.color, width: item.type === 'total' ? 3 : 2 },
            itemStyle: { color: item.color },
            areaStyle: item.type === 'total' ? { color: verticalFade(TOTAL_COLOR, 0.2, 0.02) } : undefined,
            emphasis: { focus: 'series' },
        })),
    }
}
