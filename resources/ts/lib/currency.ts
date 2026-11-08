import { getCurrency } from '@/stores/currencies'
import { intlLocale } from '@/lib/i18n'

type CurrencyInput = {
    symbol?: string | null
    code?: string | null
    decimals?: number | null
} | string | null | undefined

type FormatCurrencyOptions = {
    showSymbol?: boolean
    compact?: boolean
}

function resolve(currency?: CurrencyInput): { symbol?: string; decimals: number } {
    const code = typeof currency === 'string' ? currency : currency?.code
    const fromTable = getCurrency(code)
    const symbol = fromTable?.symbol.trim()
        || (typeof currency === 'string' ? currency.trim() : currency?.symbol?.trim())
        || undefined

    return {
        symbol,
        decimals: fromTable?.decimals ?? (typeof currency === 'object' ? currency?.decimals : undefined) ?? 2,
    }
}

function formatNumber(value: number, decimals: number, compact: boolean): string {
    return new Intl.NumberFormat(intlLocale(), {
        style: 'decimal',
        useGrouping: true,
        minimumFractionDigits: compact ? 0 : decimals,
        maximumFractionDigits: compact ? 1 : decimals,
        ...(compact ? { notation: 'compact', compactDisplay: 'short' } : {}),
    }).format(value)
}

export function formatCurrency(
    value: number,
    currency?: CurrencyInput,
    options?: FormatCurrencyOptions
): string {
    const { symbol, decimals } = resolve(currency)
    const amount = formatNumber(value, decimals, options?.compact ?? false)

    if (!(options?.showSymbol ?? true) || !symbol) {
        return amount
    }

    return `${amount} ${symbol}`
}

export function formatCurrencyCompact(
    value: number,
    currency?: CurrencyInput,
    options?: Omit<FormatCurrencyOptions, 'compact'>
): string {
    return formatCurrency(value, currency, { ...options, compact: true })
}

/** A balance in a narrow space: in full below ten thousand, shortened (100K, 1.2M) from there. No symbol. */
export function formatBalanceShort(value: number, currency?: CurrencyInput): string {
    return formatCurrency(value, currency, { showSymbol: false, compact: Math.abs(value) >= 10_000 })
}
