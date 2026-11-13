import { describe, expect, it } from 'vitest'
import type { ReconciliationDay, ReconciliationPoint } from '@/types'
import { filterDaysByAccounts, loadedRange, mergeReconciliationDays,
    reconciliationRange, reconciledLabel, signedAmount } from './reconciliation-days'

const point = (accountId: number, differenceBase: number | null): ReconciliationPoint => ({
    id: accountId,
    accountId,
    date: '2026-05-02',
    actualBalance: 0,
    expectedBalance: 0,
    difference: differenceBase ?? 0,
    note: null,
    differenceBase,
})

const day = (date: string, points: ReconciliationPoint[] = [point(1, 5)]): ReconciliationDay => ({
    date,
    differenceBase: 5,
    unconverted: false,
    points,
})

const group = (date: string | null) => ({ date, items: [1] })

describe('filterDaysByAccounts', () => {
    it('keeps the server days when no account is chosen', () => {
        const days = [day('2026-05-02')]
        expect(filterDaysByAccounts(days, [], 2)).toEqual(days)
    })

    it('sums the kept points exactly and drops empty days', () => {
        const days = [day('2026-05-02', [point(1, 0.1), point(2, 0.2), point(3, 9)]), day('2026-05-03', [point(3, 1)])]
        const result = filterDaysByAccounts(days, [1, 2], 2)
        expect(result).toHaveLength(1)
        expect(result[0].differenceBase).toBe(0.3)
        expect(result[0].unconverted).toBe(false)
    })

    it('marks a point without a base value as unconverted', () => {
        const result = filterDaysByAccounts([day('2026-05-02', [point(1, null), point(2, 2)])], [1, 2], 2)
        expect(result[0]).toMatchObject({ differenceBase: 2, unconverted: true })
    })
})

describe('mergeReconciliationDays', () => {
    it('attaches a day to the group of its date', () => {
        const merged = mergeReconciliationDays([group('2026-05-04'), group('2026-05-01')], [day('2026-05-01')])
        expect(merged.map((g) => g.reconciliation?.date)).toEqual([undefined, '2026-05-01'])
    })

    it('adds a day without transactions inside the range, newest first', () => {
        const merged = mergeReconciliationDays([group('2026-05-04'), group('2026-05-01')], [day('2026-05-02')])
        expect(merged.map((g) => g.date)).toEqual(['2026-05-04', '2026-05-02', '2026-05-01'])
        expect(merged[1].items).toEqual([])
    })

    it('keeps oldest first order and leaves out days beyond the loaded range', () => {
        const merged = mergeReconciliationDays(
            [group('2026-05-01'), group('2026-05-04')],
            [day('2026-05-02'), day('2026-04-20'), day('2026-06-01')],
        )
        expect(merged.map((g) => g.date)).toEqual(['2026-05-01', '2026-05-02', '2026-05-04'])
    })

    it('keeps undated groups last', () => {
        const merged = mergeReconciliationDays([group('2026-05-04'), group(null), group('2026-05-01')], [day('2026-05-02')])
        expect(merged.map((g) => g.date)).toEqual(['2026-05-04', '2026-05-02', '2026-05-01', null])
    })
})

describe('loadedRange', () => {
    it('is the span of the dated groups', () => {
        expect(loadedRange([group('2026-05-04 10:00:00'), group(null), group('2026-05-01')])).toEqual({
            start: '2026-05-01',
            end: '2026-05-04',
        })
        expect(loadedRange([group(null)])).toBeNull()
    })
})

describe('reconciledLabel', () => {
    const format = (value: number) => `${value}$`
    it('says nothing of the amount when the day is exact', () => {
        expect(reconciledLabel({ ...day('d'), differenceBase: 0 }, format)).toEqual({ amount: null, tone: 'calm' })
    })
    it('signs the amount and tones it', () => {
        expect(reconciledLabel({ ...day('d'), differenceBase: 1500 }, format)).toEqual({ amount: '+1500$', tone: 'positive' })
        expect(reconciledLabel({ ...day('d'), differenceBase: -2 }, format)).toEqual({ amount: '-2$', tone: 'negative' })
    })
})

describe('signedAmount', () => {
    it('puts a plus on a gain only', () => {
        expect(signedAmount(3, String)).toBe('+3')
        expect(signedAmount(-3, String)).toBe('-3')
        expect(signedAmount(0, String)).toBe('0')
    })

    it('uses the given direction instead of guessing it from the groups', () => {
        const days = [day('2026-05-03')]
        const oldestFirst = mergeReconciliationDays([group('2026-05-01'), group('2026-05-05')], days, false)
        expect(oldestFirst.map((g) => g.date)).toEqual(['2026-05-01', '2026-05-03', '2026-05-05'])
        const newestFirst = mergeReconciliationDays([group('2026-05-01'), group('2026-05-05')], days, true)
        expect(newestFirst.map((g) => g.date)).toEqual(['2026-05-05', '2026-05-03', '2026-05-01'])
    })
})

describe('reconciliationRange', () => {
    const loaded = { start: '2026-05-01', end: '2026-05-04' }
    it('opens the newest end, and the oldest once the last page is in', () => {
        expect(reconciliationRange(loaded, true, true)).toEqual({ start: '2026-05-01', end: '2999-12-31' })
        expect(reconciliationRange(loaded, true, false)).toEqual({ start: '1900-01-01', end: '2999-12-31' })
    })
    it('opens the oldest end of an oldest first list', () => {
        expect(reconciliationRange(loaded, false, true)).toEqual({ start: '1900-01-01', end: '2026-05-04' })
    })
    it('covers everything when nothing is loaded', () => {
        expect(reconciliationRange(null, true, false)).toEqual({ start: '1900-01-01', end: '2999-12-31' })
    })
})

describe('mergeReconciliationDays with a range', () => {
    it('shows a day with no transactions beyond them, or with no transactions at all', () => {
        const range = { start: '1900-01-01', end: '2999-12-31' }
        expect(mergeReconciliationDays([group('2026-05-01')], [day('2026-05-09')], true, range).map((g) => g.date)).toEqual(['2026-05-09', '2026-05-01'])
        expect(mergeReconciliationDays([], [day('2026-05-09')], true, range).map((g) => g.date)).toEqual(['2026-05-09'])
    })
})
