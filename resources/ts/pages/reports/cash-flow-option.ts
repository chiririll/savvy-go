import type { CashFlowDataPoint, CashFlowGroupBy, CashFlowOverTimeData } from '@/api/reports'
import { CHART_COLORS, axisStyle, legendTextStyle, verticalFade, withAlpha, type ChartTheme } from '@/lib/chart-theme'
import i18n from '@/lib/i18n'
import { formatCurrency, formatCurrencyCompact } from '@/lib/utils'

import { formatReportPeriodLabel } from './utils'

/** Why the chart cannot be drawn, or 'ok' when it can. A bar chart of one-sided flow says nothing. */
export type CashFlowState = 'noData' | 'noIncomeOrExpenses' | 'noIncome' | 'noExpenses' | 'ok'

export function cashFlowState(items: CashFlowDataPoint[] | undefined): CashFlowState {
    if (!items?.length) return 'noData'
    const hasIncome = items.some((d) => d.income > 0)
    const hasExpenses = items.some((d) => d.expenses > 0)
    if (!hasIncome && !hasExpenses) return 'noIncomeOrExpenses'
    if (!hasIncome) return 'noIncome'
    if (!hasExpenses) return 'noExpenses'
    return 'ok'
}

interface SeriesNames {
    income: string
    expenses: string
    balance: string
    prevIncome: string
    prevExpenses: string
    prevBalance: string
    flow: string
    previousPeriod: string
}

function seriesNames(): SeriesNames {
    return {
        income: i18n.t('pages:reports.series.income'),
        expenses: i18n.t('pages:reports.series.expenses'),
        balance: i18n.t('pages:reports.series.balance'),
        prevIncome: i18n.t('pages:reports.series.prevIncome'),
        prevExpenses: i18n.t('pages:reports.series.prevExpenses'),
        prevBalance: i18n.t('pages:reports.series.prevBalance'),
        flow: i18n.t('pages:reports.series.flow'),
        previousPeriod: i18n.t('pages:reports.series.previousPeriod'),
    }
}

const BAR_WIDTH = 24

interface BarSpec {
    name: string
    stack: string
    data: number[]
    color: string
    borderRadius: number[]
    barGap?: string
}

function bar({ color, borderRadius, ...rest }: BarSpec) {
    return { type: 'bar', itemStyle: { color, borderRadius }, barMaxWidth: BAR_WIDTH, ...rest }
}

function currentSeries(items: CashFlowDataPoint[], names: SeriesNames, theme: ChartTheme) {
    return [
        bar({
            name: names.income,
            stack: 'current',
            data: items.map((d) => d.income),
            color: CHART_COLORS.income,
            borderRadius: [4, 4, 0, 0],
        }),
        bar({
            name: names.expenses,
            stack: 'current',
            // Negative, so the bars point down.
            data: items.map((d) => -d.expenses),
            color: CHART_COLORS.expense,
            borderRadius: [0, 0, 4, 4],
        }),
        {
            name: names.balance,
            type: 'line',
            yAxisIndex: 1,
            data: items.map((d) => d.balance),
            smooth: true,
            symbol: 'circle',
            symbolSize: 6,
            lineStyle: { color: CHART_COLORS.balance, width: 3 },
            itemStyle: { color: CHART_COLORS.balance, borderColor: theme.surface, borderWidth: 2 },
            areaStyle: { color: verticalFade(CHART_COLORS.balance, 0.15) },
        },
    ]
}

function previousSeries(items: CashFlowDataPoint[], names: SeriesNames) {
    return [
        bar({
            name: names.prevIncome,
            stack: 'previous',
            data: items.map((d) => d.prevIncome || 0),
            color: withAlpha(CHART_COLORS.income, 0.3),
            borderRadius: [4, 4, 0, 0],
            barGap: '-100%',
        }),
        bar({
            name: names.prevExpenses,
            stack: 'previous',
            data: items.map((d) => -(d.prevExpenses || 0)),
            color: withAlpha(CHART_COLORS.expense, 0.3),
            borderRadius: [0, 0, 4, 4],
        }),
        {
            name: names.prevBalance,
            type: 'line',
            yAxisIndex: 1,
            data: items.map((d) => d.prevBalance || 0),
            smooth: true,
            symbol: 'none',
            lineStyle: { color: CHART_COLORS.balance, width: 2, type: 'dashed', opacity: 0.5 },
        },
    ]
}

