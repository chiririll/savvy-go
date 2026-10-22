import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from '@/components/shared/charts/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useBalanceHistory } from '@/hooks'
import { useChartTheme } from '@/lib/chart-theme'
import { cn } from '@/lib/utils'
import { CHART_COLORS } from '@/constants'
import { BalanceSeriesLegend } from './BalanceSeriesLegend'
import { TOTAL_COLOR, balanceOption, seriesKey, type LabeledSeries } from './balance-dynamics'

interface BalanceDynamicsChartProps {
    startDate: string
    endDate: string
    className?: string
}

const EMPTY_HIDDEN = new Set<string>()

export function BalanceDynamicsChart({ startDate, endDate, className }: BalanceDynamicsChartProps) {
    const { t } = useTranslation('pages')
    const chartTheme = useChartTheme()
    const { data: historyData, isLoading } = useBalanceHistory({
        start_date: startDate,
        end_date: endDate,
    })
    const periodKey = `${startDate}|${endDate}`
    const [hiddenByPeriod, setHiddenByPeriod] = useState<Record<string, Set<string>>>({})
    const hiddenKeys = hiddenByPeriod[periodKey] ?? EMPTY_HIDDEN

    const labeledSeries = useMemo<LabeledSeries[]>(() => {
        if (!historyData) return []

        return historyData.series.map((series, index) => ({
            ...series,
            key: seriesKey(series),
            label: series.type === 'total' ? t('reports.series.total') : series.name,
            color: series.type === 'total' ? TOTAL_COLOR : CHART_COLORS[index % CHART_COLORS.length],
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

        return balanceOption({
            dates: historyData.dates,
            series: labeledSeries,
            hiddenKeys,
            baseCurrency: historyData.currency,
            theme: chartTheme,
        })
    }, [historyData, labeledSeries, hiddenKeys, chartTheme])

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
                        <BalanceSeriesLegend series={labeledSeries} hiddenKeys={hiddenKeys} onToggle={toggleSeries} />
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
