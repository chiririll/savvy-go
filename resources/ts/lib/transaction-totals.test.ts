import { describe, expect, it } from 'vitest'
import { largestTotals, netTotalsByCurrency } from './transaction-totals'

describe('largestTotals', () => {
    const base = { code: 'USD', decimals: 2, isBase: true, rate: 1 } as never
    const eur = { code: 'EUR', decimals: 2, isBase: false, rate: 1.1 } as never
    const jpy = { code: 'JPY', decimals: 0, isBase: false, rate: 0.007 } as never

    it('ranks by absolute value in the base currency, not by the raw number', () => {
        const totals = [
            { currency: jpy, amount: -2000 },
            { currency: base, amount: 20 },
            { currency: eur, amount: -30 },
        ]
        const { shown, hidden } = largestTotals(totals, 2)
        expect(shown.map((t) => t.currency)).toEqual([eur, base])
        expect(hidden).toBe(1)
    })

    it('hides nothing when everything fits', () => {
        expect(largestTotals([{ currency: base, amount: 1 }], 2)).toEqual({
            shown: [{ currency: base, amount: 1 }],
            hidden: 0,
        })
    })
})

const usd = { code: 'USD', decimals: 2 }
const jpy = { code: 'JPY', decimals: 0 }

function tx(type: string, amount: number, currency: { code: string; decimals: number } | null, status = 'confirmed') {
    return { type, status, amount, account: { currency } } as never
}

describe('netTotalsByCurrency', () => {
    it('nets income against expense in the same currency without float drift', () => {
        const result = netTotalsByCurrency([tx('income', 0.3, usd), tx('expense', 0.1, usd), tx('expense', 0.1, usd)])
        expect(result).toEqual([{ currency: usd, amount: 0.1 }])
    })

    it('keeps currencies apart, in order of first appearance', () => {
        const result = netTotalsByCurrency([tx('expense', 500, jpy), tx('income', 12.5, usd), tx('expense', 100, jpy)])
        expect(result).toEqual([
            { currency: jpy, amount: -600 },
            { currency: usd, amount: 12.5 },
        ])
    })

    it('ignores skipped rows, plain transfers and accounts without a currency', () => {
        const result = netTotalsByCurrency([
            tx('expense', 10, usd, 'skipped'),
            tx('transfer', 10, usd),
            tx('expense', 10, null),
        ])
        expect(result).toEqual([])
    })

    it('counts debt and one-sided transfer legs by direction', () => {
        const result = netTotalsByCurrency([tx('transfer_in', 20, usd), tx('debt_lend', 5, usd)])
        expect(result).toEqual([{ currency: usd, amount: 15 }])
    })
})