type TooltipParam = { seriesName: string; value?: number; axisValue?: string }

function tooltipFormatter(
    names: SeriesNames,
    currency: CashFlowOverTimeData['currency'],
    theme: ChartTheme,
    showComparison: boolean,
) {
    return (params: TooltipParam[]) => {
        const valueOf = (name: string) => params.find((p) => p.seriesName === name)?.value || 0
        const money = (value: number) => formatCurrency(value, currency)
        const dot = (color: string, name: string, value: number) =>
            `<div class="flex items-center gap-2">
                <span class="w-2 h-2 rounded-full ${color}"></span>
                <span>${name}: <strong>${money(value)}</strong></span>
            </div>`
        const plain = (name: string, value: number) =>
            `<div class="flex items-center gap-2 text-sm"><span>${name}: ${money(value)}</span></div>`

        let html = `<div class="font-medium mb-2">${params[0]?.axisValue || ''}</div>`
        html += `<div class="space-y-1">
            ${dot('bg-green-500', names.income, valueOf(names.income))}
            ${dot('bg-red-500', names.expenses, Math.abs(valueOf(names.expenses)))}
            ${dot('bg-blue-500', names.balance, valueOf(names.balance))}
        </div>`

        if (showComparison) {
            html += `<div class="mt-2 pt-2 space-y-1 opacity-70" style="border-top:1px solid ${theme.axisLine}">
                <div class="text-xs mb-1" style="color:${theme.text}">${names.previousPeriod}</div>
                ${plain(names.income, valueOf(names.prevIncome))}
                ${plain(names.expenses, Math.abs(valueOf(names.prevExpenses)))}
                ${plain(names.balance, valueOf(names.prevBalance))}
            </div>`
        }
        return html
    }
}

interface CashFlowOptionInput {
    data: CashFlowOverTimeData
    groupBy: CashFlowGroupBy
    showComparison: boolean
    theme: ChartTheme
    isNarrow: boolean
}

export function buildCashFlowOption({ data, groupBy, showComparison, theme, isNarrow }: CashFlowOptionInput) {
    const items = data.items
    const currency = data.currency
    const names = seriesNames()
    const formatValue = (val: number) => formatCurrencyCompact(val, currency)
    const sidePadding = isNarrow ? 44 : 60

    return {
        tooltip: {
            ...theme.tooltip,
            trigger: 'axis',
            axisPointer: { type: 'cross', crossStyle: { color: theme.text } },
            formatter: tooltipFormatter(names, currency, theme, showComparison),
        },
        legend: {
            data: [names.income, names.expenses, names.balance],
            bottom: 0,
            itemGap: 16,
            textStyle: legendTextStyle(theme),
        },
        grid: { left: sidePadding, right: sidePadding, top: 20, bottom: 96 },
        xAxis: axisStyle(theme, 'category', {
            data: items.map((d) => formatReportPeriodLabel(d.date, groupBy)),
            axisLabel: { interval: groupBy === 'day' ? Math.floor(items.length / 10) : 0 },
        }),
        yAxis: [
            // Left axis for the bars.
            axisStyle(theme, 'value', {
                name: isNarrow ? undefined : names.flow,
                nameTextStyle: { color: theme.text },
                position: 'left',
                axisLabel: { formatter: formatValue },
            }),
            // Right axis for the balance line.
            axisStyle(theme, 'value', {
                name: isNarrow ? undefined : names.balance,
                nameTextStyle: { color: CHART_COLORS.balance },
                position: 'right',
                axisLabel: { formatter: formatValue, color: CHART_COLORS.balance },
                splitLine: { show: false },
            }),
        ],
        series: [...currentSeries(items, names, theme), ...(showComparison ? previousSeries(items, names) : [])],
    }
}
