import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import i18n from '@/lib/i18n/i18n'
import { isNarrowChart } from '@/hooks/use-element-width'
import { axisStyle, legendTextStyle, type useChartTheme } from '@/lib/ui/chart-theme'
import { colorShades } from '@/lib/ui/color-shades'
import type { StructureItem } from '../charts/StructureChart'

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
// The parts band, inside the main ring (50%..75%) with a small gap to it.
const PARTS_INNER = 43
const PARTS_OUTER = 48

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

export interface StructureGroup {
    name: string
    color: string
    value: number
    children: StructureItem[]
}

/**
 * Items under one parent, largest group first and largest child first. A category
 * without a parent is a group of its own. Without any parent in the data every group
 * is a single item, so callers use `isGrouped` to keep the flat look.
 */
export function groupItems(items: StructureItem[]): StructureGroup[] {
    const groups = new Map<string, StructureGroup>()
    for (const item of items) {
        const key = item.parent?.name ?? item.name
        const group = groups.get(key) ?? { name: key, color: item.parent?.color ?? item.color, value: 0, children: [] }
        group.value += item.value
        group.children.push(item)
        groups.set(key, group)
    }
    const out = [...groups.values()].sort((a, b) => b.value - a.value)
    out.forEach((g) => g.children.sort((a, b) => b.value - a.value))
    return out
}

export const isGrouped = (items: StructureItem[]) => items.some((item) => item.parent)

/** A group's children coloured in shades of the group's colour, largest first. */
const shaded = (group: StructureGroup): StructureItem[] => {
    const shades = colorShades(group.color, group.children.length)
    return group.children.map((item, i) => ({ ...item, color: shades[i] }))
}

/**
 * What the donut and the bars draw: one entry per top-level group, `drill` naming
 * the groups that open into subcategories; or, with `focus` naming a group, that
 * group's children in its shades. A focus that is not in the data shows the top.
 */
export function chartLevel(items: StructureItem[], focus: string | null): StructureItem[] {
    const groups = groupItems(items)
    const open = focus === null ? undefined : groups.find((g) => g.name === focus)
    if (open) return shaded(open).map(({ parent: _, ...item }) => item)
    return groups.map((g) => {
        if (!g.children.some((c) => c.parent)) return { name: g.name, value: g.value, color: g.color }
        const parts = shaded(g).map(({ parent: _, ...item }) => item)
        return { name: g.name, value: g.value, color: g.color, drill: g.name, parts }
    })
}

const percentOf = (value: number, whole: number) => `${((value / (whole || 1)) * 100).toFixed(1)}%`

/** Tooltip lines for a group's parts: colour dot, name, amount and share of the group. */
function partsTooltip(item: StructureItem, currency: string | null | undefined) {
    if (!item.parts) return ''
    const rows = item.parts.map(
        (p) => `<div class="flex items-center gap-1.5">
            <span style="display:inline-block;width:8px;height:8px;border-radius:9999px;background:${p.color}"></span>
            <span>${p.name}</span>
            <span class="ml-auto pl-3 tabular-nums">${formatCurrency(p.value, currency)} · ${percentOf(p.value, item.value)}</span>
        </div>`,
    )
    return `<div class="mt-1 space-y-0.5 text-xs">${rows.join('')}</div>`
}

/** Outer-ring entry for a part: names its group, so the tooltip gives its share of it. */
interface PartDatum {
    group?: StructureItem
}


/** Pie and bar data: a slice that opens shows the pointer, the rest do not. */
const datum = (item: StructureItem) => ({
    name: item.name,
    value: item.value,
    drill: item.drill,
    cursor: item.drill ? 'pointer' : 'default',
    itemStyle: { color: item.color },
})

