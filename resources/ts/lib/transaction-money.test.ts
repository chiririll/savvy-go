import { beforeEach, describe, expect, it } from 'vitest'

import { useCurrenciesStore } from '@/stores/currencies'
import { accounts, categories, currencies, line, payment, transaction } from '@/test/fixtures'
import { lineCurrency, mainAmount, netByCurrency, primaryCategory, spentByCurrency } from './transaction-money'

const [usd, eur, jpy] = currencies
const [cash, euroCard, yenWallet] = accounts

beforeEach(() => useCurrenciesStore.getState().setAll(currencies))

describe('netByCurrency', () => {
    it('is one total for one account', () => {
        expect(netByCurrency(transaction())).toEqual([{ currency: usd, amount: -42.5 }])
    })

    it('adds payments in the same currency in whole minor units', () => {
        const tx = transaction({
            payments: [payment({ amount: -0.1 }), payment({ amount: -0.2, account: cash, accountId: 9 })],
        })
        expect(netByCurrency(tx)).toEqual([{ currency: usd, amount: -0.3 }])
    })

    it('keeps a total per currency, and the change of an exchange as the opposite sign', () => {
        const tx = transaction({
            payments: [
                payment({ amount: -50 }),
                payment({ account: euroCard, accountId: 2, amount: 8 }),
                payment({ account: yenWallet, accountId: 3, amount: 1000 }),
            ],
        })
        expect(netByCurrency(tx)).toEqual([
            { currency: usd, amount: -50 },
            { currency: eur, amount: 8 },
            { currency: jpy, amount: 1000 },
        ])
    })

    it('skips a payment whose account is not loaded', () => {
        expect(netByCurrency(transaction({ payments: [payment({ account: null })] }))).toEqual([])
    })
})

describe('spentByCurrency', () => {
    it('reads the lines, not the payments (change is not spent)', () => {
        const tx = transaction({
            payments: [payment({ amount: -50 }), payment({ account: euroCard, accountId: 2, amount: 8 })],
            lines: [line({ amount: 40, currencyId: eur.id })],
        })
        expect(spentByCurrency(tx)).toEqual([{ currency: eur, amount: 40 }])
    })

    it('finds the currency of a line among the space currencies when no payment account has it', () => {
        const tx = transaction({ lines: [line({ currencyId: jpy.id, amount: 300 })] })
        expect(lineCurrency(tx, tx.lines[0])).toBe(jpy)
        expect(spentByCurrency(tx)).toEqual([{ currency: jpy, amount: 300 }])
    })

    it('adds lines of one currency exactly', () => {
        const tx = transaction({ lines: [line({ amount: 0.1 }), line({ seq: 1, amount: 0.2 })] })
        expect(spentByCurrency(tx)).toEqual([{ currency: usd, amount: 0.3 }])
    })
})

describe('mainAmount', () => {
    it('is the largest by value in the base currency, not by number', () => {
        const totals = [
            { currency: jpy, amount: 3000 }, // 15 USD
            { currency: eur, amount: 20 }, // 25 USD
            { currency: usd, amount: 10 },
        ]
        expect(mainAmount(totals)).toBe(totals[1])
    })

    it('is null without totals', () => {
        expect(mainAmount([])).toBeNull()
    })
})

describe('primaryCategory', () => {
    const [groceries, salary, transport] = categories

    it('is the category of the largest line by base value', () => {
        const tx = transaction({
            lines: [
                line({ category: groceries, amount: 5 }),
                line({ seq: 1, category: transport, categoryId: transport.id, amount: 20 }),
            ],
        })
        expect(primaryCategory(tx)).toBe(transport)
    })

    it('weighs lines in other currencies by rate', () => {
        const tx = transaction({
            payments: [payment(), payment({ account: yenWallet, accountId: 3 })],
            lines: [
                line({ category: groceries, amount: 100 }), // 100 USD
                line({ seq: 1, category: salary, categoryId: salary.id, currencyId: jpy.id, amount: 1000 }), // 5 USD
            ],
        })
        expect(primaryCategory(tx)).toBe(groceries)
    })

    it('takes the first line on a tie and skips uncategorised lines', () => {
        const tied = transaction({
            lines: [
                line({ category: null, categoryId: null, amount: 99 }),
                line({ seq: 1, category: groceries, amount: 10 }),
                line({ seq: 2, category: transport, categoryId: transport.id, amount: 10 }),
            ],
        })
        expect(primaryCategory(tied)).toBe(groceries)
        expect(primaryCategory(transaction({ lines: [] }))).toBeNull()
    })
})
