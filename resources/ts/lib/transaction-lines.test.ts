import { describe, expect, it } from 'vitest'

import { accounts, currencies } from '@/test/fixtures'
import { excessAmount, lineCoverage, linesNeedNotes, linesToAdd, paymentCurrencies } from './transaction-lines'

const [usd, eur] = currencies
const [cash, euroCard, yenWallet] = accounts

describe('paymentCurrencies', () => {
    it('lists each currency of the payment and change accounts once, in order', () => {
        const values = {
            payments: [{ account_id: cash.id, amount: 10 }],
            change: [{ account_id: euroCard.id, amount: 1 }],
        }

        expect(paymentCurrencies(values, accounts).map((c) => c.code)).toEqual(['USD', 'EUR'])
    })

    it('does not repeat a currency two accounts share', () => {
        const other = { ...euroCard, id: 9, currency: usd }
        const values = {
            payments: [{ account_id: cash.id, amount: 10 }, { account_id: other.id, amount: 5 }],
        }

        expect(paymentCurrencies(values, [...accounts, other])).toHaveLength(1)
    })

    it('skips a row that has no account yet', () => {
        expect(paymentCurrencies({ payments: [{ account_id: 0, amount: 1 }] }, accounts)).toEqual([])
    })
})

describe('excessAmount', () => {
    it('is what the lines have above the payments, never negative', () => {
        expect(excessAmount([{ currency: usd, amount: 10 }], [{ currency: usd, amount: 12.5 }], usd)).toBe(2.5)
        expect(excessAmount([{ currency: usd, amount: 10 }], [{ currency: usd, amount: 4 }], usd)).toBe(0)
    })

    it('compares in the base currency and states the excess in the target', () => {
        // 10 USD paid, lines worth 15 USD (12 EUR): 5 USD is 4 EUR
        expect(excessAmount([{ currency: usd, amount: 10 }], [{ currency: eur, amount: 12 }], eur)).toBe(4)
    })

    it('counts change as less paid', () => {
        // 10 USD paid, 2.50 USD back (2 EUR) leaves 7.50, lines of 10 are 2.50 over
        expect(
            excessAmount([{ currency: usd, amount: 10 }], [{ currency: usd, amount: 10 }], usd, [
                { currency: eur, amount: 2 },
            ]),
        ).toBe(2.5)
    })
})

describe('lineCoverage', () => {
    const paid = { payments: [{ account_id: cash.id, amount: 52.4 }] }

    it('is the whole payment to cover while there are no lines', () => {
        expect(lineCoverage(paid, accounts)).toMatchObject({ total: 52.4, remainder: 52.4, excess: 0 })
        expect(lineCoverage(paid, accounts).currency?.code).toBe('USD')
    })

    it('leaves the rest of the payment to the default category', () => {
        const values = { ...paid, lines: [{ amount: 40 }, { amount: 2.4 }] }

        expect(lineCoverage(values, accounts)).toMatchObject({ total: 52.4, remainder: 10, excess: 0 })
    })

    it('has nothing left when the lines cover the payment', () => {
        const values = { ...paid, lines: [{ amount: 40 }, { amount: 12.4 }] }

        expect(lineCoverage(values, accounts)).toMatchObject({ remainder: 0, excess: 0 })
    })

    it('reports an excess when the lines are more than the payment', () => {
        const values = { ...paid, lines: [{ amount: 50 }, { amount: 12.4 }] }

        expect(lineCoverage(values, accounts)).toMatchObject({ remainder: 0, excess: 10 })
    })

    it('reads a line in another currency of the accounts at the rates', () => {
        // 25 USD + 20 EUR (25 USD) paid; a line of 20 EUR is 25 of the 50 USD
        const values = {
            payments: [{ account_id: cash.id, amount: 25 }, { account_id: euroCard.id, amount: 20 }],
            lines: [{ amount: 20, currency_id: eur.id }],
        }

        expect(lineCoverage(values, accounts)).toMatchObject({ total: 50, remainder: 25 })
    })

    it('ignores a line with no amount yet', () => {
        const values = { ...paid, lines: [{ amount: '' as unknown as number }] }

        expect(lineCoverage(values, accounts).remainder).toBe(52.4)
    })

    it('has no currency before an account is chosen', () => {
        expect(lineCoverage({ payments: [{ account_id: 0, amount: 5 }] }, accounts)).toMatchObject({
            currency: null,
            total: 0,
            remainder: 0,
        })
    })

    it('takes the change off what there is to cover', () => {
        const values = {
            payments: [{ account_id: yenWallet.id, amount: 1000 }],
            change: [{ account_id: cash.id, amount: 2.5 }],
        }

        // 1000 JPY is 5 USD; 2.50 USD back leaves 2.50 USD = 500 JPY
        expect(lineCoverage(values, accounts)).toMatchObject({ total: 500, remainder: 500 })
    })
})

describe('linesNeedNotes', () => {
    it('is false for no line or one line, true from two lines on', () => {
        expect(linesNeedNotes(0)).toBe(false)
        expect(linesNeedNotes(1)).toBe(false)
        expect(linesNeedNotes(2)).toBe(true)
    })
})

describe('linesToAdd', () => {
    const paid = { payments: [{ account_id: cash.id, amount: 52.4 }] }

    it('starts with two lines: the whole payment with the default category, and an empty one', () => {
        expect(linesToAdd({ ...paid, default_category_id: 3 }, accounts)).toEqual([
            { category_id: 3, currency_id: null, amount: 52.4, note: '' },
            { category_id: null, currency_id: null, amount: '', note: '' },
        ])
    })

    it('adds one empty line to lines that are already there', () => {
        expect(linesToAdd({ ...paid, lines: [{ amount: 40 }] }, accounts)).toEqual([
            { category_id: null, currency_id: null, amount: '', note: '' },
        ])
    })

    it('leaves the amount of the first line empty while no payment is filled in', () => {
        const [first] = linesToAdd({ payments: [{ account_id: cash.id, amount: '' as unknown as number }] }, accounts)

        expect(first.amount).toBe('')
    })
})
