import { testIdControl, testIdControls } from '@/lib/test-id'
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/charts/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { PieChart, BarChart3, LayoutGrid } from 'lucide-react'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import { useChartTheme } from '@/lib/chart-theme'
import { barOption, CHART_HEIGHT, donutOption, NARROW_CHART_HEIGHT, treemapOption } from './structure-chart-options'

type ViewMode = 'donut' | 'bar' | 'treemap'

export interface StructureItem {
    name: string
    value: number
    color: string
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

/** Breakdown of a total by item, shown as a donut, bars or a treemap. */
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
    const input = useMemo(
        () => ({ items: chartData, total, currency, theme, chartWidth }),
        [chartData, total, currency, theme, chartWidth, i18nInstance.language],
    )
    const options = useMemo(
        () => ({
            donut: () => donutOption(input),
            bar: () => barOption(input),
            treemap: () => treemapOption(input),
        }),
        [input],
    )

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
                        <p className="text-sm text-muted-foreground">{subtitle}</p>
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
                            <ReactECharts option={options[viewMode]()} ariaLabel={title} style={{ height: chartHeight }} key={viewMode} />
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
