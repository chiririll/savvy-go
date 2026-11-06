import { describe, expect, it } from 'vitest'
import { currencies, accounts, line, payment, transaction } from '@/test/fixtures'
import type { Transaction } from '@/types'
import { useCurrenciesStore } from '@/stores/currencies'
import { largestTotals, netTotalsByCurrency } from './transaction-totals'

useCurrenciesStore.getState().setAll(currencies)

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

const [usd, eur, jpy] = currencies
const [cash, euroCard, yenWallet] = accounts
const spend = (type: Transaction['type'], amount: number, account = cash, status: Transaction['status'] = 'confirmed') =>
    transaction({
        type,
        status,
        payments: [payment({ account, accountId: account.id, amount })],
        lines: [line({ amount: Math.abs(amount), currencyId: account.currency?.id })],
    })

describe('netTotalsByCurrency', () => {
    it('nets income against expense in the same currency without float drift', () => {
        const result = netTotalsByCurrency([spend('income', 0.3), spend('expense', -0.1), spend('expense', -0.1)])
        expect(result).toEqual([{ currency: usd, amount: 0.1 }])
    })

    it('keeps currencies apart, in order of first appearance', () => {
        const result = netTotalsByCurrency([
            spend('expense', -500, yenWallet),
            spend('income', 12.5),
            spend('expense', -100, yenWallet),
        ])
        expect(result).toEqual([
            { currency: jpy, amount: -600 },
            { currency: usd, amount: 12.5 },
        ])
    })

    it('ignores skipped rows and plain transfers, but a transfer fee is spending', () => {
        const transfer = transaction({
            type: 'transfer',
            payments: [payment({ amount: -102 }), payment({ account: euroCard, accountId: 2, amount: 80 })],
            lines: [line({ amount: 2, category: null, categoryId: null })],
        })
        const plain = transaction({ type: 'transfer', payments: [payment({ amount: -10 })], lines: [] })
        const result = netTotalsByCurrency([spend('expense', -10, cash, 'skipped'), plain, transfer])
        expect(result).toEqual([{ currency: usd, amount: -2 }])
    })

    it('counts every currency of a row on its own', () => {
        const tx = transaction({
            payments: [payment({ amount: -30 }), payment({ account: euroCard, accountId: 2, amount: -10 })],
            lines: [line({ amount: 30, currencyId: usd.id }), line({ seq: 1, amount: 10, currencyId: eur.id })],
        })
        expect(netTotalsByCurrency([tx])).toEqual([
            { currency: usd, amount: -30 },
            { currency: eur, amount: -10 },
        ])
    })

    it('counts debt and one-sided transfer legs by direction', () => {
        const result = netTotalsByCurrency([spend('transfer_in', 20), spend('debt_lend', -5)])
        expect(result).toEqual([{ currency: usd, amount: 15 }])
    })
})
