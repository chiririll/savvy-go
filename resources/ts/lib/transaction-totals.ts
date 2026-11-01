import { transactionAmountAppearance } from '@/lib/transaction-description'
import type { Transaction } from '@/types'

export interface CurrencyTotal {
    currency: NonNullable<Transaction['account']['currency']>
    amount: number
}

/**
 * The `limit` largest totals by absolute value in the base currency (rates are base per unit; the
 * base currency itself counts as 1). Ranking only: the amounts themselves stay in their currency.
 */
export function largestTotals(totals: CurrencyTotal[], limit: number): { shown: CurrencyTotal[]; hidden: number } {
    const weight = ({ currency, amount }: CurrencyTotal) =>
        Math.abs(amount) * (currency.isBase ? 1 : currency.rate || 0)
    const shown = [...totals].sort((a, b) => weight(b) - weight(a)).slice(0, limit)

    return { shown, hidden: totals.length - shown.length }
}

/**
 * Net income minus expense per account currency, in order of first appearance. Currencies are never
 * mixed. Skipped rows and plain transfers don't change the balance sheet of a day, so they're left out.
 */
export function netTotalsByCurrency(
    transactions: Pick<Transaction, 'type' | 'status' | 'amount' | 'account'>[],
): CurrencyTotal[] {
    const totals = new Map<string, { currency: CurrencyTotal['currency']; units: number }>()

    for (const { type, status, amount, account } of transactions) {
        const { currency } = account
        const { sign } = transactionAmountAppearance(type, status)
        if (!currency || !sign) {
            continue
        }

        const scale = 10 ** (currency.decimals ?? 2)
        const entry = totals.get(currency.code) ?? { currency, units: 0 }
        entry.units += (sign === '-' ? -1 : 1) * Math.round(amount * scale)
        totals.set(currency.code, entry)
    }

    return [...totals.values()].map(({ currency, units }) => ({
        currency,
        amount: units / 10 ** (currency.decimals ?? 2),
    }))
}
