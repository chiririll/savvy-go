import type { Account, Currency } from '@/types'
import {
    baseValue,
    remainderAmount,
    roundToCurrency,
    toBase,
    type MoneyRow,
} from './transaction-payload'

/** The rows of the form this maths reads; amounts may still be empty while the user types. */
export interface CoverageValues {
    default_category_id?: number | null
    payments: Array<{ account_id: number; amount: number }>
    change?: Array<{ account_id: number; amount: number }>
    lines?: Array<{ amount: number; currency_id?: number | null }>
}

/** From two lines on each line has a note; a single line has the title of the transaction instead. */
export const linesNeedNotes = (count: number): boolean => count >= 2

/** What the lines have above what was paid, in `target`, never negative: the lines are for more than the payments. */
export function excessAmount(paid: MoneyRow[], lines: MoneyRow[], target: Currency, change: MoneyRow[] = []): number {
    const rate = toBase(target)
    if (rate === 0) {
        return 0
    }
    const over = (baseValue(lines) - (baseValue(paid) - baseValue(change))) / rate
    return over > 0 ? roundToCurrency(over, target) : 0
}

const accountOf = (accounts: Account[], id: number): Account | undefined => accounts.find((a) => a.id === Number(id))

/** The currencies of the payment and change accounts chosen so far, each once: what a line can be in. */
export function paymentCurrencies(values: Pick<CoverageValues, 'payments' | 'change'>, accounts: Account[]): Currency[] {
    const found: Currency[] = []
    for (const row of [...values.payments, ...(values.change ?? [])]) {
        const currency = accountOf(accounts, row?.account_id)?.currency
        if (currency && !found.some((c) => c.id === currency.id)) {
            found.push(currency)
        }
    }
    return found
}

export interface LineCoverage {
    /** The currency lines are in unless they say another: the first payment's. Null before an account is chosen. */
    currency: Currency | null
    /** What the payments net to, less the change, in `currency`: what the lines are to cover. */
    total: number
    /** What the lines leave uncovered, in `currency`: it goes to the default category. */
    remainder: number
    /** What the lines have above the payments, in `currency`. */
    excess: number
}

const positive = (amount: unknown): number => {
    const value = Number(amount)
    return value > 0 ? value : 0
}

/** How far the lines the user wrote cover the payments, compared in the base currency at the current rates. */
export function lineCoverage(values: CoverageValues, accounts: Account[]): LineCoverage {
    const rows = (list: CoverageValues['payments']): MoneyRow[] =>
        list.flatMap((row) => {
            const currency = accountOf(accounts, row?.account_id)?.currency
            const amount = positive(row?.amount)
            return currency && amount > 0 ? [{ currency, amount: roundToCurrency(amount, currency) }] : []
        })
    const paid = rows(values.payments)
    const back = rows(values.change ?? [])

    const primary = accountOf(accounts, values.payments[0]?.account_id)?.currency ?? null
    if (!primary) {
        return { currency: null, total: 0, remainder: 0, excess: 0 }
    }

    const currencies = paymentCurrencies(values, accounts)
    const written: MoneyRow[] = (values.lines ?? []).flatMap((line) => {
        const currency = line.currency_id ? currencies.find((c) => c.id === Number(line.currency_id)) : primary
        const amount = positive(line.amount)
        return currency && amount > 0 ? [{ currency, amount: roundToCurrency(amount, currency) }] : []
    })

    return {
        currency: primary,
        total: remainderAmount(paid, [], primary, back),
        remainder: remainderAmount(paid, written, primary, back),
        excess: excessAmount(paid, written, primary, back),
    }
}

/** A line as the form holds it; the amount is empty until the user types it. */
export interface NewLine {
    category_id: number | null
    currency_id: number | null
    /** An empty string until typed; the form's inputs hold it so. */
    amount: number
    note: string
}

const EMPTY_AMOUNT = '' as unknown as number

const emptyLine = (): NewLine => ({ category_id: null, currency_id: null, amount: EMPTY_AMOUNT, note: '' })

/**
 * The lines the (+) button adds. The first press makes two: the first takes the whole payment and the default
 * category, so nothing changes yet, and the second is empty, ready to take its part off the first. After that
 * each press adds an empty one.
 */
export function linesToAdd(values: CoverageValues, accounts: Account[]): NewLine[] {
    if (values.lines?.length) {
        return [emptyLine()]
    }
    const { total } = lineCoverage({ ...values, lines: [] }, accounts)
    return [
        { category_id: values.default_category_id ?? null, currency_id: null, amount: total > 0 ? total : EMPTY_AMOUNT, note: '' },
        emptyLine(),
    ]
}
