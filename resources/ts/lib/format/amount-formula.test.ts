import { describe, expect, it } from 'vitest'

import { evaluateAmount, isFormula, sanitizeAmountInput } from './amount-formula'

describe('isFormula', () => {
    it.each([
        ['12', false],
        ['12.5', false],
        ['12,5', false],
        ['', false],
        ['-12', false],
        ['12+3', true],
        ['12 - 3', true],
        ['2*3', true],
        ['10/4', true],
        ['(1)', true],
        ['=5', true],
    ])('%j -> %j', (text, expected) => {
        expect(isFormula(text)).toBe(expected)
    })
})

describe('evaluateAmount', () => {
    it('leaves a plain number alone', () => {
        expect(evaluateAmount('12.5', 2)).toEqual({ kind: 'plain' })
        expect(evaluateAmount('', 2)).toEqual({ kind: 'plain' })
    })

    it.each([
        ['1+2', 2, '3.00'],
        ['10-2.5', 2, '7.50'],
        ['12,5*3', 2, '37.50'],
        ['(40-5)/2', 2, '17.50'],
        ['12,5*3 + (40-5)/2', 2, '55.00'],
        ['2+3*4', 2, '14.00'],
        ['=1+1', 2, '2.00'],
        [' 1 + 1 ', 2, '2.00'],
        ['5-(-2)', 2, '7.00'],
        ['10/4', 0, '3'],
        ['0.1+0.2', 2, '0.30'],
        ['10/3', 2, '3.33'],
        ['20/3', 2, '6.67'],
        ['1/8', 2, '0.13'],
        ['1/3*3', 2, '1.00'],
    ])('%j (%i decimals) = %j', (text, decimals, expected) => {
        expect(evaluateAmount(text, decimals)).toEqual({ kind: 'value', value: expected })
    })

    // The decimals of the currency decide the rounding: JPY has none, USD two, KWD three.
    it.each([
        ['10/4', 0, '3'],
        ['10/3', 0, '3'],
        ['5/2', 0, '3'],
        ['7/2*2', 0, '7'],
        ['1/3', 3, '0.333'],
        ['2/3', 3, '0.667'],
        ['1/8', 3, '0.125'],
        ['1/16', 3, '0.063'],
        ['1/16', 2, '0.06'],
        ['0.005+0', 2, '0.01'],
        ['0.0049+0', 2, '0.00'],
        ['1000/3', 0, '333'],
        ['0.1+0.2', 3, '0.300'],
        ['12,5*3', 3, '37.500'],
        // Bitcoin: eight decimals.
        ['1/3', 8, '0.33333333'],
        ['2/3', 8, '0.66666667'],
        ['0.00000001*3', 8, '0.00000003'],
        ['0.1+0.2', 8, '0.30000000'],
        ['0.000000005+0', 8, '0.00000001'],
        ['0.0000000049+0', 8, '0.00000000'],
        ['21000000/3', 8, '7000000.00000000'],
    ])('rounds %j to %i decimals as %j', (text, decimals, expected) => {
        expect(evaluateAmount(text, decimals)).toEqual({ kind: 'value', value: expected })
    })

    it('stays exact on big sums', () => {
        expect(evaluateAmount('9007199254740993+1', 2)).toEqual({ kind: 'value', value: '9007199254740994.00' })
    })

    it.each(['1+', '(1+2', '1+2)', '1/0', '1++', '*2', '1..2+1', '5-10', '1 2+3', '()'])(
        'refuses %j',
        (text) => {
            expect(evaluateAmount(text, 2)).toEqual({ kind: 'error' })
        },
    )
})

describe('sanitizeAmountInput', () => {
    it('keeps a plain amount as sanitizeAmount does', () => {
        expect(sanitizeAmountInput('12,567', 2)).toBe('12.56')
        expect(sanitizeAmountInput('-12', 2)).toBe('12')
        expect(sanitizeAmountInput('abc', 2)).toBe('')
    })

    it('lets a formula through, without letters', () => {
        expect(sanitizeAmountInput('12,5*3a + (4/2)', 2)).toBe('12,5*3 + (4/2)')
        expect(sanitizeAmountInput('1.234+1', 2)).toBe('1.234+1')
    })
})
