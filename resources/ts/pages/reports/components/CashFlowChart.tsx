import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import ReactECharts from '@/components/shared/charts/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useCashFlowOverTime } from '@/hooks'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'
import { useChartTheme } from '@/lib/chart-theme'

import { buildCashFlowOption, cashFlowState, type CashFlowState } from '../cash-flow-option'
import { defaultGroupBy } from '../types'
import type { ReportFilters } from '../types'

const CHART_HEIGHT = 400

interface CashFlowChartProps {
    filters: ReportFilters
}

export function CashFlowChart({ filters }: CashFlowChartProps) {
    const { t, i18n: i18nInstance } = useTranslation('pages')
    const theme = useChartTheme()
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>()
    const isNarrow = isNarrowChart(chartWidth)
    // Detail follows the report period (a year of daily points is slow and unreadable).
    const groupBy = defaultGroupBy(filters)
    const { data, isLoading, error } = useCashFlowOverTime(filters, groupBy)
    const showComparison = filters.compareWith !== 'none'

    const state = cashFlowState(data?.items)

    const chartOption = useMemo(
        () => (data && state === 'ok' ? buildCashFlowOption({ data, groupBy, showComparison, theme, isNarrow }) : null),
        // The series names are translated at build time, so a language change must rebuild the option.
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data, state, showComparison, groupBy, i18nInstance.language, theme, isNarrow],
    )

    const emptyMessages: Record<Exclude<CashFlowState, 'ok'>, string> = {
        noData: t('reports.noData'),
        noIncomeOrExpenses: t('reports.cashFlowChart.noIncomeOrExpenses'),
        noIncome: t('reports.cashFlowChart.noIncome'),
        noExpenses: t('reports.cashFlowChart.noExpenses'),
    }

    if (error) {
        return (
            <Card>
                <CardContent className="py-8 text-center text-red-500">{t('reports.errors.cashflowChart')}</CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader className="pb-2">
                <div>
                    <CardTitle className="text-lg">{t('reports.cashFlowChart.title')}</CardTitle>
                    <p className="text-sm text-muted-foreground">{t('reports.cashFlowChart.subtitle')}</p>
                </div>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-[400px]" />
                ) : state !== 'ok' ? (
                    <div className="h-[400px] flex items-center justify-center text-muted-foreground">
                        {emptyMessages[state]}
                    </div>
                ) : (
                    <div ref={chartRef} style={{ height: CHART_HEIGHT }}>
                        {/* Mount once measured so the chart animates in a single pass */}
                        {chartWidth > 0 && <ReactECharts option={chartOption} style={{ height: CHART_HEIGHT }} />}
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
