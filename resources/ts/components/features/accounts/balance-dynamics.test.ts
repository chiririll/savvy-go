import { describe, expect, it } from 'vitest'

import { axisCurrencyOf, dateLabeler, lastAmount, seriesKey, type LabeledSeries } from './balance-dynamics'

const series = (currency: string | null, extra: Partial<LabeledSeries> = {}): LabeledSeries =>
    ({ type: 'account', id: 1, data: [1, 2], currency, key: 'k', label: 'l', color: '#000', ...extra }) as LabeledSeries

describe('balance dynamics helpers', () => {
    it('keys the total apart from accounts', () => {
        expect(seriesKey(series('USD', { type: 'total' }))).toBe('total')
        expect(seriesKey(series('USD', { id: 7 }))).toBe('account-7')
    })

    it('reads the last amount in the native currency when there is one', () => {
        expect(lastAmount(series('USD'))).toBe(2)
        expect(lastAmount(series('EUR', { native_data: [5, 9] }))).toBe(9)
        expect(lastAmount(series('EUR', { data: [] }))).toBe(0)
    })

    it('labels short ranges by day and long ones by month', () => {
        expect(dateLabeler(30)('2026-03-05T12:00:00')).toBe('5.03')
        expect(dateLabeler(120)('2026-03-05T12:00:00')).toBe('03.26')
    })

    it('uses a shared currency on the axis, else the base currency', () => {
        expect(axisCurrencyOf([series('EUR'), series('EUR')], 'USD')).toBe('EUR')
        expect(axisCurrencyOf([series('EUR'), series('GBP')], 'USD')).toBe('USD')
        expect(axisCurrencyOf([], 'USD')).toBe('USD')
    })
})
