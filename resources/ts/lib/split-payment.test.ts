import { describe, expect, it } from 'vitest'
import { payWithChange, rateDeviation, splitAcrossAccounts, type PaymentAccount } from './split-payment'

const eur: PaymentAccount = { id: 1, currency: 'EUR', decimals: 2 }
const rsd: PaymentAccount = { id: 2, currency: 'RSD', decimals: 2 }
const card: PaymentAccount = { id: 3, currency: 'EUR', decimals: 2 }
const jpy: PaymentAccount = { id: 4, currency: 'JPY', decimals: 0 }

const common = { date: '2026-10-17', description: 'Rent', categoryId: 7, tagIds: [5] }

describe('payWithChange: the price is in the currency paid with', () => {
    it('turns 500 EUR for 450 EUR rent with 5900 RSD back into expense and exchange', () => {
        const result = payWithChange(common, {
            priceOn: 'paid', price: 450, paidFrom: eur, handedOver: 500, changeTo: rsd, change: 5900,
        })
        expect(result).toEqual({
            ok: true,
            rate: 118,
            legs: [
                {
                    type: 'expense', account_id: 1, category_id: 7, amount: 450,
                    description: 'Rent', date: '2026-10-17', tag_ids: [5],
                },
                {
                    type: 'transfer', account_id: 1, to_account_id: 2, amount: 50, to_amount: 5900,
                    description: 'Rent', date: '2026-10-17', tag_ids: [5],
                },
            ],
        })
    })

    it('keeps the surplus exact where float arithmetic would not', () => {
        const result = payWithChange(common, {
            priceOn: 'paid', price: 0.1, paidFrom: eur, handedOver: 0.3, changeTo: rsd, change: 23,
        })
        expect(result.ok && result.legs[1].amount).toBe(0.2)
    })

    it('rounds the surplus to the decimals of the currency paid with', () => {
        const result = payWithChange(common, {
            priceOn: 'paid', price: 1200, paidFrom: jpy, handedOver: 2000, changeTo: eur, change: 4.8,
        })
        expect(result.ok && result.legs[1]).toMatchObject({ amount: 800, to_amount: 4.8 })
    })

    it('needs more handed over than the price', () => {
        for (const handedOver of [450, 400]) {
            expect(payWithChange(common, {
                priceOn: 'paid', price: 450, paidFrom: eur, handedOver, changeTo: rsd, change: 5900,
            })).toEqual({ ok: false, error: 'handedOverTooLow' })
        }
    })
})

describe('payWithChange: the price is in the currency of the change', () => {
    it('exchanges what was handed over, then pays the price from the account that got the change', () => {
        // 53000 RSD due, 450 EUR handed over, 200 RSD back: 53200 RSD came out of the exchange.
        const result = payWithChange(common, {
            priceOn: 'change', price: 53000, paidFrom: eur, handedOver: 450, changeTo: rsd, change: 200,
        })
        expect(result).toEqual({
            ok: true,
            rate: 118.222222,
            legs: [
                {
                    type: 'expense', account_id: 2, category_id: 7, amount: 53000,
                    description: 'Rent', date: '2026-10-17', tag_ids: [5],
                },
                {
                    type: 'transfer', account_id: 1, to_account_id: 2, amount: 450, to_amount: 53200,
                    description: 'Rent', date: '2026-10-17', tag_ids: [5],
                },
            ],
        })
    })
})

describe('payWithChange: input that is not a payment with change', () => {
    const base = { priceOn: 'paid', price: 450, paidFrom: eur, handedOver: 500, changeTo: rsd, change: 5900 } as const

    it('refuses change in the currency paid with: enter what was paid instead', () => {
        expect(payWithChange(common, { ...base, changeTo: card, change: 50 }))
            .toEqual({ ok: false, error: 'sameCurrency' })
    })

    it('refuses amounts that are not positive', () => {
        for (const patch of [{ price: 0 }, { handedOver: 0 }, { change: 0 }, { change: -1 }, { price: Number.NaN }]) {
            expect(payWithChange(common, { ...base, ...patch })).toEqual({ ok: false, error: 'amountPositive' })
        }
    })

    it('leaves description and tags out when there are none', () => {
        const result = payWithChange({ date: '2026-10-17', categoryId: 7 }, base)
        expect(result.ok && result.legs[0]).toEqual({
            type: 'expense', account_id: 1, category_id: 7, amount: 450, date: '2026-10-17',
        })
    })
})

describe('splitAcrossAccounts', () => {
    it('records one expense per account', () => {
        const result = splitAcrossAccounts(common, [{ account: eur, amount: 15.5 }, { account: card, amount: 24.5 }])
        expect(result).toEqual({
            ok: true,
            legs: [
                {
                    type: 'expense', account_id: 1, category_id: 7, amount: 15.5,
                    description: 'Rent', date: '2026-10-17', tag_ids: [5],
                },
                {
                    type: 'expense', account_id: 3, category_id: 7, amount: 24.5,
                    description: 'Rent', date: '2026-10-17', tag_ids: [5],
                },
            ],
        })
    })

    it('works across currencies', () => {
        const result = splitAcrossAccounts(common, [{ account: eur, amount: 20 }, { account: rsd, amount: 1000 }])
        expect(result.ok && result.legs.map((leg) => leg.account_id)).toEqual([1, 2])
    })

    it('needs two accounts', () => {
        expect(splitAcrossAccounts(common, [{ account: eur, amount: 5 }]))
            .toEqual({ ok: false, error: 'needTwoParts' })
        expect(splitAcrossAccounts(common, [{ account: eur, amount: 5 }, { account: eur, amount: 5 }]))
            .toEqual({ ok: false, error: 'needTwoParts' })
    })

    it('refuses an amount that is not positive', () => {
        expect(splitAcrossAccounts(common, [{ account: eur, amount: 5 }, { account: rsd, amount: 0 }]))
            .toEqual({ ok: false, error: 'amountPositive' })
    })
})

describe('rateDeviation', () => {
    it('tells how far the rate paid is from the expected one', () => {
        expect(rateDeviation(118, 117.5)).toBeCloseTo(0.0043, 4)
        expect(rateDeviation(100, 125)).toBeCloseTo(0.2, 6)
    })

    it('is null when there is nothing to compare with', () => {
        expect(rateDeviation(118, null)).toBeNull()
        expect(rateDeviation(118, 0)).toBeNull()
    })
})
