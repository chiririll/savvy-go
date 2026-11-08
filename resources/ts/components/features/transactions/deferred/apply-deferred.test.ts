import { describe, expect, it } from 'vitest'

import { accounts, line, payment, pendingTransaction } from '@/test/fixtures'
import type { Transaction } from '@/types'
import {
    amountsChanged,
    amountsValid,
    applyWarnings,
    buildApplyOptions,
    initialAmounts,
    isOccurrence,
    linesFollowPayments,
    selectedApplyDate,
} from './apply-deferred'

const occurrence = (overrides: Partial<Transaction> = {}) =>
    pendingTransaction({
        recurringTransactionId: 5,
        restrictions: [{ reason: 'recurring_pending', message: '' }],
        actions: { edit: false, editDetails: false, duplicate: false, delete: true, confirm: true, skip: true },
        ...overrides,
    })

// Rent: 1100 from the dollar account, one line.
const rent = () =>
    occurrence({ payments: [payment({ amount: -1100 })], lines: [line({ amount: 1100 })] })

// Two currencies: 100 EUR leave the euro card (payments[0]), 90 USD arrive... as an expense with two payments.
const twoCurrencies = () =>
    occurrence({
        payments: [
            payment({ amount: -30 }),
            payment({ account: accounts[1], accountId: 2, amount: -10 }),
        ],
        lines: [line({ seq: 0, amount: 30, currencyId: 1 }), line({ seq: 1, amount: 10, currencyId: 2 })],
    })

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

describe('isOccurrence', () => {
    it('is true only for the open occurrence of a recurring rule', () => {
        expect(isOccurrence(rent())).toBe(true)
        expect(isOccurrence(pendingTransaction())).toBe(false)
    })
})

describe('linesFollowPayments', () => {
    it('is true for one line in the currency of every payment', () => {
        expect(linesFollowPayments(rent())).toBe(true)
    })

    it('is false for several lines, or payments in other currencies than the line', () => {
        expect(linesFollowPayments(twoCurrencies())).toBe(false)
        const foreign = occurrence({
            payments: [payment({ account: accounts[1], accountId: 2, amount: -10 })],
            lines: [line({ amount: 10, currencyId: 1 })],
        })
        expect(linesFollowPayments(foreign)).toBe(false)
    })

    it('is false for a transaction without lines', () => {
        expect(linesFollowPayments(occurrence({ lines: [] }))).toBe(false)
    })
})

describe('initialAmounts', () => {
    it('takes the amounts as they are, without sign, in the order of payments and lines', () => {
        expect(initialAmounts(twoCurrencies())).toEqual({ payments: ['30', '10'], lines: ['30', '10'] })
    })

    it('orders lines by their sequence', () => {
        const tx = occurrence({
            payments: [payment({ amount: -30 })],
            lines: [line({ seq: 1, amount: 20 }), line({ seq: 0, amount: 10 })],
        })
        expect(initialAmounts(tx).lines).toEqual(['10', '20'])
    })
})

describe('amountsChanged', () => {
    it('is false for the amounts it started with, however they are written', () => {
        const tx = rent()
        expect(amountsChanged(tx, { payments: ['1100.00'], lines: ['1100'] })).toBe(false)
    })

    it('is true when a payment or a line differs', () => {
        expect(amountsChanged(rent(), { payments: ['1090'], lines: ['1100'] })).toBe(true)
        expect(amountsChanged(twoCurrencies(), { payments: ['30', '10'], lines: ['30', '11'] })).toBe(true)
    })

    it('ignores the line of a transaction whose line follows the payments', () => {
        expect(amountsChanged(rent(), { payments: ['1100'], lines: ['5'] })).toBe(false)
    })
})

describe('amountsValid', () => {
    it('accepts anything for a transaction that is not an occurrence', () => {
        expect(amountsValid(pendingTransaction(), { payments: ['0'], lines: ['0'] })).toBe(true)
    })

    it('wants every payment above zero', () => {
        expect(amountsValid(rent(), { payments: ['1090'], lines: ['1100'] })).toBe(true)
        expect(amountsValid(rent(), { payments: ['0'], lines: ['1100'] })).toBe(false)
        expect(amountsValid(rent(), { payments: [''], lines: ['1100'] })).toBe(false)
    })

    it('wants every written line above zero, but not the one that follows the payments', () => {
        expect(amountsValid(twoCurrencies(), { payments: ['30', '10'], lines: ['30', '0'] })).toBe(false)
        expect(amountsValid(rent(), { payments: ['1100'], lines: [''] })).toBe(true)
    })
})

