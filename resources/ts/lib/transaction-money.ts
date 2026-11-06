import { useCurrenciesStore } from '@/stores/currencies'
import type { Category, Currency, Transaction, TransactionLine } from '@/types'

/** An amount in one currency. Currencies are never mixed: a transaction has one total per currency. */
export interface CurrencyAmount {
    currency: Currency
    amount: number
}

type WithPayments = Pick<Transaction, 'payments'>
type WithLines = Pick<Transaction, 'payments' | 'lines'>

/** Sums in whole minor units, so cents never drift; returns in order of first appearance. */
function sumByCurrency(entries: Array<{ currency: Currency | null; amount: number }>): CurrencyAmount[] {
    const totals = new Map<string, { currency: Currency; units: number }>()

    for (const { currency, amount } of entries) {
        if (!currency) {
            continue
        }
        const entry = totals.get(currency.code) ?? { currency, units: 0 }
        entry.units += Math.round(amount * 10 ** currency.decimals)
        totals.set(currency.code, entry)
    }

    return [...totals.values()].map(({ currency, units }) => ({ currency, amount: units / 10 ** currency.decimals }))
}

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

/** What the payments net to, per currency. A zero total (an exchange that came out even) is kept. */
export function netByCurrency(transaction: WithPayments): CurrencyAmount[] {
    return sumByCurrency(transaction.payments.map((p) => ({ currency: p.account?.currency ?? null, amount: p.amount })))
}

/** What was spent or earned, per currency, from the lines. Change never shows here. */
export function spentByCurrency(transaction: WithLines): CurrencyAmount[] {
    return sumByCurrency(
        transaction.lines.map((line) => ({ currency: lineCurrency(transaction, line), amount: line.amount })),
    )
}

/** The value of an amount in the base currency, for ranking only (rates are base per unit). */
function baseWeight({ currency, amount }: CurrencyAmount): number {
    return Math.abs(amount) * (currency.isBase ? 1 : currency.rate || 0)
}

/** The largest total by value in the base currency; the "main" currency of a row. */
export function mainAmount(totals: CurrencyAmount[]): CurrencyAmount | null {
    return totals.reduce<CurrencyAmount | null>(
        (best, total) => (best === null || baseWeight(total) > baseWeight(best) ? total : best),
        null,
    )
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
