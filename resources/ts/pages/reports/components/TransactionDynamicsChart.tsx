import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/charts/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useTransactionReportDynamics } from '@/hooks'
import { defaultGroupBy } from '../types'
import { dynamicsSeriesName, formatReportPeriodLabel } from '../utils'
import type { ReportFilters } from '../types'
import type { ReportTransactionType } from '@/api/reports'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import { useHoveredSeries } from '@/hooks/use-hovered-series'
import { useChartTheme } from '@/lib/chart-theme'
import { ChartTypeBadges, SeriesPicker, TOTAL_SERIES_ID, type SeriesConfig } from './DynamicsControls'
import { dynamicsOption, type DynamicsChartType } from './dynamics-chart-option'

interface TransactionDynamicsChartProps {
    filters: ReportFilters
    type: ReportTransactionType
}

export function TransactionDynamicsChart({ filters, type }: TransactionDynamicsChartProps) {
    const { t, i18n } = useTranslation('pages')
    const theme = useChartTheme()
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>()
    const isNarrow = isNarrowChart(chartWidth)
    const copy =
        type === 'income'
            ? { title: t('reports.incomeDynamics.title'), subtitle: t('reports.incomeDynamics.subtitle') }
            : { title: t('reports.expensesDynamics.title'), subtitle: t('reports.expensesDynamics.subtitle') }
    const seriesLabel = type === 'income' ? t('reports.series.sources') : t('reports.filters.categories')
    const [chartType, setChartType] = useState<DynamicsChartType>('line')
    // The user's picks by series id. Series not picked yet follow the default: every category, not the total.
    const [picked, setPicked] = useState<Record<number, boolean>>({})

    // Detail follows the report period (a year of daily points is slow and unreadable).
    const groupBy = defaultGroupBy(filters)

    const { data, isLoading } = useTransactionReportDynamics(filters, type, groupBy)

    // Derived, not copied into state by an effect: an effect would redraw the chart a second time.
    const series = useMemo<SeriesConfig[]>(
        () =>
            (data?.datasets ?? []).map((dataset) => ({
                id: dataset.id,
                name: dynamicsSeriesName(dataset.id, dataset.name, t),
                color: dataset.color,
                enabled: picked[dataset.id] ?? dataset.id !== TOTAL_SERIES_ID,
            })),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data?.datasets, picked, t, i18n.language],
    )

    const toggleSeries = (id: number) => {
        const current = series.find((item) => item.id === id)
        if (current) setPicked((previous) => ({ ...previous, [id]: !current.enabled }))
    }

    const toggleAllSeries = (enabled: boolean) => {
        setPicked((previous) => ({
            ...previous,
            ...Object.fromEntries(series.filter((item) => item.id !== TOTAL_SERIES_ID).map((item) => [item.id, enabled])),
        }))
    }

    const currency = data?.currency

    const chartData = useMemo(() => {
        if (!data) return { labels: [], datasets: [] }

        const enabledIds = new Set(series.filter((item) => item.enabled).map((item) => item.id))
        return {
            labels: data.dates.map((date) => formatReportPeriodLabel(date, groupBy)),
            datasets: data.datasets
                .filter((dataset) => enabledIds.has(dataset.id))
                .map((dataset) => ({
                    name: dynamicsSeriesName(dataset.id, dataset.name, t),
                    color: dataset.color,
                    data: dataset.data,
                })),
        }
    }, [data, series, groupBy, t, i18n.language])

    const { getHoveredIndex, events: chartEvents } = useHoveredSeries()

    const chartOption = useMemo(
        () => dynamicsOption({ ...chartData, chartType, groupBy, currency, theme, isNarrow, getHoveredIndex }),
        [chartData, chartType, groupBy, currency, theme, isNarrow, getHoveredIndex],
    )

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="space-y-3">
                    <div>
                        <CardTitle className="text-lg">{copy.title}</CardTitle>
                        <p className="text-sm text-muted-foreground">{copy.subtitle}</p>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                        <ChartTypeBadges value={chartType} onChange={setChartType} />
                        <SeriesPicker label={seriesLabel} series={series} onToggle={toggleSeries} onToggleAll={toggleAllSeries} />
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
                    <div ref={chartRef} style={{ height: 350 }}>
                        {/* Mount once measured so the chart animates in a single pass */}
                        {chartWidth > 0 && (
                            <ReactECharts
                                option={chartOption}
                                notMerge
                                onEvents={chartEvents}
                                ariaLabel={copy.title}
                                style={{ height: 350 }}
                                key={`${chartType}-${groupBy}`}
                            />
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
