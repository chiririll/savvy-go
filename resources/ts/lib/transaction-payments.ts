import type { Account, Currency } from '@/types'
import {
    findDuplicateAccount,
    netSignHolds,
    roundToCurrency,
    type MoneyRow,
    type PayloadError,
    type PayloadValues,
} from './transaction-payload'

/** A payment row as the form holds it; the amount is empty until the user types it. */
interface Row {
    account_id: number
    amount: number
}

export interface PaymentRowsValues {
    payments: Row[]
    change?: Row[]
}

const accountOf = (accounts: Account[], id: unknown): Account | undefined =>
    accounts.find((a) => a.id === Number(id))

const positive = (amount: unknown): number => {
    const value = Number(amount)
    return value > 0 ? value : 0
}

/** The rows that have an account and an amount, as money in the currency of their account. */
function moneyRows(rows: Row[] | undefined, accounts: Account[]): MoneyRow[] {
    return (rows ?? []).flatMap((row) => {
        const currency = accountOf(accounts, row?.account_id)?.currency
        const amount = positive(row?.amount)
        return currency && amount > 0 ? [{ currency, amount: roundToCurrency(amount, currency) }] : []
    })
}

export interface CurrencyTotal {
    currency: Currency
    /** What the payments add up to in this currency. */
    paid: number
    /** What came back as change in this currency. */
    back: number
    /** Paid less change: below zero when only change came back in this currency. */
    net: number
}

/** One total per currency, in the order the currencies first appear: payments first, then the change. */
export function paymentTotals(values: PaymentRowsValues, accounts: Account[]): CurrencyTotal[] {
    const totals = new Map<number, CurrencyTotal>()
    const add = (rows: MoneyRow[], key: 'paid' | 'back') => {
        for (const { currency, amount } of rows) {
            const total = totals.get(currency.id) ?? { currency, paid: 0, back: 0, net: 0 }
            total[key] = roundToCurrency(total[key] + amount, currency)
            total.net = roundToCurrency(total.paid - total.back, currency)
            totals.set(currency.id, total)
        }
    }
    add(moneyRows(values.payments, accounts), 'paid')
    add(moneyRows(values.change, accounts), 'back')
    return [...totals.values()]
}

/** The first account no row uses: an account is in a transaction once. Undefined means the (+) is off. */
export function nextFreeAccount(accounts: Account[] | undefined, usedIds: number[]): Account | undefined {
    const used = new Set(usedIds.map(Number))
    return accounts?.find((a) => !used.has(a.id))
}

/**
 * Whether the change is worth as much as the payments or more, in the base currency: the API refuses a transaction
 * whose payments do not net to the sign of its type. False until both sides have an amount.
 */
export function changeTooLarge(values: PaymentRowsValues, accounts: Account[]): boolean {
    const paid = moneyRows(values.payments, accounts)
    const back = moneyRows(values.change, accounts)
    return paid.length > 0 && back.length > 0 && !netSignHolds(paid, back)
}

/** The field of the form a refused payload belongs to, so the message stands next to what the user has to change. */
export function payloadErrorPath(error: PayloadError, values: PayloadValues, accounts: Account[]): string {
    const rows = [
        ...values.payments.map((row, index) => ({ row, list: 'payments', index })),
        ...(values.change ?? []).map((row, index) => ({ row, list: 'change', index })),
    ]
    switch (error) {
        case 'unknownAccount': {
            const bad = rows.find(({ row }) => !accountOf(accounts, row.account_id)?.currency) ?? rows[0]
            return `${bad.list}.${bad.index}.account_id`
        }
        case 'duplicateAccount': {
            const twice = findDuplicateAccount(values.payments, values.change ?? [])
            const seen = rows.filter(({ row }) => Number(row.account_id) === twice)
            const second = seen[1] ?? rows[0]
            return `${second.list}.${second.index}.account_id`
        }
        case 'amountZero': {
            const zero = rows.find(({ row }) => !(Number(row.amount) > 0)) ?? rows[0]
            return `${zero.list}.${zero.index}.amount`
        }
        case 'netSign':
            return (values.change ?? []).length > 0 ? 'change.0.amount' : 'payments.0.amount'
        case 'lineCurrency': {
            const currencies = rows.flatMap(({ row }) => accountOf(accounts, row.account_id)?.currency?.id ?? [])
            const index = (values.lines ?? []).findIndex(
                (line) => line.currency_id && !currencies.includes(Number(line.currency_id)),
            )
            return `lines.${Math.max(index, 0)}.currency_id`
        }
    }
}
