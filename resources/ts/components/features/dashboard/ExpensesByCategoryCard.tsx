import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import ReactECharts from '@/components/shared/charts/ReactECharts'
import { legendTextStyle, useChartTheme } from '@/lib/chart-theme'
import { formatCurrency } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import { CATEGORY_COLORS } from '@/constants'
import type { useCategorySummary, useTotalBalance } from '@/hooks'

type CategorySummary = NonNullable<ReturnType<typeof useCategorySummary>['data']>

interface ExpensesByCategoryCardProps {
    summary?: CategorySummary
    /** Fallback when the summary names no currency of its own. */
    currency?: NonNullable<ReturnType<typeof useTotalBalance>['data']>['currency']
}

export function ExpensesByCategoryCard({ summary, currency }: ExpensesByCategoryCardProps) {
    const { t, i18n } = useTranslation('pages')
    const chartTheme = useChartTheme()

    const option = useMemo(() => {
        if (!summary?.data.length) return {}

        const categoryCurrency = summary.currency || currency
        const data = summary.data
            .filter((c) => (c.totalAmount ?? 0) > 0)
            .map((c, i) => ({
                name: localizeDefaultName(c.name),
                value: c.totalAmount ?? 0,
                itemStyle: { color: c.color || CATEGORY_COLORS[i % CATEGORY_COLORS.length] },
            }))

        return {
            tooltip: {
                trigger: 'item',
                confine: true,
                ...chartTheme.tooltip,
                formatter: (params: { name: string; value: number; percent: number }) => {
                    const amount = formatCurrency(params.value, categoryCurrency)
                    return `${params.name}<br/>${amount} (${params.percent.toFixed(1)}%)`
                },
            },
            legend: {
                orient: 'horizontal',
                bottom: 0,
                left: 'center',
                textStyle: legendTextStyle(chartTheme, 11),
                icon: 'circle',
                itemWidth: 8,
                itemHeight: 8,
                itemGap: 8,
            },
            grid: { containLabel: true },
            series: [
                {
                    type: 'pie',
                    radius: ['35%', '60%'],
                    center: ['50%', '40%'],
                    avoidLabelOverlap: false,
                    itemStyle: {
                        borderRadius: 4,
                        borderColor: chartTheme.surface,
                        borderWidth: 2,
                    },
                    label: { show: false },
                    emphasis: { label: { show: false } },
                    data,
                },
            ],
        }
        // i18n.language: category names are localized at build time.
    }, [summary, chartTheme, currency, i18n.language])

    return (
        <Card className="min-w-0">
            <CardHeader>
                <CardTitle>{t('dashboard.expensesByCategory')}</CardTitle>
            </CardHeader>
            <CardContent className="min-w-0">
                {summary && summary.data.some((c) => (c.totalAmount ?? 0) > 0) ? (
                    <div className="min-w-0 overflow-x-auto overscroll-x-contain">
                        <ReactECharts
                            option={option}
                            ariaLabel={t('dashboard.expensesByCategory')}
                            style={{ height: '280px', width: '100%' }}
                            className="min-w-0 max-w-full"
                            opts={{ renderer: 'svg' }}
                        />
                    </div>
                ) : (
                    <div className="flex items-center justify-center h-[280px] text-muted-foreground">
                        {t('dashboard.noDataPeriod')}
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