export function donutOption({ items: chartData, total, currency, theme, chartWidth }: StructureOptionInput) {
    // Inner hole diameter in px: radius 50%..75% of half the smaller side. The parts
    // band sits inside the ring, so with parts the hole is smaller.
    const hasParts = chartData.some((item) => item.parts)
    const width = chartWidth || 600
    const isNarrow = isNarrowChart(chartWidth)
    const height = isNarrow ? NARROW_CHART_HEIGHT : CHART_HEIGHT
    const hole = Math.min(width, height) * 0.75 * 0.5 * (hasParts ? PARTS_INNER / 50 : 1)
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
            formatter: (params: { name: string; value: number; percent: number; seriesIndex: number; data: PartDatum }) => {
                const group = params.data.group
                if (params.seriesIndex === 1 && group) {
                    return `<div class="font-medium">${group.name} › ${params.name}</div>
                        <div>${formatCurrency(params.value, currency)} (${params.percent.toFixed(1)}%)</div>
                        <div class="text-xs opacity-80">${i18n.t('pages:reports.structure.shareOfParent', {
                            percent: percentOf(params.value, group.value),
                            name: group.name,
                        })}</div>`
                }
                const item = chartData.find((entry) => entry.name === params.name)
                return `<div class="font-medium">${params.name}</div>
                    <div>${formatCurrency(params.value, currency)} (${params.percent.toFixed(1)}%)</div>
                    ${item ? partsTooltip(item, currency) : ''}`
            },
        },
        legend: {
            type: 'scroll',
            // Parents only: the parts ring has no entries of its own. Hiding a slice would
            // shift the ring off its parents, so entries do not toggle.
            data: chartData.map((item) => item.name),
            selectedMode: false,
            ...(isNarrow
                ? { orient: 'horizontal', left: 12, right: 12, top: NARROW_DONUT_CENTER_Y * 2 - 4 }
                : { orient: 'vertical', right: 20, top: 'center' }),
            textStyle: legendTextStyle(theme, legendFont),
            pageTextStyle: { color: theme.text },
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
                data: chartData.map(datum),
            },
            // The parts ring: a thin band just inside each parent with subcategories, cut
            // in its shades. Same order and start angle as the main ring, so each band lines
            // up with its parent; a parent without parts leaves a blank gap.
            ...(hasParts
                ? [{
                    type: 'pie',
                    radius: [`${PARTS_INNER}%`, `${PARTS_OUTER}%`],
                    center: isNarrow ? ['50%', NARROW_DONUT_CENTER_Y] : ['35%', '50%'],
                    label: { show: false },
                    labelLine: { show: false },
                    itemStyle: { borderColor: theme.surface, borderWidth: 1 },
                    emphasis: { label: { show: false }, scale: false },
                    data: chartData.flatMap((item): Record<string, unknown>[] =>
                        item.parts
                            ? item.parts.map((part) => ({
                                ...datum(part),
                                drill: item.drill,
                                cursor: 'pointer',
                                group: item,
                            }))
                            : [{
                                name: item.name,
                                value: item.value,
                                cursor: 'default',
                                itemStyle: { color: 'transparent', borderWidth: 0 },
                                tooltip: { show: false },
                                emphasis: { disabled: true },
                            }],
                    ),
                }]
                : []),
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
    // A bar is cut into its parts, in its shades; one without parts is one piece.
    const pieces = (item: StructureItem) => item.parts ?? [item]
    const depth = Math.max(1, ...sortedData.map((item) => pieces(item).length))

    return {
        tooltip: {
            ...theme.tooltip,
            trigger: 'axis',
            axisPointer: {
                type: 'shadow',
            },
            formatter: (params: { dataIndex: number }[]) => {
                const item = sortedData[params[0].dataIndex]
                return `<div class="font-medium">${item.name}</div>
                        <div>${formatCurrency(item.value, currency)} (${percentOf(item.value, total)})</div>
                        ${partsTooltip(item, currency)}`
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
                : { fontSize: 12, color: theme.textStrong, width: 122, overflow: 'truncate', rotate: 0 },
            axisLine: { show: false },
        }),
        series: [
            ...Array.from({ length: depth }, (_, k) => ({
                type: 'bar',
                stack: 'total',
                barWidth: 20,
                data: sortedData.map((item) => {
                    const piece = pieces(item)[k]
                    if (!piece) return null
                    const last = k === pieces(item).length - 1
                    return {
                        ...datum(piece),
                        drill: item.drill,
                        cursor: item.drill ? 'pointer' : 'default',
                        itemStyle: {
                            color: piece.color,
                            borderRadius: last ? [0, 4, 4, 0] : 0,
                            // A hairline between parts, so neighbouring shades read apart.
                            ...(item.parts ? { borderColor: theme.surface, borderWidth: 1 } : {}),
                        },
                    }
                }),
            })),
            // The amount at the end of each bar: an empty piece on top of the stack.
            {
                type: 'bar',
                stack: 'total',
                barWidth: 20,
                silent: true,
                data: sortedData.map(() => 0),
                label: {
                    show: true,
                    position: 'right',
                    formatter: (params: { dataIndex: number }) => formatCurrency(sortedData[params.dataIndex].value, currency),
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
                data: groupItems(chartData).map((g) => {
                    const leaf = (item: StructureItem) => ({
                        name: item.name,
                        value: item.value,
                        itemStyle: { color: item.color },
                        label: treemapLabel(item, total, currency, chartWidth || 600),
                    })
                    // A lone top-level category stays a plain tile; a group wraps its children,
                    // in its shades, under a labelled header.
                    if (!g.children.some((c) => c.parent)) return leaf(g.children[0])
                    return {
                        name: g.name,
                        value: g.value,
                        itemStyle: { color: g.color, borderColor: g.color, borderWidth: 2, gapWidth: 2 },
                        upperLabel: { show: true, height: 18, color: '#fff', fontSize: 11 },
                        children: shaded(g).map(leaf),
                    }
                }),
            },
        ],
    }
}
