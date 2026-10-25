import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import i18n from '@/lib/i18n'
import { isNarrowChart } from '@/hooks/use-element-width'
import { axisStyle, legendTextStyle, type useChartTheme } from '@/lib/chart-theme'
import type { StructureItem } from './StructureChart'

type ChartTheme = ReturnType<typeof useChartTheme>

export interface StructureOptionInput {
    items: StructureItem[]
    total: number
    currency?: string | null
    theme: ChartTheme
    chartWidth: number
}

export const CHART_HEIGHT = 350
// Narrow screens: legend moves below the donut, so the chart needs more height.
export const NARROW_CHART_HEIGHT = 460
const NARROW_DONUT_CENTER_Y = 160
const DONUT_CENTER_X = 0.35

// Squarify aims for this width/height ratio, so tiles come out wider than tall
// (rows are split horizontally first); ECharts' default is the golden ratio (~1.6).
const TREEMAP_SQUARE_RATIO = 2.2

/**
 * Treemap label sized to the tile it sits in. The tile is estimated from its share of
 * the chart area and the layout's target ratio: tiles too small get no label, and
 * the amount and percentage are added only when the tile is tall and wide enough.
 */
function treemapLabel(
    item: { name: string; value: number },
    total: number,
    currency: string | null | undefined,
    chartWidth: number,
) {
    const area = (item.value / (total || 1)) * chartWidth * CHART_HEIGHT
    const tileW = Math.sqrt(area * TREEMAP_SQUARE_RATIO)
    const tileH = Math.sqrt(area / TREEMAP_SQUARE_RATIO)
    if (tileW < 26 || tileH < 15) return { show: false }

    const font = Math.max(8, Math.min(14, Math.round(Math.min(tileW / 6, tileH / 2.6))))
    const lineHeight = font + 4
    const lines = Math.max(1, Math.floor((tileH - 4) / lineHeight))
    const width = Math.max(tileW - 8, 18)
    const text = (extra: Record<string, unknown> = {}) => ({
        fontSize: font,
        color: '#fff',
        lineHeight,
        width,
        overflow: 'truncate',
        ...extra,
    })
    const amount = formatCurrency(item.value, currency)
    // Rich-text markup can't carry these characters inside a name
    const name = item.name.replace(/[{}|]/g, ' ')
    const rows = [`{name|${name}}`]
    if (lines >= 2 && width >= amount.length * font * 0.55) rows.push(`{value|${amount}}`)
    if (lines >= 3) rows.push(`{percent|${Math.round((item.value / (total || 1)) * 100)}%}`)

    return {
        show: true,
        formatter: rows.join('\n'),
        rich: {
            name: text({ fontWeight: 'bold' }),
            value: text(),
            percent: text({ fontSize: Math.max(9, font - 2), color: 'rgba(255,255,255,0.8)' }),
        },
    }
}

export function donutOption({ items: chartData, total, currency, theme, chartWidth }: StructureOptionInput) {
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
        series: [
            {
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
                    // The legend and tooltip already name and size the slice; a label on it just clutters the ring
                    label: { show: false },
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
            },
        ],
        graphic: [
            {
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
            },
            {
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
            },
        ],
    }
}

export function barOption({ items: chartData, total, currency, theme, chartWidth }: StructureOptionInput) {
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
            left: isNarrow ? 96 : 130,
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
        series: [
            {
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
            },
        ],
    }
}

export function treemapOption({ items: chartData, total, currency, theme, chartWidth }: StructureOptionInput) {
    return {
        tooltip: {
            ...theme.tooltip,
            formatter: (params: { name: string; value: number }) => {
                const percent = ((params.value / total) * 100).toFixed(1)
                return `<div class="font-medium">${params.name}</div>
                    <div>${formatCurrency(params.value, currency)} (${percent}%)</div>`
            },
        },
        series: [
            {
                type: 'treemap',
                left: 0,
                top: 0,
                right: 0,
                bottom: 0,
                roam: false,
                squareRatio: TREEMAP_SQUARE_RATIO,
                nodeClick: false,
                breadcrumb: { show: false },
                label: { show: true, position: 'inside', overflow: 'truncate' },
                upperLabel: { show: false },
                itemStyle: {
                    borderColor: theme.surface,
                    borderWidth: 2,
                    gapWidth: 2,
                },
                levels: [
                    {
                        itemStyle: {
                            borderColor: theme.surface,
                            borderWidth: 3,
                            gapWidth: 3,
                        },
                    },
                ],
                data: chartData.map((item) => ({
                    name: item.name,
                    value: item.value,
                    itemStyle: {
                        color: item.color,
                    },
                    label: treemapLabel(item, total, currency, chartWidth || 600),
                })),
            },
        ],
    }
}
