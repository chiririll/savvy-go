import { describe, expect, it } from 'vitest'
import { paymentLegs, type PaymentAccount } from './split-payment'

const eur: PaymentAccount = { id: 1, currency: 'EUR', decimals: 2 }
const rsd: PaymentAccount = { id: 2, currency: 'RSD', decimals: 2 }
const card: PaymentAccount = { id: 3, currency: 'EUR', decimals: 2 }
const jpy: PaymentAccount = { id: 4, currency: 'JPY', decimals: 0 }

const common = { date: '2026-10-17', description: 'Rent', categoryId: 7, tagIds: [5] }
const shared = { description: 'Rent', date: '2026-10-17', tag_ids: [5] }

describe('paymentLegs: several accounts paid', () => {
    it('records one expense per account', () => {
        const result = paymentLegs(common, [{ account: eur, amount: 15.5 }, { account: card, amount: 24.5 }])
        expect(result).toEqual({
            ok: true,
            legs: [
                { type: 'expense', account_id: 1, category_id: 7, amount: 15.5, ...shared },
                { type: 'expense', account_id: 3, category_id: 7, amount: 24.5, ...shared },
            ],
        })
    })

    it('works across currencies', () => {
        const result = paymentLegs(common, [{ account: eur, amount: 20 }, { account: rsd, amount: 1000 }])
        expect(result.ok && result.legs.map((l) => l.account_id)).toEqual([1, 2])
    })

    it('needs two accounts when there is no change', () => {
        expect(paymentLegs(common, [{ account: eur, amount: 5 }])).toEqual({ ok: false, error: 'needTwoParts' })
        expect(paymentLegs(common, [{ account: eur, amount: 5 }, { account: eur, amount: 5 }]))
            .toEqual({ ok: false, error: 'needTwoParts' })
    })

    it('rounds an amount to the decimals of its currency', () => {
        const result = paymentLegs(common, [{ account: jpy, amount: 1200.4 }, { account: eur, amount: 1 }])
        expect(result.ok && result.legs[0].amount).toBe(1200)
    })
})

describe('paymentLegs: change', () => {
    it('is an expense of what was handed over and an income of the change', () => {
        // 500 EUR handed over for rent, 5900 RSD back.
        const result = paymentLegs(common, [{ account: eur, amount: 500 }], [{ account: rsd, amount: 5900 }])
        expect(result).toEqual({
            ok: true,
            legs: [
                { type: 'expense', account_id: 1, category_id: 7, amount: 500, ...shared },
                { type: 'income', account_id: 2, category_id: 7, amount: 5900, ...shared },
            ],
        })
    })

    it('takes several change rows, after the expenses', () => {
        const result = paymentLegs(
            common,
            [{ account: eur, amount: 500 }, { account: rsd, amount: 100 }],
            [{ account: rsd, amount: 3000 }, { account: card, amount: 2 }],
        )
        expect(result.ok && result.legs.map((l) => [l.type, l.account_id, l.amount])).toEqual([
            ['expense', 1, 500],
            ['expense', 2, 100],
            ['income', 2, 3000],
            ['income', 3, 2],
        ])
    })
})

describe('paymentLegs: input that is not a payment', () => {
    it('refuses amounts that are not positive', () => {
        const paid = [{ account: eur, amount: 5 }]
        for (const amount of [0, -1, Number.NaN]) {
            expect(paymentLegs(common, [...paid, { account: rsd, amount }])).toEqual({ ok: false, error: 'amountPositive' })
            expect(paymentLegs(common, paid, [{ account: rsd, amount }])).toEqual({ ok: false, error: 'amountPositive' })
        }
    })

    it('leaves description and tags out when there are none', () => {
        const result = paymentLegs({ date: '2026-10-17', categoryId: 7 }, [{ account: eur, amount: 5 }], [{ account: rsd, amount: 9 }])
        expect(result.ok && result.legs[0]).toEqual({
            type: 'expense', account_id: 1, category_id: 7, amount: 5, date: '2026-10-17',
        })
    })
})
