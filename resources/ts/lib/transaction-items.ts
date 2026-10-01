export const ITEM_QTY_DECIMALS = 3

export function currencyDecimals(currency?: { decimals?: number | null } | null): number {
    return currency?.decimals ?? 2
}

export function roundMoney(value: number, decimals: number): number {
    const factor = 10 ** Math.max(0, decimals)
    return Math.round((Number(value) + Number.EPSILON) * factor) / factor
}

export function roundedItemPrice(price: number, decimals: number): number {
    return roundMoney(Number(price) || 0, decimals)
}

export function itemContribution(quantity: number, price: number, decimals: number): number {
    return roundedItemPrice(price, decimals) * (Number(quantity) || 0)
}

export function sumTransactionItems(
    items: Array<{ quantity?: number | string | null; price_per_unit?: number | string | null }> | undefined,
    decimals: number,
): number {
    const sum = (items ?? []).reduce((total, item) => (
        total + itemContribution(Number(item?.quantity) || 0, Number(item?.price_per_unit) || 0, decimals)
    ), 0)

    return roundMoney(sum, decimals)
}

export function quantityDecimalPlaces(value: number): number {
    if (!Number.isFinite(value)) {
        return Number.POSITIVE_INFINITY
    }

    const text = String(value)
    if (/[eE]/.test(text)) {
        const [base, exponent] = text.toLowerCase().split('e')
        const baseDecimals = base.includes('.') ? base.split('.')[1].length : 0
        return Math.max(0, baseDecimals - Number(exponent))
    }

    const dot = text.indexOf('.')
    return dot === -1 ? 0 : text.length - dot - 1
}
