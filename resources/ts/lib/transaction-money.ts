import { useCurrenciesStore } from '@/stores/currencies'
import type { Category, Currency, CurrencyAmount, Transaction, TransactionLine } from '@/types'

type WithPayments = Pick<Transaction, 'payments'>
type WithLines = Pick<Transaction, 'payments' | 'lines'>

/**
 * The currency of a line: it is one of the payment accounts' currencies, so those are looked at first,
 * then the currencies of the space. Null only when neither knows it.
 */
export function lineCurrency(transaction: WithPayments, line: Pick<TransactionLine, 'currencyId'>): Currency | null {
    for (const { account } of transaction.payments) {
        if (account?.currency?.id === line.currencyId) {
            return account.currency
        }
    }

    return Object.values(useCurrenciesStore.getState().byCode).find((c) => c.id === line.currencyId) ?? null
}

/** The value of an amount in the base currency, for ranking only (rates are base per unit). */
function baseWeight({ currency, amount }: CurrencyAmount): number {
    return Math.abs(amount) * (currency.isBase ? 1 : currency.rate || 0)
}

/**
 * The category that stands for the transaction: the one of its largest line by value in the base
 * currency (the first on a tie). Lines without a category are skipped; null when none has one.
 */
export function primaryCategory(transaction: WithLines): Category | null {
    let best: { category: Category; weight: number } | null = null

    for (const line of transaction.lines) {
        const currency = lineCurrency(transaction, line)
        if (!line.category || !currency) {
            continue
        }
        const weight = baseWeight({ currency, amount: line.amount })
        if (best === null || weight > best.weight) {
            best = { category: line.category, weight }
        }
    }

    return best?.category ?? null
}
