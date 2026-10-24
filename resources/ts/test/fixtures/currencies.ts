// Currency fixtures by ROLE, not by code. A test says "the zero-decimal currency", never "JPY", so a different set
// can be swapped in here without touching the tests. To swap the set, change the codes, symbols, decimals and rates
// in `roles` below; keep one currency per role (base: 2 decimals, zeroDecimal: 0, highPrecision: 8, the ones that
// break float and rounding code). Expected strings come from `formatMoney`, which uses the app's own formatter,
// so they follow the set and the locale. Do not write '1,500 ¥' by hand in new tests.
import { formatCurrency } from '@/lib/currency'
import type { Currency } from '@/types'

import { currency } from '../fixtures'

export const base: Currency = currency({ id: 1, code: 'USD', name: 'US Dollar', symbol: '$', decimals: 2, isBase: true, rate: 1 })

export const zeroDecimal: Currency = currency({
    id: 2,
    code: 'JPY',
    name: 'Japanese Yen',
    symbol: '¥',
    decimals: 0,
    isBase: false,
    rate: 0.005,
})

export const highPrecision: Currency = currency({
    id: 3,
    code: 'BTC',
    name: 'Bitcoin',
    symbol: '₿',
    decimals: 8,
    isBase: false,
    rate: 60000,
})

export const roles = { base, zeroDecimal, highPrecision }

/** The roles as the list the API returns for a space. */
export const roleCurrencies: Currency[] = [base, zeroDecimal, highPrecision]

type Minor = bigint | number

/**
 * An amount in major units (what the API sends) built from minor units: money(150050, base) is 1500.5 for 2 decimals,
 * money(1500, zeroDecimal) is 1500. The digits are split as an integer string, so there is no float division or
 * rounding on the way; the only float is the final parse, the same one JSON.parse does for the API.
 */
export function money(minor: Minor, currencyRole: Pick<Currency, 'decimals'> = base): number {
    if (typeof minor === 'number' && !Number.isSafeInteger(minor)) throw new Error(`money() needs whole minor units, got ${minor}`)
    const value = BigInt(minor)
    const digits = (value < 0n ? -value : value).toString().padStart(currencyRole.decimals + 1, '0')
    const cut = digits.length - currencyRole.decimals
    const text = currencyRole.decimals === 0 ? digits : `${digits.slice(0, cut)}.${digits.slice(cut)}`
    return Number(value < 0n ? `-${text}` : text)
}

/** The text the app shows for an amount, built by the app's own formatter (grouping, decimals, symbol). */
export function formatMoney(amount: number, currencyRole: Currency = base, options?: Parameters<typeof formatCurrency>[2]): string {
    return formatCurrency(amount, currencyRole, options)
}

/** formatMoney for minor units: the expected text of money(minor, role). */
export const expectedMoney = (minor: Minor, currencyRole: Currency = base): string => formatMoney(money(minor, currencyRole), currencyRole)

/** Asserts that `text` is exactly what the app shows for these minor units, e.g. expectMoney(el.textContent, 150050). */
export function expectMoney(text: string | null | undefined, minor: Minor, currencyRole: Currency = base): void {
    const expected = expectedMoney(minor, currencyRole)
    if (text?.trim() !== expected) throw new Error(`Expected money text "${expected}", got "${text?.trim()}"`)
}