describe('buildApplyOptions', () => {
    it('sends only the date for a transaction that is not an occurrence', () => {
        const tx = pendingTransaction()
        expect(buildApplyOptions(tx, '2026-05-02', initialAmounts(tx))).toEqual({ date: '2026-05-02' })
    })

    it('sends only the date for an occurrence whose amounts stand', () => {
        const tx = rent()
        expect(buildApplyOptions(tx, '2026-05-02', initialAmounts(tx))).toEqual({ date: '2026-05-02' })
    })

    it('sends the payments with their sign, and the line that follows them', () => {
        const options = buildApplyOptions(rent(), '2026-05-02', { payments: ['1090.5'], lines: ['1100'] })
        expect(options).toEqual({
            date: '2026-05-02',
            payments: [{ account_id: 1, amount: -1090.5 }],
            lines: [{ category_id: 1, currency_id: 1, amount: 1090.5, note: null }],
        })
    })

    it('keeps the sign of change, and the note and category of a line', () => {
        const tx = occurrence({
            type: 'expense',
            payments: [payment({ amount: -100 }), payment({ account: accounts[1], accountId: 2, amount: 4 })],
            lines: [line({ amount: 96, note: '- milk', categoryId: 3 }), line({ seq: 1, amount: 2, currencyId: 2 })],
        })
        const options = buildApplyOptions(tx, '2026-05-02', { payments: ['100', '5'], lines: ['96', '3'] })
        expect(options.payments).toEqual([
            { account_id: 1, amount: -100 },
            { account_id: 2, amount: 5 },
        ])
        expect(options.lines).toEqual([
            { category_id: 3, currency_id: 1, amount: 96, note: '- milk' },
            { category_id: 1, currency_id: 2, amount: 3, note: null },
        ])
    })

    it('sends payments only when the transaction has no lines to correct', () => {
        const tx = occurrence({
            type: 'transfer',
            payments: [payment({ amount: -30 }), payment({ account: accounts[1], accountId: 2, amount: 24 })],
            lines: [],
        })
        expect(buildApplyOptions(tx, '2026-05-02', { payments: ['30', '25'], lines: [] })).toEqual({
            date: '2026-05-02',
            payments: [
                { account_id: 1, amount: -30 },
                { account_id: 2, amount: 25 },
            ],
        })
    })

    it('rounds to the scale of the account currency', () => {
        const options = buildApplyOptions(rent(), '2026-05-02', { payments: ['1090.456'], lines: ['1100'] })
        expect(options.payments).toEqual([{ account_id: 1, amount: -1090.46 }])
    })
})

describe('applyWarnings', () => {
    const poor = (balance: number) => ({ ...accounts[0], currentBalance: balance })

    it('warns about an account the payments of the transaction take below zero', () => {
        const tx = pendingTransaction({ payments: [payment({ account: poor(50), amount: -80 })] })
        const warnings = applyWarnings(tx, initialAmounts(tx))
        expect(warnings).toHaveLength(1)
        expect(warnings[0]).toMatchObject({ currentBalance: 50, resultingBalance: -30 })
    })

    it('uses the real amounts of an occurrence', () => {
        const tx = occurrence({
            payments: [payment({ account: poor(100), amount: -80 })],
            lines: [line({ amount: 80 })],
        })
        expect(applyWarnings(tx, { payments: ['120'], lines: ['80'] })).toHaveLength(1)
        expect(applyWarnings(tx, { payments: ['60'], lines: ['80'] })).toHaveLength(0)
    })

    it('does not warn about money coming in', () => {
        const tx = pendingTransaction({ payments: [payment({ account: poor(-20), amount: 30 })] })
        expect(applyWarnings(tx, initialAmounts(tx))).toEqual([])
    })
})
