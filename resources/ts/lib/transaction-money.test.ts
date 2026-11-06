import { beforeEach, describe, expect, it } from 'vitest'

import { useCurrenciesStore } from '@/stores/currencies'
import { accounts, categories, currencies, line, payment, transaction } from '@/test/fixtures'
import {
    convertAmount,
    lineCurrency,
    mainAmount,
    netByCurrency,
    primaryCategory,
    rowAmount,
    spentByCurrency,
    transferSides,
} from './transaction-money'

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

describe('convertAmount', () => {
    it('goes through the base currency and rounds to the target decimals', () => {
        expect(convertAmount(10, eur, usd)).toBe(12.5)
        expect(convertAmount(100, usd, jpy)).toBe(20000)
        expect(convertAmount(1000, jpy, eur)).toBe(4)
    })

    it('leaves the same currency alone', () => {
        expect(convertAmount(1.234, usd, usd)).toBe(1.234)
    })
})

describe('rowAmount', () => {
    it('is the lines of one currency, unsigned, not converted', () => {
        expect(rowAmount(transaction())).toEqual({
            main: { currency: usd, amount: 42.5 },
            totals: [{ currency: usd, amount: 42.5 }],
            converted: false,
        })
    })

    it('ignores change: it is only in the payments', () => {
        const tx = transaction({
            payments: [payment({ amount: -50 }), payment({ account: cash, accountId: 9, amount: 7.5 })],
        })
        expect(rowAmount(tx)?.main).toEqual({ currency: usd, amount: 42.5 })
    })

    it('shows one value in the first currency, with the exact totals behind it', () => {
        const tx = transaction({
            payments: [payment({ amount: -30 }), payment({ account: euroCard, accountId: 2, amount: -10 })],
            lines: [
                line({ amount: 30, currencyId: usd.id }),
                line({ seq: 1, amount: 10, currencyId: eur.id, categoryId: categories[2].id, category: categories[2] }),
            ],
        })
        expect(rowAmount(tx)).toEqual({
            main: { currency: usd, amount: 42.5 },
            totals: [
                { currency: usd, amount: 30 },
                { currency: eur, amount: 10 },
            ],
            converted: true,
        })
    })

    it('falls back to the payments on ordinary accounts when there are no lines', () => {
        const debt = { ...accounts[0], id: 7, type: 'debt' as const }
        const tx = transaction({
            lines: [],
            payments: [payment({ amount: -20 }), payment({ account: debt, accountId: 7, amount: 20 })],
        })
        expect(rowAmount(tx)?.main).toEqual({ currency: usd, amount: 20 })
    })

    it('is null with nothing to show', () => {
        expect(rowAmount(transaction({ lines: [], payments: [] }))).toBeNull()
    })
})

describe('transferSides', () => {
    it('splits what left from what arrived, each in its own currency', () => {
        const tx = transaction({
            type: 'transfer',
            payments: [payment({ amount: -100 }), payment({ account: yenWallet, accountId: 3, amount: 20000 })],
            lines: [],
        })
        const { sent, received } = transferSides(tx)
        expect(sent?.main).toEqual({ currency: usd, amount: 100 })
        expect(received?.main).toEqual({ currency: jpy, amount: 20000 })
    })

    it('leaves out the space account of a transfer to another space', () => {
        const space = { ...accounts[0], id: 8, type: 'space' as const }
        const tx = transaction({
            type: 'transfer',
            payments: [payment({ amount: -100 }), payment({ account: space, accountId: 8, amount: 100 })],
            lines: [],
        })
        const { sent, received } = transferSides(tx)
        expect(sent?.main.amount).toBe(100)
        expect(received).toBeNull()
    })
})
