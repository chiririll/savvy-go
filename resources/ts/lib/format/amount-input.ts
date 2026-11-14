/**
 * What an amount field lets through: digits and one decimal separator, with at most `decimals` digits after it.
 * A comma counts as the separator, so a Russian keyboard works; the result always uses a point. The value stays
 * text, so nothing is rounded on the way.
 */
export function sanitizeAmount(raw: string, decimals: number): string {
    const normalized = raw.replace(',', '.').replace(/[^\d.]/g, '')
    const [whole, ...rest] = normalized.split('.')
    if (rest.length === 0 || decimals <= 0) {
        return whole
    }
    return `${whole}.${rest.join('').slice(0, decimals)}`
}
