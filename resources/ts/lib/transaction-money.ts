import { useCurrenciesStore } from '@/stores/currencies'
import {
    isServiceAccountType,
    type Category,
    type Currency,
    type CurrencyAmount,
    type Payment,
    type RowAmount,
    type Transaction,
    type TransactionLine,
} from '@/types'

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

/*
 * `rowAmount`, `transferSides` and `convertAmount` below are for a recurring rule's template only: the server
 * works the same figures out for every transaction (`Transaction.rowAmount`), but a rule has no transaction.
 */

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

/** An amount in another currency, by the space's rates (base per unit), rounded to the target's decimals. */
export function convertAmount(amount: number, from: Currency, to: Currency): number {
    if (from.code === to.code) {
        return amount
    }
    const toBase = (currency: Currency) => (currency.isBase ? 1 : currency.rate || 0)
    if (toBase(to) === 0) {
        return amount
    }
    return Number(((amount * toBase(from)) / toBase(to)).toFixed(to.decimals))
}

function toRowAmount(totals: CurrencyAmount[]): RowAmount | null {
    if (totals.length === 0) {
        return null
    }
    const [first] = totals
    if (totals.length === 1) {
        return { main: first, totals, converted: false }
    }
    const sum = totals.reduce((acc, t) => acc + convertAmount(t.amount, t.currency, first.currency), 0)
    return { main: { currency: first.currency, amount: sum }, totals, converted: true }
}

const isOrdinary = (payment: Payment) => !!payment.account && !isServiceAccountType(payment.account.type)

/**
 * The amount of a row, unsigned: what was spent or earned, from the lines. Without lines (a debt
 * payment) it is the payments on ordinary accounts. Null when there is nothing to show.
 */
export function rowAmount(transaction: WithLines): RowAmount | null {
    const spent = spentByCurrency(transaction).map((t) => ({ ...t, amount: Math.abs(t.amount) }))
    if (spent.length > 0) {
        return toRowAmount(spent)
    }
    const ordinary = transaction.payments.filter(isOrdinary)
    return toRowAmount(
        netByCurrency({ payments: ordinary }).map((t) => ({ ...t, amount: Math.abs(t.amount) })),
    )
}

/** What left and what arrived on the ordinary accounts of a transfer; each is null when that side is missing. */
export function transferSides(transaction: WithPayments): { sent: RowAmount | null; received: RowAmount | null } {
    const ordinary = transaction.payments.filter(isOrdinary)
    const side = (sign: 1 | -1) =>
        toRowAmount(
            netByCurrency({ payments: ordinary.filter((p) => Math.sign(p.amount) === sign) }).map((t) => ({
                ...t,
                amount: Math.abs(t.amount),
            })),
        )
    return { sent: side(-1), received: side(1) }
}
