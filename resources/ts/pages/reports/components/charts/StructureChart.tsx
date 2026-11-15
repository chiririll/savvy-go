import { testId, testIdControl, testIdControls } from '@/lib/test-id'
import { useEffect, useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/charts/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { ArrowLeft, PieChart, BarChart3, LayoutGrid } from 'lucide-react'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import { useChartTheme } from '@/lib/ui/chart-theme'
import { Button } from '@/components/ui/button'
import {
    barOption,
    CHART_HEIGHT,
    chartLevel,
    donutOption,
    NARROW_CHART_HEIGHT,
    treemapOption,
} from '../chart-options/structure-chart-options'

type ViewMode = 'donut' | 'bar' | 'treemap'

export interface StructureItem {
    name: string
    value: number
    color: string
    /** The category this one is grouped under; a parent with lines of its own names itself. */
    parent?: { name: string; color: string }
    /** The group a click on this entry opens; set on top-level groups that have subcategories. */
    drill?: string
    /** A top-level group's subcategories in its shades, drawn inside its slice or bar. */
    parts?: StructureItem[]
}

interface StructureChartProps {
    title: string
    subtitle: string
    noData: string
    items: StructureItem[]
    total: number
    currency?: string | null
    isLoading?: boolean
}

/**
 * Breakdown of a total by item, shown as a donut, bars or a treemap. Grouped items
 * show their top-level groups first; clicking one opens its subcategories.
 */
export function StructureChart({
    title,
    subtitle,
    noData,
    items: chartData,
    total,
    currency,
    isLoading = false,
}: StructureChartProps) {
    const { t, i18n: i18nInstance } = useTranslation('pages')
    const theme = useChartTheme()
    const [viewMode, setViewMode] = useState<ViewMode>('donut')
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>()
    // The group opened in the donut or bars; the treemap shows every group at once.
    const [focus, setFocus] = useState<string | null>(null)
    const level = useMemo(() => chartLevel(chartData, focus), [chartData, focus])
    const opened = focus !== null && chartData.some((item) => item.parent?.name === focus)
    // New data may drop the opened group; go back to the top rather than show a stale level.
    useEffect(() => {
        if (focus !== null && !opened) setFocus(null)
    }, [focus, opened])
    const shown = viewMode === 'treemap' ? chartData : level
    const shownTotal = opened && viewMode !== 'treemap' ? level.reduce((sum, item) => sum + item.value, 0) : total
    const input = useMemo(
        () => ({ items: shown, total: shownTotal, currency, theme, chartWidth }),
        [shown, shownTotal, currency, theme, chartWidth, i18nInstance.language],
    )
    const onEvents = useMemo(
        () => ({
            click: (params: { data?: { drill?: string } }) => {
                if (params.data?.drill) setFocus(params.data.drill)
            },
        }),
        [],
    )
    // Built once per input, not per render: a fresh option (new formatter functions) makes the chart
    // re-apply it and replay the animation.
    // New data remounts the chart so it draws in from zero like the first time, instead of morphing the old shape.
    const dataKey = useMemo(() => shown.map((item) => `${item.parent?.name ?? ''}/${item.name}:${item.value}`).join('|'), [shown])
    const option = useMemo(() => ({ donut: donutOption, bar: barOption, treemap: treemapOption })[viewMode](input), [input, viewMode])

    const chartHeight = viewMode === 'donut' && isNarrowChart(chartWidth) ? NARROW_CHART_HEIGHT : CHART_HEIGHT

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
                        <CardTitle className="text-lg">{title}</CardTitle>
                        {opened && viewMode !== 'treemap' ? (
                            <div className="flex items-center gap-1 text-sm">
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    className="-ml-2 h-7 gap-1 px-2 text-muted-foreground"
                                    {...testId('structure-back')}
                                    onClick={() => setFocus(null)}
                                >
                                    <ArrowLeft className="size-3.5" />
                                    {t('reports.structure.allCategories')}
                                </Button>
                                <span className="text-muted-foreground">/</span>
                                <span className="font-medium">{focus}</span>
                            </div>
                        ) : (
                            <p className="text-sm text-muted-foreground">{subtitle}</p>
                        )}
                    </div>
                    <div className="flex flex-wrap gap-1" {...testIdControls('view')}>
                        {viewModes.map((mode) => (
                            <Badge
                                asChild
                                key={mode.value}
                                variant={viewMode === mode.value ? 'default' : 'outline'}
                                className="cursor-pointer gap-1.5"
                            >
                                {/* A real button, so the keyboard reaches it. */}
                                <button
                                    type="button"
                                    aria-pressed={viewMode === mode.value}
                                    {...testIdControl(mode.value, viewMode === mode.value)}
                                    onClick={() => setViewMode(mode.value)}
                                >
                                    {mode.icon}
                                    {mode.label}
                                </button>
                            </Badge>
                        ))}
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-[350px]" />
                ) : chartData.length === 0 ? (
                    <div className="h-[350px] flex items-center justify-center text-muted-foreground">{noData}</div>
                ) : (
                    <div ref={chartRef} style={{ height: chartHeight }}>
                        {/* Mount once measured so the chart animates in a single pass */}
                        {chartWidth > 0 && (
                            <ReactECharts
                                option={option}
                                onEvents={onEvents}
                                ariaLabel={title}
                                style={{ height: chartHeight }}
                                key={`${viewMode}|${dataKey}`}
                            />
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
