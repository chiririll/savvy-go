import { describe, expect, it } from 'vitest'

import { accounts, currencies } from '@/test/fixtures'
import {
    buildTransactionInput,
    findDuplicateAccount,
    netSignHolds,
    remainderAmount,
    type PayloadValues,
} from './transaction-payload'

const [usd, eur, jpy] = currencies
const [cash, euroCard, yenWallet] = accounts

const values = (overrides: Partial<PayloadValues> = {}): PayloadValues => ({
    type: 'expense',
    description: 'Groceries',
    date: '2026-11-07',
    tag_ids: [],
    payments: [{ account_id: cash.id, amount: 52.4 }],
    default_category_id: 1,
    ...overrides,
})

describe('findDuplicateAccount', () => {
    it('is null when every account appears once', () => {
        expect(findDuplicateAccount([{ account_id: 1 }, { account_id: 2 }], [{ account_id: 3 }])).toBeNull()
    })

    it('finds an account repeated among the payments', () => {
        expect(findDuplicateAccount([{ account_id: 1 }, { account_id: 1 }])).toBe(1)
    })

    it('finds an account that is both paid from and given the change', () => {
        expect(findDuplicateAccount([{ account_id: 1 }, { account_id: 2 }], [{ account_id: 2 }])).toBe(2)
    })

    it('does not mix up ids that arrive as numbers or numeric strings', () => {
        expect(findDuplicateAccount([{ account_id: 5 }], [{ account_id: '5' as unknown as number }])).toBe(5)
    })
})

describe('netSignHolds', () => {
    it('holds for a payment with no change', () => {
        expect(netSignHolds([{ currency: usd, amount: 10 }], [])).toBe(true)
        expect(netSignHolds([{ currency: usd, amount: 10 }], [])).toBe(true)
    })

    it('holds when the change is less than what was paid, in the base currency', () => {
        // 100 USD paid, 10 EUR (12.50 USD) back
        expect(netSignHolds([{ currency: usd, amount: 100 }], [{ currency: eur, amount: 10 }])).toBe(true)
    })

    it('fails when the change is worth more than what was paid', () => {
        // 10 USD paid, 10 EUR (12.50 USD) back
        expect(netSignHolds([{ currency: usd, amount: 10 }], [{ currency: eur, amount: 10 }])).toBe(false)
    })

    it('fails when the change is worth exactly what was paid: the net is zero', () => {
        expect(netSignHolds([{ currency: usd, amount: 12.5 }], [{ currency: eur, amount: 10 }])).toBe(false)
    })
})

describe('remainderAmount', () => {
    it('is the whole payment when there are no lines', () => {
        expect(remainderAmount([{ currency: usd, amount: 52.4 }], [], usd)).toBe(52.4)
    })

    it('is zero when the lines cover the payment', () => {
        const lines = [
            { currency: usd, amount: 40 },
            { currency: usd, amount: 12.4 },
        ]
        expect(remainderAmount([{ currency: usd, amount: 52.4 }], lines, usd)).toBe(0)
    })

    it('is what the lines leave', () => {
        expect(remainderAmount([{ currency: usd, amount: 52.4 }], [{ currency: usd, amount: 40 }], usd)).toBe(12.4)
    })

    it('is zero, never negative, when the lines exceed the payment', () => {
        expect(remainderAmount([{ currency: usd, amount: 10 }], [{ currency: usd, amount: 12 }], usd)).toBe(0)
    })

    it('compares in the base currency: EUR lines can cover a USD payment', () => {
        // 25 USD is 20 EUR
        expect(remainderAmount([{ currency: usd, amount: 25 }], [{ currency: eur, amount: 20 }], usd)).toBe(0)
    })

    it('gives the rest in the currency asked for, rounded to its decimals', () => {
        // paid 25 USD, lines 10 EUR (12.50 USD): 12.50 USD left, which is 10 EUR
        expect(remainderAmount([{ currency: usd, amount: 25 }], [{ currency: eur, amount: 10 }], eur)).toBe(10)
        // paid 1000 JPY (5 USD), lines 2 USD: 3 USD left is 600 JPY
        expect(remainderAmount([{ currency: jpy, amount: 1000 }], [{ currency: usd, amount: 2 }], jpy)).toBe(600)
    })

    it('adds payments of several currencies and takes the change off', () => {
        // paid 10 USD + 8 EUR (10 USD) = 20 USD, change 2 USD back, lines 8 USD: 10 USD left
        const paid = [
            { currency: usd, amount: 10 },
            { currency: eur, amount: 8 },
        ]
        expect(remainderAmount(paid, [{ currency: usd, amount: 8 }], usd, [{ currency: usd, amount: 2 }])).toBe(10)
    })

    it('does not leave a cent of float noise', () => {
        expect(remainderAmount([{ currency: usd, amount: 0.3 }], [{ currency: usd, amount: 0.1 }], usd)).toBe(0.2)
    })
})

