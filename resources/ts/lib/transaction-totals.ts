import { transactionAmountAppearance } from '@/lib/transaction-description'
import { rowAmount, spentByCurrency } from '@/lib/transaction-money'
import type { Currency } from '@/types'
import type { Transaction } from '@/types'

export interface CurrencyTotal {
    currency: Currency
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
 * Net income minus expense per currency, in order of first appearance. Currencies are never mixed.
 * Skipped rows don't change the balance sheet of a day. A plain transfer moves money between accounts
 * and adds nothing, but its fee lines are spending.
 */
export function netTotalsByCurrency(
    transactions: Pick<Transaction, 'type' | 'status' | 'payments' | 'lines'>[],
): CurrencyTotal[] {
    const totals = new Map<string, { currency: CurrencyTotal['currency']; units: number }>()

    for (const transaction of transactions) {
        if (transaction.status === 'skipped') {
            continue
        }
        const isTransfer = transaction.type === 'transfer'
        const sign = isTransfer ? '-' : transactionAmountAppearance(transaction.type).sign
        const parts = isTransfer ? spentByCurrency(transaction) : (rowAmount(transaction)?.totals ?? [])
        if (!sign) {
            continue
        }

        for (const { currency, amount } of parts) {
            const scale = 10 ** currency.decimals
            const entry = totals.get(currency.code) ?? { currency, units: 0 }
            entry.units += (sign === '-' ? -1 : 1) * Math.round(Math.abs(amount) * scale)
            totals.set(currency.code, entry)
        }
    }

    return [...totals.values()].map(({ currency, units }) => ({
        currency,
        amount: units / 10 ** currency.decimals,
    }))
}
