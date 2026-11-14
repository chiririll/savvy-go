/**
 * Exact decimal amounts as text. Money never goes through float arithmetic: a value is scaled to an integer of
 * its smallest unit (BigInt), computed, and written back as text.
 */

const AMOUNT = /^(-?)(\d*)(?:\.(\d*))?$/

/** The amount in smallest units of a currency with `decimals`, or null when it is not a number or has more decimals. */
export function toUnits(value: string, decimals: number): bigint | null {
    const match = AMOUNT.exec(value.trim().replace(',', '.'))
    if (!match) return null
    const [, sign, whole, fraction = ''] = match
    if (whole === '' && fraction === '') return null
    if (fraction.length > decimals) return null
    const units = BigInt(`${whole || '0'}${fraction.padEnd(decimals, '0')}`)
    return sign ? -units : units
}

/** Units back to text with exactly `decimals` decimals. */
export function fromUnits(units: bigint, decimals: number): string {
    const negative = units < 0n
    const digits = (negative ? -units : units).toString().padStart(decimals + 1, '0')
    const cut = digits.length - decimals
    const text = decimals > 0 ? `${digits.slice(0, cut)}.${digits.slice(cut)}` : digits
    return negative ? `-${text}` : text
}

/** An amount the API sent as a JSON number, as text with `decimals` decimals. */
export function numberToAmount(value: number, decimals: number): string {
    return value.toFixed(decimals)
}

/** `left - right` as text; null when either is not an amount of the currency. */
export function subtractAmounts(left: string, right: string, decimals: number): string | null {
    const a = toUnits(left, decimals)
    const b = toUnits(right, decimals)
    return a === null || b === null ? null : fromUnits(a - b, decimals)
}

/** Whether the text is a complete amount of the currency (not empty, not a lone sign or point). */
export function isAmount(value: string, decimals: number): boolean {
    return toUnits(value, decimals) !== null
}
