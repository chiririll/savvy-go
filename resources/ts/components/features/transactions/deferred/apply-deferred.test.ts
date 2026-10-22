import { describe, expect, it } from 'vitest'

import { amountsValid, buildApplyOptions, selectedApplyDate } from './apply-deferred'

describe('selectedApplyDate', () => {
    it('picks the date of the chosen option', () => {
        expect(selectedApplyDate('today', '2026-05-02', '2026-04-01', '2026-03-03')).toBe('2026-05-02')
        expect(selectedApplyDate('original', '2026-05-02', '2026-04-01', '2026-03-03')).toBe('2026-04-01')
        expect(selectedApplyDate('other', '2026-05-02', '2026-04-01', '2026-03-03')).toBe('2026-03-03')
    })

    it('has no date when the original is missing', () => {
        expect(selectedApplyDate('original', '2026-05-02', null, '')).toBeNull()
    })
})

describe('buildApplyOptions', () => {
    const base = { date: '2026-05-02', crossCurrency: false, amount: 10, toAmount: 12 }

    it('sends only the date for a firm transaction', () => {
        expect(buildApplyOptions({ ...base, estimated: false })).toEqual({ date: '2026-05-02' })
    })

    it('sends the amount of an estimate, and the second one across currencies', () => {
        expect(buildApplyOptions({ ...base, estimated: true })).toEqual({ date: '2026-05-02', amount: 10 })
        expect(buildApplyOptions({ ...base, estimated: true, crossCurrency: true })).toEqual({
            date: '2026-05-02',
            amount: 10,
            to_amount: 12,
        })
    })
})

describe('amountsValid', () => {
    it('always accepts a firm transaction', () => {
        expect(amountsValid(false, true, 0, 0)).toBe(true)
    })

    it('wants a positive amount, and a positive second one across currencies', () => {
        expect(amountsValid(true, false, 0, 5)).toBe(false)
        expect(amountsValid(true, false, 5, 0)).toBe(true)
        expect(amountsValid(true, true, 5, 0)).toBe(false)
        expect(amountsValid(true, true, 5, 6)).toBe(true)
    })
})
