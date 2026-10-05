import { useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import ReactECharts from '@/components/shared/ReactECharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useMoneyFlow } from '@/hooks'
import { formatCurrency } from '@/lib/utils'
import type { ReportFilters } from '../types'
import { localizeSavingsNodeName } from '../utils'
import { useChartTheme } from '@/lib/chart-theme'
import { isNarrowChart, useElementWidth } from '@/hooks/use-element-width'

interface SankeyDiagramProps {
    filters: ReportFilters
}

export function SankeyDiagram({ filters }: SankeyDiagramProps) {
    const { t, i18n } = useTranslation('pages')
    const navigate = useNavigate()
    const theme = useChartTheme()
    const [chartRef, chartWidth] = useElementWidth<HTMLDivElement>()
    const isNarrow = isNarrowChart(chartWidth)
    const { data, isLoading, error } = useMoneyFlow(filters)

    const sankeyOption = useMemo(() => {
        if (!data || data.nodes.length === 0 || data.links.length === 0) return null

        const nodes = data.nodes.map((node) => ({
            ...node,
            name: localizeSavingsNodeName(node.name, t),
        }))
        const links = data.links.map((link) => ({
            ...link,
            source: localizeSavingsNodeName(link.source, t),
            target: localizeSavingsNodeName(link.target, t),
        }))

        return {
            tooltip: {
                ...theme.tooltip,
                trigger: 'item',
                triggerOn: 'mousemove',
                formatter: (params: { data: { source?: string; target?: string; value: number }; name?: string; value?: number }) => {
                    if (params.data.source && params.data.target) {
                        // Link tooltip
                        return `${params.data.source} → ${params.data.target}<br/><strong>${formatCurrency(params.data.value, data.currency)}</strong>`
                    }
                    // Node tooltip - params.value contains the calculated sum from ECharts
                    const nodeValue = params.value ?? params.data.value ?? 0
                    return `${params.name}: <strong>${formatCurrency(nodeValue, data.currency)}</strong>`
                },
            },
            series: [{
                type: 'sankey',
                layout: 'none',
                emphasis: {
                    focus: 'adjacency',
                },
                nodeAlign: 'justify',
                lineStyle: {
                    color: 'gradient',
                    curveness: 0.5,
                },
                nodeGap: isNarrow ? 8 : 12,
                nodeWidth: isNarrow ? 12 : 20,
                left: 8,
                right: isNarrow ? 84 : '20%',
                label: {
                    fontSize: isNarrow ? 11 : 13,
                    ...(isNarrow ? { width: 76, overflow: 'truncate' } : {}),
                    color: theme.textStrong,
                    textBorderWidth: 0,
                },
                data: nodes,
                links,
            }],
        }
    }, [data, t, i18n.language, theme, isNarrow])

    const handleSankeyClick = useCallback((params: { data: { source?: string; target?: string } }) => {
        if (params.data.source && params.data.target) {
            const searchParams = new URLSearchParams()
            searchParams.set('search', params.data.target)
            navigate(`/transactions?${searchParams.toString()}`)
        }
    }, [navigate])

    if (error) {
        return (
            <Card>
                <CardContent className="py-8 text-center text-red-500">
                    {t('reports.errors.moneyFlow')}
                </CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader className="pb-2">
                <CardTitle className="text-lg">{t('reports.moneyFlow.title')}</CardTitle>
                <p className="text-sm text-muted-foreground">
                    {t('reports.moneyFlow.subtitle')}
                </p>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-[400px]" />
                ) : !sankeyOption ? (
                    <div className="h-[400px] flex items-center justify-center text-muted-foreground">
                        {t('reports.noData')}
                    </div>
                ) : (
                    <div ref={chartRef}>
                        <ReactECharts
                            option={sankeyOption}
                            style={{ height: 400 }}
                            onEvents={{
                                click: handleSankeyClick,
                            }}
                        />
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
