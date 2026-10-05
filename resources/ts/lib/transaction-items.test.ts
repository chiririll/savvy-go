import { describe, expect, it } from 'vitest'

import {
    currencyDecimals,
    itemContribution,
    quantityDecimalPlaces,
    roundMoney,
    roundedItemPrice,
    sumTransactionItems,
} from './transaction-items'

describe('currencyDecimals', () => {
    it('defaults to two when the currency does not say', () => {
        expect(currencyDecimals()).toBe(2)
        expect(currencyDecimals(null)).toBe(2)
        expect(currencyDecimals({})).toBe(2)
        expect(currencyDecimals({ decimals: null })).toBe(2)
    })

    it('keeps zero, which is a real precision', () => {
        expect(currencyDecimals({ decimals: 0 })).toBe(0)
        expect(currencyDecimals({ decimals: 3 })).toBe(3)
    })
})

describe('roundMoney', () => {
    it('rounds half up despite binary floating point', () => {
        expect(roundMoney(1.005, 2)).toBe(1.01)
        expect(roundMoney(2.675, 2)).toBe(2.68)
        // These came out as 2.13 and 4.01 with the old Number.EPSILON nudge.
        expect(roundMoney(2.135, 2)).toBe(2.14)
        expect(roundMoney(4.015, 2)).toBe(4.02)
        expect(roundMoney(1.4, 0)).toBe(1)
        expect(roundMoney(1.5, 0)).toBe(2)
        expect(roundMoney(0.0005, 3)).toBe(0.001)
    })

    it('rounds every half cent up, like the server', () => {
        const wrong: string[] = []
        for (let cents = 0; cents < 100_000; cents++) {
            const half = `${(cents / 100).toFixed(2)}5`
            const expected = Number(((cents + 1) / 100).toFixed(2))
            if (roundMoney(Number(half), 2) !== expected) wrong.push(half)
        }
        expect(wrong.slice(0, 10)).toEqual([])
    })

    it('does not round up below the half', () => {
        expect(roundMoney(1.0049999, 2)).toBe(1)
        expect(roundMoney(2.134, 2)).toBe(2.13)
    })

    it('treats negative decimals as zero', () => {
        expect(roundMoney(12.7, -2)).toBe(13)
    })

    it('rounds negative values away from zero, as decimal.Round does', () => {
        expect(roundMoney(-1.239, 2)).toBe(-1.24)
        expect(roundMoney(-1.005, 2)).toBe(-1.01)
        expect(roundMoney(-2.5, 0)).toBe(-3)
    })

    it('keeps sums of floats exact', () => {
        expect(roundMoney(0.1 + 0.2, 2)).toBe(0.3)
        expect(roundMoney(123456789.125, 2)).toBe(123456789.13)
    })
})

describe('roundedItemPrice', () => {
    it('rounds the price and treats junk as zero', () => {
        expect(roundedItemPrice(1.239, 2)).toBe(1.24)
        expect(roundedItemPrice(NaN, 2)).toBe(0)
        expect(roundedItemPrice(Number('abc'), 2)).toBe(0)
    })
})

describe('itemContribution', () => {
    it('multiplies the rounded price by the quantity', () => {
        // 0.333 is billed as 0.33 per unit, not 0.333.
        expect(itemContribution(3, 0.333, 2)).toBeCloseTo(0.99, 10)
        expect(itemContribution(2.5, 10, 2)).toBe(25)
    })

    it('treats a bad quantity as zero', () => {
        expect(itemContribution(NaN, 10, 2)).toBe(0)
    })
})

describe('sumTransactionItems', () => {
    it('sums quantity times price and rounds once at the end', () => {
        expect(sumTransactionItems([{ quantity: 3, price_per_unit: 0.1 }], 2)).toBe(0.3)
        expect(sumTransactionItems([
            { quantity: 2, price_per_unit: 1.5 },
            { quantity: 1, price_per_unit: 0.25 },
        ], 2)).toBe(3.25)
    })

    it('accepts numeric strings from the API', () => {
        expect(sumTransactionItems([{ quantity: '2', price_per_unit: '4.50' }], 2)).toBe(9)
    })

    it('skips empty or missing values', () => {
        expect(sumTransactionItems(undefined, 2)).toBe(0)
        expect(sumTransactionItems([], 2)).toBe(0)
        expect(sumTransactionItems([{ quantity: null, price_per_unit: 5 }, { quantity: 2 }], 2)).toBe(0)
    })

    it('uses the currency precision', () => {
        expect(sumTransactionItems([{ quantity: 1.5, price_per_unit: 100 }], 0)).toBe(150)
        expect(sumTransactionItems([{ quantity: 1, price_per_unit: 10.4 }], 0)).toBe(10)
    })
})

describe('quantityDecimalPlaces', () => {
    it('counts decimal places', () => {
        expect(quantityDecimalPlaces(1)).toBe(0)
        expect(quantityDecimalPlaces(1.5)).toBe(1)
        expect(quantityDecimalPlaces(0.001)).toBe(3)
        expect(quantityDecimalPlaces(-2.25)).toBe(2)
    })

    it('understands exponent notation', () => {
        expect(quantityDecimalPlaces(1e-7)).toBe(7)
        expect(quantityDecimalPlaces(1.5e-7)).toBe(8)
        expect(quantityDecimalPlaces(1e21)).toBe(0)
    })

    it('reports non-finite numbers as infinitely precise', () => {
        expect(quantityDecimalPlaces(NaN)).toBe(Infinity)
        expect(quantityDecimalPlaces(Infinity)).toBe(Infinity)
    })
})
