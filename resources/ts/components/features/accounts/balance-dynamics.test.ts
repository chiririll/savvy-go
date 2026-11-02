import { describe, expect, it } from 'vitest'

import type { ChartTheme } from '@/lib/chart-theme'
import { axisCurrencyOf, balanceOption, dateLabeler, lastAmount, seriesKey, type LabeledSeries } from './balance-dynamics'

const theme: ChartTheme = {
    isDark: false,
    text: '#000',
    textStrong: '#000',
    axisLine: '#000',
    splitLine: '#000',
    surface: '#fff',
    tooltip: { backgroundColor: '#fff', borderColor: '#000', textStyle: { color: '#000' } },
}

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

describe('balance tooltip', () => {
    const tooltip = (hovered: number | null) => {
        const cash = series('USD', { key: 'account-1', label: 'Cash', data: [0, 10] })
        const bank = series('USD', { key: 'account-2', label: 'Bank', data: [5, 0] })
        const option = balanceOption({
            dates: ['2026-03-01', '2026-03-02'],
            series: [cash, bank],
            hiddenKeys: new Set(),
            baseCurrency: 'USD',
            theme,
            getHoveredIndex: () => hovered,
        })
        const formatter = option.tooltip.formatter
        return (dataIndex: number) =>
            formatter([
                { seriesName: 'Cash', seriesIndex: 0, dataIndex, color: '#0f0' },
                { seriesName: 'Bank', seriesIndex: 1, dataIndex, color: '#00f' },
            ])
    }

    it('lists only series with a non-zero value on that day', () => {
        const at = tooltip(null)
        expect(at(0)).toContain('Bank')
        expect(at(0)).not.toContain('Cash')
        expect(at(1)).toContain('Cash')
        expect(at(1)).not.toContain('Bank')
    })

    it('greys out rows other than the hovered line', () => {
        expect(tooltip(0)(1)).toContain('opacity:1')
        expect(tooltip(0)(1)).not.toContain('opacity:0.4')
        expect(tooltip(1)(1)).toContain('opacity:0.4')
    })
})
