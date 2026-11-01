import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import { isDimmed, tooltipRow } from '@/lib/chart-tooltip'
import { axisStyle, legendTextStyle, type ChartTheme } from '@/lib/chart-theme'

export type DynamicsChartType = 'line' | 'bar'

export interface DynamicsDataset {
    name: string
    color: string
    data: number[]
}

export interface DynamicsOptionInput {
    labels: string[]
    datasets: DynamicsDataset[]
    chartType: DynamicsChartType
    groupBy: string
    currency?: string | null
    theme: ChartTheme
    isNarrow: boolean
    /** Index of the series under the cursor; the tooltip greys out every other row. */
    getHoveredIndex?: () => number | null
}

function dynamicsSeries(dataset: DynamicsDataset, chartType: DynamicsChartType, single: boolean) {
    const isLine = chartType === 'line'

    return {
        name: dataset.name,
        type: chartType,
        data: dataset.data,
        smooth: isLine,
        symbol: isLine ? 'circle' : undefined,
        symbolSize: isLine ? 6 : undefined,
        lineStyle: isLine ? { color: dataset.color, width: 2 } : undefined,
        itemStyle: {
            color: dataset.color,
            borderRadius: isLine ? undefined : [4, 4, 0, 0],
        },
        // A fill only reads well under a lone line; stacked fills hide each other
        areaStyle:
            isLine && single
                ? {
                      color: {
                          type: 'linear',
                          x: 0,
                          y: 0,
                          x2: 0,
                          y2: 1,
                          colorStops: [
                              { offset: 0, color: `${dataset.color}30` },
                              { offset: 1, color: `${dataset.color}05` },
                          ],
                      },
                  }
                : undefined,
        barMaxWidth: 20,
    }
}

export function dynamicsOption({
    labels,
    datasets,
    chartType,
    groupBy,
    currency,
    theme,
    isNarrow,
    getHoveredIndex,
}: DynamicsOptionInput) {
    const multiple = datasets.length > 1

    return {
        tooltip: {
            ...theme.tooltip,
            trigger: 'axis',
            axisPointer: { type: chartType === 'bar' ? 'shadow' : 'cross' },
            formatter: (
                params: { seriesName: string; seriesIndex: number; value: number; color: string; axisValue: string }[],
            ) => {
                const label = params[0]?.axisValue || ''
                let html = `<div class="font-medium mb-2">${label}</div>`
                const hovered = getHoveredIndex?.() ?? null
                params.forEach((param) => {
                    if (!param.value) return
                    html += tooltipRow({
                        color: param.color,
                        label: param.seriesName,
                        amount: formatCurrency(param.value, currency),
                        dimmed: isDimmed(hovered, param.seriesIndex),
                    })
                })
                return html
            },
        },
        legend: multiple
            ? {
                  data: datasets.map((dataset) => dataset.name),
                  bottom: 0,
                  textStyle: legendTextStyle(theme),
              }
            : undefined,
        grid: {
            left: isNarrow ? 48 : 60,
            right: 20,
            top: 20,
            bottom: multiple ? 76 : 50,
        },
        xAxis: axisStyle(theme, 'category', {
            data: labels,
            axisLabel: { interval: groupBy === 'day' ? 4 : 0 },
        }),
        yAxis: axisStyle(theme, 'value', {
            axisLabel: { formatter: (val: number) => formatCurrencyCompact(val, currency) },
        }),
        series: datasets.map((dataset) => dynamicsSeries(dataset, chartType, !multiple)),
    }
}
