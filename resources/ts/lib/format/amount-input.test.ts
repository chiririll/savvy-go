import { describe, expect, it } from 'vitest'

import { sanitizeAmount } from './amount-input'

describe('sanitizeAmount', () => {
    it.each([
        ['', 2, ''],
        ['12', 2, '12'],
        ['12.5', 2, '12.5'],
        ['12.50', 2, '12.50'],
        ['.5', 2, '.5'],
        ['12.', 2, '12.'],
    ])('keeps %j (%i decimals) as %j', (raw, decimals, expected) => {
        expect(sanitizeAmount(raw, decimals)).toBe(expected)
    })

    it('takes a comma as the decimal separator', () => {
        expect(sanitizeAmount('12,5', 2)).toBe('12.5')
    })

    it('drops letters, signs and exponents', () => {
        expect(sanitizeAmount('1e5', 2)).toBe('15')
        expect(sanitizeAmount('-12', 2)).toBe('12')
        expect(sanitizeAmount('+12abc', 2)).toBe('12')
        expect(sanitizeAmount('1 234', 2)).toBe('1234')
    })

    it('keeps the first separator only', () => {
        expect(sanitizeAmount('1.2.', 2)).toBe('1.2')
        expect(sanitizeAmount('1.2.3', 2)).toBe('1.23')
        expect(sanitizeAmount('1,2.3', 3)).toBe('1.23')
    })

    it('cuts the fraction to the decimals of the currency', () => {
        expect(sanitizeAmount('1.239', 2)).toBe('1.23')
        expect(sanitizeAmount('1.2345', 3)).toBe('1.234')
    })

    it('takes no separator for a currency without decimals', () => {
        expect(sanitizeAmount('1234.5', 0)).toBe('1234')
        expect(sanitizeAmount('12,', 0)).toBe('12')
    })
})
