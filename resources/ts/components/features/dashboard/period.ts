import { addDaysLocal, formatDateLocal, formatYearMonth } from '@/lib/utils'
import { DEFAULT_FILTERS, type ReportFilters } from '@/pages/reports/types'

export type PeriodPreset =
    | 'last_30_days'
    | 'this_month'
    | 'last_month'
    | 'last_3_months'
    | 'last_6_months'
    | 'this_year'
    | 'custom'

export interface DateRange {
    start_date: string
    end_date: string
}

/** Whole calendar months, as offsets from the current month (first month, last month). */
const MONTH_SPANS: Partial<Record<PeriodPreset, [number, number]>> = {
    this_month: [0, 0],
    last_month: [-1, -1],
    last_3_months: [-2, 0],
    last_6_months: [-5, 0],
}

export function getPresetDates(preset: PeriodPreset, now = new Date()): DateRange {
    const year = now.getFullYear()
    const month = now.getMonth()

    const span = MONTH_SPANS[preset]
    if (span) {
        return {
            start_date: formatDateLocal(new Date(year, month + span[0], 1)),
            end_date: formatDateLocal(new Date(year, month + span[1] + 1, 0)),
        }
    }
    if (preset === 'this_year') {
        return {
            start_date: formatDateLocal(new Date(year, 0, 1)),
            end_date: formatDateLocal(new Date(year, 11, 31)),
        }
    }
    return { start_date: addDaysLocal(now, -29), end_date: formatDateLocal(now) }
}

/** True when both dates are set and the range does not end before it starts (ISO dates compare as text). */
export function isValidDateRange(start: string, end: string): boolean {
    return Boolean(start && end && start <= end)
}

/** The range the dashboard shows: the picked dates for a valid custom range, else the preset's. */
export function resolvePeriodDates(preset: PeriodPreset, customStart: string, customEnd: string): DateRange {
    if (preset === 'custom' && isValidDateRange(customStart, customEnd)) {
        return { start_date: customStart, end_date: customEnd }
    }
    return getPresetDates(preset)
}

/** True when today falls inside the range. Budgets and debts show the present, so only such a period fits them. */
export function includesToday(range: DateRange, now = new Date()): boolean {
    const today = formatDateLocal(now)
    return range.start_date <= today && today <= range.end_date
}

export function toReportFilters(
    preset: PeriodPreset,
    customStartDate: string,
    customEndDate: string,
    now = new Date(),
): ReportFilters {
    const dates = resolvePeriodDates(preset, customStartDate, customEndDate)

    const filters: ReportFilters = {
        ...DEFAULT_FILTERS,
        customStartDate: dates.start_date,
        customEndDate: dates.end_date,
        compareWith: 'previous_period',
    }

    switch (preset) {
        case 'last_30_days':
            return { ...filters, periodType: 'last_30_days' }
        case 'this_month':
            return { ...filters, periodType: 'month', selectedMonth: formatYearMonth(now) }
        case 'last_month':
            return {
                ...filters,
                periodType: 'month',
                selectedMonth: formatYearMonth(new Date(now.getFullYear(), now.getMonth() - 1, 1)),
            }
        case 'this_year':
            return { ...filters, periodType: 'year', selectedYear: String(now.getFullYear()) }
        default:
            return { ...filters, periodType: 'custom' }
    }
}

/** Change from `previous` to `current` in percent; null when there is nothing to compare with. */
export function percentChange(current: number, previous: number | null | undefined): number | null {
    if (previous == null) return null
    if (previous === 0) return current === 0 ? 0 : 100
    return ((current - previous) / Math.abs(previous)) * 100
}
