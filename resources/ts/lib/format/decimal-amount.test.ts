import { describe, expect, it } from 'vitest'

import { fromUnits, isAmount, numberToAmount, subtractAmounts, toUnits } from './decimal-amount'

describe('decimal amounts', () => {
    it('subtracts without float error', () => {
        expect(0.3 - 0.1).not.toBe(0.2)
        expect(subtractAmounts('0.30', '0.10', 2)).toBe('0.20')
        expect(subtractAmounts('0.3', '0.1', 2)).toBe('0.20')
        expect(subtractAmounts('1.1', '1', 1)).toBe('0.1')
        expect(subtractAmounts('0.1', '0.3', 1)).toBe('-0.2')
    })

    it('keeps the sign and the zero before the point', () => {
        expect(fromUnits(-5n, 2)).toBe('-0.05')
        expect(fromUnits(5n, 2)).toBe('0.05')
        expect(fromUnits(0n, 2)).toBe('0.00')
        expect(fromUnits(1234n, 0)).toBe('1234')
    })

    it('handles amounts beyond the safe float range', () => {
        expect(subtractAmounts('90071992547409.93', '0.01', 2)).toBe('90071992547409.92')
    })

    it('parses a comma and rejects what is not an amount', () => {
        expect(toUnits('1,5', 2)).toBe(150n)
        expect(toUnits('.5', 2)).toBe(50n)
        expect(toUnits('5.', 2)).toBe(500n)
        expect(toUnits('', 2)).toBeNull()
        expect(toUnits('.', 2)).toBeNull()
        expect(toUnits('-', 2)).toBeNull()
        expect(toUnits('1.234', 2)).toBeNull()
        expect(toUnits('abc', 2)).toBeNull()
    })

    it('works for currencies without decimals', () => {
        expect(subtractAmounts('1000', '999', 0)).toBe('1')
        expect(isAmount('10.5', 0)).toBe(false)
    })

    it('writes API numbers with the currency decimals', () => {
        expect(numberToAmount(12.5, 2)).toBe('12.50')
        expect(numberToAmount(-0.1, 2)).toBe('-0.10')
        expect(numberToAmount(7, 0)).toBe('7')
    })
})
