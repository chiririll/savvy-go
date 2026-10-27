import { describe, expect, it } from 'vitest'
import { transaction } from '@/test/fixtures'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account, Category, Transaction, TransactionGroup } from '@/types'
import { buildPaymentLegs, paymentFormValues } from './payment-input'

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

describe('buildPaymentLegs', () => {
    it('makes an expense on each account of a payment settled with several', () => {
        const result = buildPaymentLegs(
            { ...base, amount: 15.5, payment_mode: 'several', extra_payments: [{ account_id: 3, amount: 24.5 }] },
            accounts,
        )
        expect(result.ok && result.legs.map((leg) => [leg.type, leg.account_id, leg.amount])).toEqual([
            ['expense', 1, 15.5],
            ['expense', 3, 24.5],
        ])
    })

    it('makes the expense and the exchange of a payment with change', () => {
        const result = buildPaymentLegs(
            { ...base, payment_mode: 'change', handed_over: 500, change_account_id: 2, change_amount: 5900 },
            accounts,
        )
        expect(result).toMatchObject({ ok: true, rate: 118 })
        expect(result.ok && result.legs).toEqual([
            expect.objectContaining({ type: 'expense', account_id: 1, category_id: 7, amount: 450 }),
            expect.objectContaining({ type: 'transfer', account_id: 1, to_account_id: 2, amount: 50, to_amount: 5900 }),
        ])
    })

    it('says so when change is in the currency paid with', () => {
        const result = buildPaymentLegs(
            { ...base, payment_mode: 'change', handed_over: 500, change_account_id: 3, change_amount: 50 },
            accounts,
        )
        expect(result).toEqual({ ok: false, error: 'sameCurrency' })
    })

    it('says so when an account is not known', () => {
        const result = buildPaymentLegs(
            { ...base, payment_mode: 'several', extra_payments: [{ account_id: 99, amount: 5 }] },
            accounts,
        )
        expect(result).toEqual({ ok: false, error: 'unknownAccount' })
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
            type: 'expense', payment_mode: 'several', account_id: 1, amount: 15.5, category_id: 7,
            date: '2026-10-17', description: 'Rent', tag_ids: [5],
            extra_payments: [{ account_id: 3, amount: 24.5 }],
        })
    })

    it('reads a payment with change back, handed over being price and surplus', () => {
        const group: TransactionGroup = {
            uuid: 'g1',
            legs: [
                leg({ id: 1, type: 'expense', amount: 450, account: eur }),
                leg({ id: 2, type: 'transfer', amount: 50, toAmount: 5900, account: eur, toAccount: rsd, category: null }),
            ],
        }
        expect(paymentFormValues(group)).toMatchObject({
            payment_mode: 'change', account_id: 1, amount: 450, handed_over: 500,
            change_account_id: 2, change_amount: 5900,
        })
    })

    it('rounds the handed over amount to the currency', () => {
        const group: TransactionGroup = {
            uuid: 'g1',
            legs: [
                leg({ id: 1, type: 'expense', amount: 0.1, account: eur }),
                leg({ id: 2, type: 'transfer', amount: 0.2, toAmount: 23, account: eur, toAccount: rsd }),
            ],
        }
        expect(paymentFormValues(group)?.handed_over).toBe(0.3)
    })

    it('gives up on shapes the form cannot show', () => {
        const refund = leg({ id: 2, type: 'income', amount: 5, account: eur })
        const elsewhere = leg({ id: 2, type: 'transfer', amount: 50, toAmount: 5900, account: card, toAccount: rsd })
        for (const legs of [
            [leg({ id: 1, type: 'expense', account: eur }), refund],
            [leg({ id: 1, type: 'expense', account: eur }), elsewhere],
            [leg({ id: 1, type: 'expense', account: eur })],
        ]) {
            expect(paymentFormValues({ uuid: 'g1', legs })).toBeNull()
        }
    })

    it('reads back what buildPaymentLegs made', () => {
        const values: TransactionFormValues = {
            ...base, payment_mode: 'change', handed_over: 500, change_account_id: 2, change_amount: 5900,
        }
        const made = buildPaymentLegs(values, accounts)
        if (!made.ok) throw new Error('not built')
        const byId = new Map(accounts.map((a) => [a.id, a]))
        const group: TransactionGroup = {
            uuid: 'g1',
            legs: made.legs.map((l, i) =>
                leg({
                    id: i + 1, type: l.type, amount: l.amount, toAmount: l.to_amount ?? null,
                    account: byId.get(l.account_id)!, toAccount: l.to_account_id ? byId.get(l.to_account_id)! : null,
                    category: l.category_id ? category : null,
                })),
        }
        expect(paymentFormValues(group)).toMatchObject({
            payment_mode: 'change', account_id: 1, amount: 450, handed_over: 500,
            change_account_id: 2, change_amount: 5900,
        })
    })
})
