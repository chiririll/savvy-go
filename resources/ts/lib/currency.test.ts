import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Currency } from '@/types'
import { useCurrenciesStore } from '@/stores/currencies'
import { formatCurrency, formatCurrencyCompact } from './currency'

// The real i18n module pulls in the language detector and every locale bundle.
vi.mock('@/lib/i18n', () => ({ intlLocale: () => 'en-US' }))

const currency = (code: string, symbol: string, decimals: number) =>
    ({ code, symbol, decimals }) as unknown as Currency

beforeEach(() => {
    useCurrenciesStore.getState().setAll([
        currency('USD', '$', 2),
        currency('JPY', '¥', 0),
        currency('BHD', 'BD', 3),
    ])
})

describe('formatCurrency', () => {
    it('uses the decimals and symbol of the currency table', () => {
        expect(formatCurrency(1234.5, 'USD')).toBe('1,234.50 $')
        expect(formatCurrency(1234, 'JPY')).toBe('1,234 ¥')
        expect(formatCurrency(1.5, 'BHD')).toBe('1.500 BD')
    })

    it('looks the code up case-insensitively and ignoring whitespace', () => {
        expect(formatCurrency(1, ' usd ')).toBe('1.00 $')
        expect(formatCurrency(1, { code: 'jpy' })).toBe('1 ¥')
    })

    it('prefers the table over what the object carries', () => {
        expect(formatCurrency(1, { code: 'USD', symbol: 'X', decimals: 5 })).toBe('1.00 $')
    })

    it('falls back to the object for a currency missing from the table', () => {
        expect(formatCurrency(1, { code: 'XYZ', symbol: 'Z', decimals: 4 })).toBe('1.0000 Z')
    })

    it('treats an unknown string as the symbol and defaults to two decimals', () => {
        expect(formatCurrency(5, 'XYZ')).toBe('5.00 XYZ')
    })

    it('omits the symbol when there is none, or when asked to', () => {
        expect(formatCurrency(5)).toBe('5.00')
        expect(formatCurrency(5, null)).toBe('5.00')
        expect(formatCurrency(5, { code: 'XYZ' })).toBe('5.00')
        expect(formatCurrency(5, 'USD', { showSymbol: false })).toBe('5.00')
    })

    it('keeps the sign of negative amounts', () => {
        expect(formatCurrency(-1234.5, 'USD')).toBe('-1,234.50 $')
    })

    it('rounds to the currency precision', () => {
        expect(formatCurrency(1.4, 'JPY')).toBe('1 ¥')
        expect(formatCurrency(1.6, 'JPY')).toBe('2 ¥')
        expect(formatCurrency(2.346, 'USD')).toBe('2.35 $')
    })
})

describe('formatCurrencyCompact', () => {
    it('abbreviates large amounts and drops trailing zeros', () => {
        expect(formatCurrencyCompact(1500, 'USD')).toBe('1.5K $')
        expect(formatCurrencyCompact(2_000_000, 'USD')).toBe('2M $')
        expect(formatCurrencyCompact(999, 'USD')).toBe('999 $')
    })

    it('still honours showSymbol', () => {
        expect(formatCurrencyCompact(1500, 'USD', { showSymbol: false })).toBe('1.5K')
    })
})