describe('buildTransactionInput', () => {
    it('saves an expense as a negative payment and one line for the whole amount', () => {
        const built = buildTransactionInput(values(), accounts)

        expect(built).toEqual({
            ok: true,
            input: {
                type: 'expense',
                description: 'Groceries',
                date: '2026-11-07',
                payments: [{ account_id: cash.id, amount: -52.4 }],
                lines: [{ category_id: 1, currency_id: usd.id, amount: 52.4 }],
                tag_ids: [],
            },
        })
    })

    it('saves an income as a positive payment', () => {
        const built = buildTransactionInput(values({ type: 'income', default_category_id: 2 }), accounts)

        expect(built).toMatchObject({
            ok: true,
            input: { type: 'income', payments: [{ account_id: cash.id, amount: 52.4 }], lines: [{ category_id: 2 }] },
        })
    })

    it('sends an empty date as null, which makes the transaction pending', () => {
        const built = buildTransactionInput(values({ date: '' }), accounts)

        expect(built).toMatchObject({ ok: true, input: { date: null } })
    })

    it('sends an empty description as null', () => {
        const built = buildTransactionInput(values({ description: '' }), accounts)

        expect(built).toMatchObject({ ok: true, input: { description: null } })
    })

    it('rounds amounts to the decimals of the account currency', () => {
        const yen = buildTransactionInput(values({ payments: [{ account_id: yenWallet.id, amount: 1000.4 }] }), accounts)
        expect(yen).toMatchObject({ ok: true, input: { payments: [{ amount: -1000 }], lines: [{ amount: 1000 }] } })

        const noise = buildTransactionInput(values({ payments: [{ account_id: cash.id, amount: 0.1 + 0.2 }] }), accounts)
        expect(noise).toMatchObject({ ok: true, input: { payments: [{ amount: -0.3 }], lines: [{ amount: 0.3 }] } })
    })

    it('keeps the note of the line', () => {
        const built = buildTransactionInput(values({ default_note: '- milk\n- bread' }), accounts)

        expect(built).toMatchObject({ ok: true, input: { lines: [{ note: '- milk\n- bread' }] } })
    })

    it('puts the line in the currency of the account', () => {
        const built = buildTransactionInput(values({ payments: [{ account_id: euroCard.id, amount: 8 }] }), accounts)

        expect(built).toMatchObject({ ok: true, input: { lines: [{ currency_id: eur.id, amount: 8 }] } })
    })

    it('saves the change with the sign opposite to the payments', () => {
        const built = buildTransactionInput(
            values({
                payments: [{ account_id: cash.id, amount: 100 }],
                change: [{ account_id: euroCard.id, amount: 10 }],
            }),
            accounts,
        )

        expect(built).toMatchObject({
            ok: true,
            input: {
                payments: [
                    { account_id: cash.id, amount: -100 },
                    { account_id: euroCard.id, amount: 10 },
                ],
                // 100 USD paid, 12.50 USD back: 87.50 USD was spent
                lines: [{ category_id: 1, currency_id: usd.id, amount: 87.5 }],
            },
        })
    })

    it('adds the rest to the default category when the lines cover less', () => {
        const built = buildTransactionInput(
            values({
                lines: [{ category_id: 3, currency_id: usd.id, amount: 40 }],
                default_category_id: 1,
            }),
            accounts,
        )

        expect(built).toMatchObject({
            ok: true,
            input: {
                lines: [
                    { category_id: 3, currency_id: usd.id, amount: 40 },
                    { category_id: 1, currency_id: usd.id, amount: 12.4 },
                ],
            },
        })
    })

    it('adds no line when the lines cover the payment', () => {
        const built = buildTransactionInput(
            values({
                lines: [
                    { category_id: 3, currency_id: usd.id, amount: 40 },
                    { category_id: 4, currency_id: usd.id, amount: 12.4 },
                ],
            }),
            accounts,
        )

        expect(built).toMatchObject({ ok: true })
        expect(built.ok && built.input.lines).toHaveLength(2)
    })

    it('adds no line when the lines exceed the payment', () => {
        const built = buildTransactionInput(
            values({ lines: [{ category_id: 3, currency_id: usd.id, amount: 60 }] }),
            accounts,
        )

        expect(built.ok && built.input.lines).toEqual([{ category_id: 3, currency_id: usd.id, amount: 60 }])
    })

    it('saves the rest without a category when there is no default one', () => {
        const built = buildTransactionInput(values({ default_category_id: null }), accounts)

        expect(built).toMatchObject({ ok: true, input: { lines: [{ category_id: null, amount: 52.4 }] } })
    })

    it('puts the rest in the currency of the first payment', () => {
        const built = buildTransactionInput(
            values({
                payments: [
                    { account_id: euroCard.id, amount: 12 },
                    { account_id: cash.id, amount: 10 },
                ],
                lines: [{ category_id: 3, currency_id: usd.id, amount: 15 }],
            }),
            accounts,
        )

        // 12 EUR + 10 USD is 25 USD; 15 USD in lines leaves 10 USD, which is 8 EUR
        expect(built.ok && built.input.lines?.[1]).toEqual({ category_id: 1, currency_id: eur.id, amount: 8 })
    })

    it('refuses a line in a currency none of the accounts has', () => {
        const built = buildTransactionInput(
            values({ lines: [{ category_id: 3, currency_id: jpy.id, amount: 5 }] }),
            accounts,
        )

        expect(built).toEqual({ ok: false, error: 'lineCurrency' })
    })

    it('refuses an account that is not loaded', () => {
        expect(buildTransactionInput(values({ payments: [{ account_id: 99, amount: 5 }] }), accounts)).toEqual({
            ok: false,
            error: 'unknownAccount',
        })
    })

    it('refuses the same account twice', () => {
        const built = buildTransactionInput(
            values({
                payments: [
                    { account_id: cash.id, amount: 5 },
                    { account_id: cash.id, amount: 5 },
                ],
            }),
            accounts,
        )

        expect(built).toEqual({ ok: false, error: 'duplicateAccount' })
    })

    it('refuses a zero amount', () => {
        expect(buildTransactionInput(values({ payments: [{ account_id: cash.id, amount: 0 }] }), accounts)).toEqual({
            ok: false,
            error: 'amountZero',
        })
    })

    it('refuses change worth more than what was paid', () => {
        const built = buildTransactionInput(
            values({
                payments: [{ account_id: cash.id, amount: 10 }],
                change: [{ account_id: euroCard.id, amount: 10 }],
            }),
            accounts,
        )

        expect(built).toEqual({ ok: false, error: 'netSign' })
    })
})
