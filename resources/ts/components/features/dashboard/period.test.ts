import { describe, expect, it } from 'vitest'

import { getPresetDates, isValidDateRange, percentChange, resolvePeriodDates, toReportFilters } from './period'

const NOW = new Date(2026, 4, 15) // 15 May 2026

describe('getPresetDates', () => {
    it.each([
        ['this_month', '2026-05-01', '2026-05-31'],
        ['last_month', '2026-04-01', '2026-04-30'],
        ['last_3_months', '2026-03-01', '2026-05-31'],
        ['last_6_months', '2025-12-01', '2026-05-31'],
        ['this_year', '2026-01-01', '2026-12-31'],
        ['last_30_days', '2026-04-16', '2026-05-15'],
    ] as const)('%s', (preset, start, end) => {
        expect(getPresetDates(preset, NOW)).toEqual({ start_date: start, end_date: end })
    })

    it('rolls January back into the previous year', () => {
        expect(getPresetDates('last_month', new Date(2026, 0, 10))).toEqual({
            start_date: '2025-12-01',
            end_date: '2025-12-31',
        })
    })
})

describe('isValidDateRange', () => {
    it.each([
        ['2026-01-02', '2026-01-09', true],
        ['2026-01-02', '2026-01-02', true],
        ['2026-02-01', '2026-01-09', false],
        ['2026-01-02', '', false],
        ['', '', false],
    ])('%s to %s is %s', (start, end, valid) => {
        expect(isValidDateRange(start, end)).toBe(valid)
    })
})

describe('resolvePeriodDates', () => {
    it('uses a complete custom range', () => {
        expect(resolvePeriodDates('custom', '2026-01-02', '2026-01-09')).toEqual({
            start_date: '2026-01-02',
            end_date: '2026-01-09',
        })
    })

    it('does not query a custom range that ends before it starts', () => {
        expect(resolvePeriodDates('custom', '2026-02-01', '2026-01-09')).toEqual(getPresetDates('last_30_days'))
    })

    it('falls back to 30 days while a custom range is incomplete', () => {
        expect(resolvePeriodDates('custom', '2026-01-02', '')).toEqual(getPresetDates('last_30_days'))
    })
})

describe('toReportFilters', () => {
    it('maps a month preset to a month filter', () => {
        const filters = toReportFilters('last_month', '', '', NOW)
        expect(filters.periodType).toBe('month')
        expect(filters.selectedMonth).toBe('2026-04')
    })

    it('maps this year to a year filter', () => {
        expect(toReportFilters('this_year', '', '', NOW)).toMatchObject({ periodType: 'year', selectedYear: '2026' })
    })

    it('treats the other presets as a custom range', () => {
        expect(toReportFilters('last_3_months', '', '', NOW).periodType).toBe('custom')
    })
})

describe('percentChange', () => {
    it('is null without a previous value', () => {
        expect(percentChange(10, null)).toBeNull()
        expect(percentChange(10, undefined)).toBeNull()
    })

    it('handles a zero base', () => {
        expect(percentChange(0, 0)).toBe(0)
        expect(percentChange(5, 0)).toBe(100)
    })

    it('measures against the magnitude of the previous value', () => {
        expect(percentChange(150, 100)).toBe(50)
        expect(percentChange(50, 100)).toBe(-50)
        expect(percentChange(-50, -100)).toBe(50)
    })
})
