import type { CashFlowGroupBy } from '@/api/reports'
import { formatDateLocal, formatYearMonth } from '@/lib/dates'

export type PeriodType = 'last_30_days' | 'month' | 'quarter' | 'year' | 'ytd' | 'custom'
export type CompareType = 'none' | 'previous_period' | 'same_period_last_year'
export type ReportTab = 'overview' | 'cashflow' | 'expenses' | 'income' | 'networth'

export interface ReportFilters {
    periodType: PeriodType
    selectedMonth: string
    selectedQuarter: string
    selectedYear: string
    customStartDate: string
    customEndDate: string
    compareWith: CompareType
    accountIds: number[]
    categoryIds: number[]
    tagIds: number[]
}

const now = new Date()

export const DEFAULT_FILTERS: ReportFilters = {
    periodType: 'last_30_days',
    selectedMonth: formatYearMonth(now),
    selectedQuarter: `${now.getFullYear()}-Q${Math.ceil((now.getMonth() + 1) / 3)}`,
    selectedYear: now.getFullYear().toString(),
    customStartDate: formatDateLocal(new Date(now.getFullYear(), now.getMonth(), 1)),
    customEndDate: formatDateLocal(now),
    compareWith: 'previous_period',
    accountIds: [],
    categoryIds: [],
    tagIds: [],
}

const DAY_MS = 86_400_000

function daysBetween(start: Date, end: Date): number {
    return Math.round((end.getTime() - start.getTime()) / DAY_MS) + 1
}

/** Number of days in the period the filters select (0 when it can't be worked out). */
export function reportPeriodDays(filters: ReportFilters, today = new Date()): number {
    switch (filters.periodType) {
        case 'last_30_days':
            return 30
        case 'month': {
            const [year, month] = filters.selectedMonth.split('-').map(Number)
            return year && month ? new Date(year, month, 0).getDate() : 30
        }
        case 'quarter': {
            const [year, quarter] = filters.selectedQuarter.split('-Q').map(Number)
            if (!year || !quarter) return 91
            const start = new Date(year, (quarter - 1) * 3, 1)
            return daysBetween(start, new Date(year, quarter * 3, 0))
        }
        case 'year': {
            const year = Number(filters.selectedYear) || today.getFullYear()
            return daysBetween(new Date(year, 0, 1), new Date(year, 11, 31))
        }
        case 'ytd':
            return daysBetween(new Date(today.getFullYear(), 0, 1), today)
        case 'custom': {
            const days = daysBetween(new Date(filters.customStartDate), new Date(filters.customEndDate))
            return Number.isFinite(days) && days > 0 ? days : 30
        }
    }
}

/**
 * Chart detail for the selected period, by its length alone: a year of daily
 * points is slow and unreadable, a month of monthly points shows nothing.
 */
export function defaultGroupBy(filters: ReportFilters): CashFlowGroupBy {
    const days = reportPeriodDays(filters)
    if (days <= 45) return 'day'
    if (days <= 185) return 'week'
    return 'month'
}

export const TABS: ReportTab[] = ['overview', 'cashflow', 'expenses', 'income', 'networth']
