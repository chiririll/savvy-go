import { describe, expect, it } from 'vitest'

import { accounts, currencies } from '@/test/fixtures'
import { buildTransactionInput, type PayloadValues } from './transaction-payload'
import { changeTooLarge, nextFreeAccount, payloadErrorPath, paymentTotals } from './transaction-payments'

const [cash, euroCard, yen] = accounts
const [, eur, jpy] = currencies

describe('paymentTotals', () => {
    it('is empty without payments', () => {
        expect(paymentTotals({ payments: [] }, accounts)).toEqual([])
    })

    it('gives one total per currency, in the order the currencies first appear', () => {
        const totals = paymentTotals(
            {
                payments: [
                    { account_id: euroCard.id, amount: 10 },
                    { account_id: cash.id, amount: 5 },
                    { account_id: yen.id, amount: 300 },
                ],
            },
            accounts,
        )

        expect(totals.map((t) => [t.currency.code, t.net])).toEqual([
            ['EUR', 10],
            ['USD', 5],
            ['JPY', 300],
        ])
    })

    it('adds the payments of one currency into one total', () => {
        const other = { ...cash, id: 9, name: 'Wallet' }
        const totals = paymentTotals(
            {
                payments: [
                    { account_id: cash.id, amount: 20.1 },
                    { account_id: other.id, amount: 0.2 },
                ],
            },
            [...accounts, other],
        )

        expect(totals).toHaveLength(1)
        expect(totals[0]).toMatchObject({ paid: 20.3, back: 0, net: 20.3 })
    })

    it('takes the change off the currency it came back in, and may go below zero', () => {
        const totals = paymentTotals(
            {
                payments: [{ account_id: cash.id, amount: 100 }],
                change: [
                    { account_id: euroCard.id, amount: 20 },
                    { account_id: yen.id, amount: 0 },
                ],
            },
            accounts,
        )

        expect(totals.map((t) => [t.currency.code, t.paid, t.back, t.net])).toEqual([
            ['USD', 100, 0, 100],
            ['EUR', 0, 20, -20],
        ])
    })

    it('leaves out rows that have no amount yet or an account that is not chosen', () => {
        const totals = paymentTotals(
            {
                payments: [
                    { account_id: cash.id, amount: '' as unknown as number },
                    { account_id: 0, amount: 5 },
                    { account_id: euroCard.id, amount: 5 },
                ],
            },
            accounts,
        )

        expect(totals.map((t) => t.currency.code)).toEqual(['EUR'])
    })
})

describe('nextFreeAccount', () => {
    it('is the first account that no row uses', () => {
        expect(nextFreeAccount(accounts, [cash.id])?.id).toBe(euroCard.id)
        expect(nextFreeAccount(accounts, [cash.id, yen.id])?.id).toBe(euroCard.id)
    })

    it('is undefined when every account is used, so the (+) is disabled', () => {
        expect(nextFreeAccount(accounts, accounts.map((a) => a.id))).toBeUndefined()
        expect(nextFreeAccount(undefined, [])).toBeUndefined()
    })

    it('does not mix up ids that arrive as numeric strings', () => {
        expect(nextFreeAccount(accounts, ['1' as unknown as number])?.id).toBe(euroCard.id)
    })
})

describe('changeTooLarge', () => {
    it('is false without change, or when the payments are worth more', () => {
        expect(changeTooLarge({ payments: [{ account_id: cash.id, amount: 10 }] }, accounts)).toBe(false)
        expect(
            changeTooLarge(
                { payments: [{ account_id: cash.id, amount: 10 }], change: [{ account_id: euroCard.id, amount: 7 }] },
                accounts,
            ),
        ).toBe(false)
    })

    it('is true when the change is worth the payments or more, in the base currency', () => {
        const paid = { payments: [{ account_id: cash.id, amount: 10 }] }
        // 8 EUR is 10 in the base currency at 1.25, so the net is zero
        expect(changeTooLarge({ ...paid, change: [{ account_id: euroCard.id, amount: 8 }] }, accounts)).toBe(true)
        expect(changeTooLarge({ ...paid, change: [{ account_id: euroCard.id, amount: 9 }] }, accounts)).toBe(true)
    })

    it('ignores rows with no amount yet', () => {
        expect(
            changeTooLarge(
                { payments: [{ account_id: cash.id, amount: 10 }], change: [{ account_id: euroCard.id, amount: '' as unknown as number }] },
                accounts,
            ),
        ).toBe(false)
    })

    it('is false while there is nothing paid yet', () => {
        const unpaid = { payments: [{ account_id: cash.id, amount: '' as unknown as number }] }
        expect(changeTooLarge({ ...unpaid, change: [{ account_id: euroCard.id, amount: 5 }] }, accounts)).toBe(false)
    })
})

describe('payloadErrorPath', () => {
    const values = (overrides: Partial<PayloadValues> = {}): PayloadValues => ({
        type: 'expense',
        payments: [{ account_id: cash.id, amount: 10 }],
        default_category_id: 1,
        ...overrides,
    })
    const pathOf = (v: PayloadValues) => {
        const built = buildTransactionInput(v, accounts)
        if (built.ok) {
            throw new Error('expected the payload to be refused')
        }
        return payloadErrorPath(built.error, v, accounts)
    }

    it('points at the second use of an account', () => {
        expect(
            pathOf(values({ payments: [{ account_id: cash.id, amount: 5 }, { account_id: cash.id, amount: 5 }] })),
        ).toBe('payments.1.account_id')
        expect(
            pathOf(values({ payments: [{ account_id: cash.id, amount: 5 }], change: [{ account_id: cash.id, amount: 1 }] })),
        ).toBe('change.0.account_id')
    })

    it('points at the first zero amount, in the payments or in the change', () => {
        expect(pathOf(values({ payments: [{ account_id: cash.id, amount: 5 }, { account_id: euroCard.id, amount: 0 }] }))).toBe(
            'payments.1.amount',
        )
        expect(pathOf(values({ change: [{ account_id: euroCard.id, amount: 0 }] }))).toBe('change.0.amount')
    })

    it('points at an account that is not loaded', () => {
        expect(pathOf(values({ payments: [{ account_id: cash.id, amount: 5 }, { account_id: 99, amount: 5 }] }))).toBe(
            'payments.1.account_id',
        )
    })

    it('points at the amount of the change when it is too large', () => {
        expect(pathOf(values({ change: [{ account_id: euroCard.id, amount: 20 }] }))).toBe('change.0.amount')
    })

    it('points at the line whose currency no account has', () => {
        expect(
            pathOf(values({ lines: [{ category_id: 1, amount: 3 }, { category_id: 3, currency_id: jpy.id, amount: 5 }] })),
        ).toBe('lines.1.currency_id')
    })

    it('accepts a line in the currency of a change account', () => {
        const built = buildTransactionInput(
            values({
                change: [{ account_id: euroCard.id, amount: 1 }],
                lines: [{ category_id: 1, currency_id: eur.id, amount: 3 }],
            }),
            accounts,
        )

        expect(built.ok).toBe(true)
    })
})
