import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
    addDaysLocal,
    formatDateLocal,
    formatTransactionGroupHeading,
    formatYearMonth,
    groupByDateKey,
    isDateInFuture,
    isDateOverdue,
    parseDateKey,
    pendingDateClassName,
} from './dates'

// 2026-03-15 at local noon, away from day boundaries in any time zone.
beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 2, 15, 12, 0, 0))
})
afterEach(() => vi.useRealTimers())

describe('parseDateKey', () => {
    it('builds a local date without a UTC shift', () => {
        const d = parseDateKey('2026-01-31')
        expect([d.getFullYear(), d.getMonth(), d.getDate()]).toEqual([2026, 0, 31])
        expect(d.getHours()).toBe(0)
    })

    it('ignores a trailing time component', () => {
        expect(formatDateLocal(parseDateKey('2026-01-31 23:59:59'))).toBe('2026-01-31')
    })

    it('defaults a missing month or day to 1', () => {
        expect(formatDateLocal(parseDateKey('2026'))).toBe('2026-01-01')
    })
})

describe('formatting', () => {
    it('pads month and day', () => {
        expect(formatDateLocal(new Date(2026, 0, 5))).toBe('2026-01-05')
        expect(formatYearMonth(new Date(2026, 8, 30))).toBe('2026-09')
    })

    it('defaults to today', () => {
        expect(formatDateLocal()).toBe('2026-03-15')
        expect(formatYearMonth()).toBe('2026-03')
    })
})

describe('addDaysLocal', () => {
    it('moves across month and year boundaries', () => {
        expect(addDaysLocal(new Date(2026, 0, 31), 1)).toBe('2026-02-01')
        expect(addDaysLocal(new Date(2026, 0, 1), -1)).toBe('2025-12-31')
        expect(addDaysLocal(new Date(2028, 1, 28), 1)).toBe('2028-02-29')
    })

    it('does not mutate its argument', () => {
        const d = new Date(2026, 2, 15)
        addDaysLocal(d, 10)
        expect(formatDateLocal(d)).toBe('2026-03-15')
    })
})

describe('isDateInFuture / isDateOverdue', () => {
    it('compares against today by day', () => {
        expect(isDateInFuture('2026-03-16')).toBe(true)
        expect(isDateInFuture('2026-03-15')).toBe(false)
        expect(isDateOverdue('2026-03-14')).toBe(true)
        expect(isDateOverdue('2026-03-15')).toBe(false)
    })

    it('is false for a missing date', () => {
        expect(isDateInFuture(null)).toBe(false)
        expect(isDateInFuture(undefined)).toBe(false)
        expect(isDateOverdue(null)).toBe(false)
        expect(isDateOverdue('')).toBe(false)
    })
})

describe('formatTransactionGroupHeading', () => {
    const labels = { today: 'Today', yesterday: 'Yesterday', tomorrow: 'Tomorrow', noDate: 'No date' }

    it('names the days around today', () => {
        expect(formatTransactionGroupHeading('2026-03-15', 'en-US', labels)).toBe('Today')
        expect(formatTransactionGroupHeading('2026-03-14', 'en-US', labels)).toBe('Yesterday')
        expect(formatTransactionGroupHeading('2026-03-16', 'en-US', labels)).toBe('Tomorrow')
        expect(formatTransactionGroupHeading(null, 'en-US', labels)).toBe('No date')
    })

    it('skips "tomorrow" when no label is given', () => {
        const withoutTomorrow = { today: 'Today', yesterday: 'Yesterday', noDate: 'No date' }
        expect(formatTransactionGroupHeading('2026-03-16', 'en-US', withoutTomorrow)).toBe('March 16')
    })

    it('shows the year only outside the current one', () => {
        expect(formatTransactionGroupHeading('2026-01-02', 'en-US', labels)).toBe('January 2')
        expect(formatTransactionGroupHeading('2025-12-31', 'en-US', labels)).toBe('December 31, 2025')
    })
})

describe('groupByDateKey', () => {
    it('groups by date in first-seen order, keeping item order', () => {
        const items = [
            { id: 1, date: '2026-03-02' },
            { id: 2, date: '2026-03-01' },
            { id: 3, date: '2026-03-02' },
            { id: 4, date: null },
            { id: 5, date: null },
        ]
        expect(groupByDateKey(items).map((g) => [g.date, g.items.map((i) => i.id)])).toEqual([
            ['2026-03-02', [1, 3]],
            ['2026-03-01', [2]],
            [null, [4, 5]],
        ])
    })

    it('returns no groups for no items', () => {
        expect(groupByDateKey([])).toEqual([])
    })
})

describe('pendingDateClassName', () => {
    it('colours by how close the date is', () => {
        expect(pendingDateClassName('2026-03-14')).toBe('text-red-600')
        expect(pendingDateClassName('2026-03-15')).toBe('text-yellow-600')
        expect(pendingDateClassName('2026-03-18')).toBe('text-yellow-600')
        expect(pendingDateClassName('2026-03-19')).toBe('text-muted-foreground')
        expect(pendingDateClassName(null)).toBe('text-muted-foreground')
    })
})
