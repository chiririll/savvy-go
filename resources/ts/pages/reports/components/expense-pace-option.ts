import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'
import i18n from '@/lib/i18n'
import { CHART_COLORS, axisStyle, verticalFade, withAlpha, type ChartTheme } from '@/lib/chart-theme'
import type { MonthChartData } from './expense-pace'

export function buildChartOption(
    chartData: MonthChartData,
    currency: string | null,
    theme: ChartTheme,
    isNarrow: boolean,
) {
    const {
        days,
        idealPace,
        actualExpenses,
        currentDay,
        currentActual,
        budget,
        daysInMonth,
        hasBudget,
        forecastTotal,
    } = chartData

    const maxValue = Math.max(hasBudget && budget ? budget * 1.1 : 0, forecastTotal * 1.1, currentActual * 1.5, 100)

    const formatYAxis = (val: number) => formatCurrencyCompact(val, currency)

    const series: any[] = []
    const nameBudgetPace = i18n.t('pages:reports.series.budgetPace')
    const nameActual = i18n.t('pages:reports.series.actual')
    const nameCurrent = i18n.t('pages:reports.series.current')
    const nameBudget = i18n.t('pages:reports.series.budget')

    if (hasBudget && idealPace.length > 0) {
        series.push({
            name: nameBudgetPace,
            type: 'line',
            data: idealPace,
            smooth: true,
            symbol: 'none',
            showSymbol: false,
            lineStyle: { color: CHART_COLORS.neutral, width: 2, type: 'dashed' },
            areaStyle: { color: verticalFade(CHART_COLORS.neutral, 0.1) },
        })
    }

    series.push({
        name: nameActual,
        type: 'line',
        data: actualExpenses,
        smooth: true,
        symbol: 'none',
        showSymbol: false,
        lineStyle: { color: CHART_COLORS.expense, width: 3 },
        areaStyle: { color: verticalFade(CHART_COLORS.expense, 0.2) },
    })

    if (currentDay !== null && currentDay > 0) {
        series.push({
            name: nameCurrent,
            type: 'scatter',
            data: [[currentDay, currentActual]],
            symbol: 'circle',
            symbolSize: 12,
            itemStyle: {
                color: CHART_COLORS.expense,
                borderColor: theme.surface,
                borderWidth: 2,
                shadowColor: withAlpha(CHART_COLORS.expense, 0.4),
                shadowBlur: 8,
            },
            label: {
                show: true,
                position: 'top',
                formatter: i18n.t('pages:reports.series.today'),
                fontSize: 11,
                color: CHART_COLORS.expense,
                fontWeight: 'bold',
                distance: 8,
            },
            z: 10,
        })
    }

    if (hasBudget && budget) {
        series.push({
            name: nameBudget,
            type: 'line',
            data: Array.from({ length: daysInMonth }, () => budget),
            symbol: 'none',
            lineStyle: { color: CHART_COLORS.income, width: 2, type: 'dotted' },
            markLine: {
                silent: true,
                symbol: 'none',
                label: {
                    show: true,
                    position: isNarrow ? 'insideEndTop' : 'end',
                    formatter: i18n.t('pages:reports.series.budgetLabel', { amount: formatCurrency(budget, currency) }),
                    fontSize: 11,
                    color: CHART_COLORS.income,
                },
                data: [{ yAxis: budget }],
                lineStyle: { color: CHART_COLORS.income, type: 'dotted' },
            },
        })
    }

    return {
        tooltip: {
            ...theme.tooltip,
            trigger: 'axis',
            formatter: (params: { seriesName: string; value: number; axisValue: number }[]) => {
                const day = params[0]?.axisValue
                let html = `<div class="font-medium mb-1">${i18n.t('pages:reports.series.day', { day })}</div>`
                params.forEach((p) => {
                    if (p.value !== undefined && p.seriesName !== nameCurrent) {
                        const colors: Record<string, string> = {
                            [nameActual]: CHART_COLORS.expense,
                            [nameBudgetPace]: CHART_COLORS.neutral,
                            [nameBudget]: CHART_COLORS.income,
                        }
                        const color = colors[p.seriesName] || theme.text
                        html += `<div class="flex items-center gap-2">
                            <span style="background:${color}" class="w-2 h-2 rounded-full inline-block"></span>
                            <span>${p.seriesName}: <strong>${formatCurrency(p.value, currency)}</strong></span>
                        </div>`
                    }
                })
                return html
            },
        },
        grid: { left: isNarrow ? 48 : 60, right: isNarrow ? 12 : 20, top: 40, bottom: 40 },
        xAxis: axisStyle(theme, 'category', {
            data: days,
            axisLabel: { interval: Math.floor(daysInMonth / 7) },
        }),
        yAxis: axisStyle(theme, 'value', {
            axisLabel: { formatter: formatYAxis },
            max: maxValue,
        }),
        series,
    }
}
