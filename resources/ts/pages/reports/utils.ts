import type { TFunction } from 'i18next'
import i18n, { intlLocale } from '@/lib/i18n'
import { formatYearMonth, parseDateKey } from '@/lib/dates'
import type { CashFlowGroupBy } from '@/api/reports'

const SAVINGS_NODE_IDS = new Set(['__savings__', 'Savings', 'messages.reports.savings'])

export function dynamicsSeriesName(id: number, fallbackName: string, t: TFunction): string {
    return id === 0 ? t('reports.series.total') : fallbackName
}

export function localizeSavingsNodeName(name: string, t: TFunction): string {
    return SAVINGS_NODE_IDS.has(name) ? t('reports.series.savings') : name
}

/** ISO-8601 week number (Carbon weekOfYear). */
function isoWeekNumber(date: Date): number {
    const tmp = new Date(date.getFullYear(), date.getMonth(), date.getDate())
    tmp.setDate(tmp.getDate() + 3 - ((tmp.getDay() + 6) % 7))
    const week1 = new Date(tmp.getFullYear(), 0, 4)
    return 1 + Math.round(((tmp.getTime() - week1.getTime()) / 86400000 - 3 + ((week1.getDay() + 6) % 7)) / 7)
}

export function formatReportPeriodLabel(
    dateKey: string,
    groupBy: CashFlowGroupBy,
    variant: 'week' | 'weekNum' = 'week',
): string {
    const date = parseDateKey(dateKey)
    const locale = intlLocale()

    if (groupBy === 'week') {
        if (variant === 'weekNum') {
            return i18n.t('pages:reports.axis.weekNum', {
                week: isoWeekNumber(date),
                date: date.toLocaleDateString(locale, { month: 'short', year: '2-digit' }),
            })
        }

        return i18n.t('pages:reports.axis.week', {
            date: date.toLocaleDateString(locale, { month: 'short', day: 'numeric' }),
        })
    }

    if (groupBy === 'month') {
        return date.toLocaleDateString(locale, { month: 'short', year: '2-digit' })
    }

    return date.toLocaleDateString(locale, { month: 'short', day: 'numeric' })
}

export function formatExpensePaceMonthLabel(monthStart: string): string {
    return parseDateKey(monthStart).toLocaleDateString(intlLocale(), { month: 'short', year: 'numeric' })
}

// Generate months for the last 2 years
export function getMonthOptions() {
    const months = []
    const now = new Date()
    for (let i = 0; i < 24; i++) {
        const date = new Date(now.getFullYear(), now.getMonth() - i, 1)
        const value = formatYearMonth(date)
        const label = date.toLocaleDateString(intlLocale(), { month: 'long', year: 'numeric' })
        months.push({ value, label })
    }
    return months
}

// Get current month in YYYY-MM format (timezone-safe)
export function getCurrentMonth(): string {
    return formatYearMonth(new Date())
}

// Generate quarters for the last 2 years
export function getQuarterOptions() {
    const quarters = []
    const now = new Date()
    const currentYear = now.getFullYear()
    const currentQuarter = Math.ceil((now.getMonth() + 1) / 3)

    for (let year = currentYear; year >= currentYear - 2; year--) {
        const maxQ = year === currentYear ? currentQuarter : 4
        for (let q = maxQ; q >= 1; q--) {
            quarters.push({
                value: `${year}-Q${q}`,
                label: i18n.t('pages:reports.filters.quarterLabel', { q, year }),
            })
        }
    }
    return quarters
}

// Generate years
export function getYearOptions() {
    const years = []
    const currentYear = new Date().getFullYear()
    for (let year = currentYear; year >= currentYear - 5; year--) {
        years.push({ value: year.toString(), label: year.toString() })
    }
    return years
}

