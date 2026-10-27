import { describe, expect, it } from 'vitest'
import { transaction } from '@/test/fixtures'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account, Category, Transaction, TransactionGroup } from '@/types'
import { buildPaymentLegs, isSplitPayment, paymentFormValues } from './payment-input'

const account = (id: number, code: string, decimals = 2) => ({ id, currency: { code, decimals } }) as Account
const eur = account(1, 'EUR')
const rsd = account(2, 'RSD')
const card = account(3, 'EUR')
const accounts = [eur, rsd, card]

const base: TransactionFormValues = {
    type: 'expense',
    account_id: 1,
    category_id: 7,
    amount: 450,
    date: '2026-10-17',
    description: 'Rent',
    tag_ids: [5],
}

describe('isSplitPayment', () => {
    it('is true once there is a row besides the first', () => {
        expect(isSplitPayment({})).toBe(false)
        expect(isSplitPayment({ extra_payments: [], changes: [] })).toBe(false)
        expect(isSplitPayment({ extra_payments: [{}] })).toBe(true)
        expect(isSplitPayment({ changes: [{}] })).toBe(true)
    })
})

describe('buildPaymentLegs', () => {
    it('makes an expense on each account that paid', () => {
        const result = buildPaymentLegs(
            { ...base, amount: 15.5, extra_payments: [{ account_id: 3, amount: 24.5 }] },
            accounts,
        )
        expect(result.ok && result.legs.map((leg) => [leg.type, leg.account_id, leg.amount])).toEqual([
            ['expense', 1, 15.5],
            ['expense', 3, 24.5],
        ])
    })

    it('makes an expense of what was handed over and an income of each change', () => {
        const result = buildPaymentLegs(
            { ...base, amount: 500, changes: [{ account_id: 2, amount: 5900 }] },
            accounts,
        )
        expect(result.ok && result.legs).toEqual([
            expect.objectContaining({ type: 'expense', account_id: 1, category_id: 7, amount: 500 }),
            expect.objectContaining({ type: 'income', account_id: 2, category_id: 7, amount: 5900 }),
        ])
    })

    it('says so when an account is not known', () => {
        expect(buildPaymentLegs({ ...base, extra_payments: [{ account_id: 99, amount: 5 }] }, accounts))
            .toEqual({ ok: false, error: 'unknownAccount' })
        expect(buildPaymentLegs({ ...base, changes: [{ account_id: 99, amount: 5 }] }, accounts))
            .toEqual({ ok: false, error: 'unknownAccount' })
    })
})

const category = { id: 7 } as Category
const leg = (overrides: Partial<Transaction>): Transaction =>
    transaction({
        groupUuid: 'g1', date: '2026-10-17', description: 'Rent', category, tags: [{ id: 5 } as never], ...overrides,
    })

describe('paymentFormValues', () => {
    it('reads a payment settled with several accounts back into the form', () => {
        const group: TransactionGroup = {
            uuid: 'g1',
            legs: [
                leg({ id: 1, type: 'expense', amount: 15.5, account: eur }),
                leg({ id: 2, type: 'expense', amount: 24.5, account: card }),
            ],
        }
        expect(paymentFormValues(group)).toMatchObject({
            type: 'expense', account_id: 1, amount: 15.5, category_id: 7,
            date: '2026-10-17', description: 'Rent', tag_ids: [5],
            extra_payments: [{ account_id: 3, amount: 24.5 }], changes: [],
        })
    })

    it('reads the incomes after the expenses back as change', () => {
        const group: TransactionGroup = {
            uuid: 'g1',
            legs: [
                leg({ id: 1, type: 'expense', amount: 500, account: eur }),
                leg({ id: 2, type: 'income', amount: 5900, account: rsd }),
                leg({ id: 3, type: 'income', amount: 2, account: card }),
            ],
        }
        expect(paymentFormValues(group)).toMatchObject({
            account_id: 1, amount: 500, extra_payments: [],
            changes: [{ account_id: 2, amount: 5900 }, { account_id: 3, amount: 2 }],
        })
    })

    it('gives up on shapes the form cannot show', () => {
        const exchange = leg({ id: 2, type: 'transfer', amount: 50, toAmount: 5900, account: eur, toAccount: rsd })
        const income = leg({ id: 2, type: 'income', amount: 5, account: eur })
        const expense = (id: number) => leg({ id, type: 'expense', account: eur })
        for (const legs of [
            [expense(1), exchange],
            [expense(1)],
            [income, expense(2)],
            [expense(1), income, expense(3)],
        ]) {
            expect(paymentFormValues({ uuid: 'g1', legs })).toBeNull()
        }
    })

    it('reads back what buildPaymentLegs made', () => {
        const values: TransactionFormValues = {
            ...base, amount: 500, extra_payments: [{ account_id: 3, amount: 10 }], changes: [{ account_id: 2, amount: 5900 }],
        }
        const made = buildPaymentLegs(values, accounts)
        if (!made.ok) throw new Error('not built')
        const byId = new Map(accounts.map((a) => [a.id, a]))
        const group: TransactionGroup = {
            uuid: 'g1',
            legs: made.legs.map((l, i) =>
                leg({ id: i + 1, type: l.type, amount: l.amount, account: byId.get(l.account_id)! })),
        }
        expect(paymentFormValues(group)).toMatchObject({
            account_id: 1, amount: 500,
            extra_payments: [{ account_id: 3, amount: 10 }], changes: [{ account_id: 2, amount: 5900 }],
        })
    })
})
