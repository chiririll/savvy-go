import type { Account, Currency, LineInput, PaymentInput, TransactionInput } from '@/types'

/** An amount in one currency, positive; the unit the payload maths works in. */
export interface MoneyRow {
    currency: Currency
    amount: number
}

export interface PaymentRowValues {
    account_id: number
    /** Positive; the type of the transaction gives it its sign. */
    amount: number
}

export interface LineRowValues {
    category_id?: number | null
    /** Left out: the currency of the first payment. */
    currency_id?: number | null
    amount: number
    note?: string | null
}

/** What the form holds that the payload is made of. */
export interface PayloadValues {
    type: 'income' | 'expense'
    description?: string | null
    date?: string | null
    tag_ids?: number[]
    /** The accounts that paid (or received, for an income). */
    payments: PaymentRowValues[]
    /** Money that came back to other accounts: a payment of the opposite sign. */
    change?: PaymentRowValues[]
    /** Lines the user wrote. What they leave uncovered goes to the default category. */
    lines?: LineRowValues[]
    default_category_id?: number | null
    /** The note of the line of the default category; only a transaction saved with one line has it. */
    default_note?: string | null
}

export type PayloadError = 'unknownAccount' | 'duplicateAccount' | 'lineCurrency' | 'amountZero' | 'netSign'

export type PayloadResult = { ok: true; input: TransactionInput } | { ok: false; error: PayloadError }

export const toBase = (currency: Currency): number => (currency.isBase ? 1 : currency.rate || 0)

/** Rounds to the decimals of a currency, so `0.1 + 0.2` is `0.3`. */
export function roundToCurrency(amount: number, currency: Currency): number {
    const factor = 10 ** currency.decimals
    return Math.round((amount + Number.EPSILON * Math.sign(amount)) * factor) / factor
}

export const baseValue = (rows: MoneyRow[]): number => rows.reduce((sum, row) => sum + row.amount * toBase(row.currency), 0)

/** The first account id used twice among the rows, or null. Ids may arrive as numeric strings from a select. */
export function findDuplicateAccount(...groups: Array<Array<{ account_id: number }>>): number | null {
    const seen = new Set<number>()
    for (const row of groups.flat()) {
        const id = Number(row.account_id)
        if (seen.has(id)) {
            return id
        }
        seen.add(id)
    }
    return null
}

/**
 * Whether the payments net to the sign the type needs, in the base currency: an expense must take money and an
 * income bring it, so change worth as much as the payments or more is refused (the API refuses it too). Both lists
 * are magnitudes, so the same holds for an expense and an income.
 */
export function netSignHolds(paid: MoneyRow[], change: MoneyRow[]): boolean {
    return baseValue(paid) - baseValue(change) > 1e-9
}

/**
 * What the payments spent and the lines have not covered, in `target`, never negative. Payments, change and lines
 * are compared in the base currency at the current rates, as the API compares them.
 */
export function remainderAmount(paid: MoneyRow[], lines: MoneyRow[], target: Currency, change: MoneyRow[] = []): number {
    const rate = toBase(target)
    if (rate === 0) {
        return 0
    }
    const left = (baseValue(paid) - baseValue(change) - baseValue(lines)) / rate
    return left > 0 ? Math.max(roundToCurrency(left, target), 0) : 0
}

/** +1 when the payments bring money (income), -1 when they take it (expense). */
export const paymentSign = (type: PayloadValues['type']): 1 | -1 => (type === 'income' ? 1 : -1)

/** The payload the API takes for what the form holds. */
export function buildTransactionInput(values: PayloadValues, accounts: Account[]): PayloadResult {
    const change = values.change ?? []
    const known = (row: PaymentRowValues) => accounts.find((a) => a.id === Number(row.account_id))

    const paidAccounts = values.payments.map(known)
    const changeAccounts = change.map(known)
    if (paidAccounts.some((a) => !a?.currency) || changeAccounts.some((a) => !a?.currency)) {
        return { ok: false, error: 'unknownAccount' }
    }
    if (findDuplicateAccount(values.payments, change) !== null) {
        return { ok: false, error: 'duplicateAccount' }
    }
    if ([...values.payments, ...change].some((row) => !(Number(row.amount) > 0))) {
        return { ok: false, error: 'amountZero' }
    }

    const rows = (list: PaymentRowValues[], accountsOf: Array<Account | undefined>): MoneyRow[] =>
        list.map((row, i) => {
            const currency = accountsOf[i]!.currency!
            return { currency, amount: roundToCurrency(Number(row.amount), currency) }
        })
    const paid = rows(values.payments, paidAccounts)
    const back = rows(change, changeAccounts)
    if (!netSignHolds(paid, back)) {
        return { ok: false, error: 'netSign' }
    }

    const sign = paymentSign(values.type)
    const payment = (row: PaymentRowValues, money: MoneyRow, direction: 1 | -1): PaymentInput => ({
        account_id: Number(row.account_id),
        amount: direction * money.amount,
    })
    const payments = [
        ...values.payments.map((row, i) => payment(row, paid[i], sign)),
        ...change.map((row, i) => payment(row, back[i], sign === 1 ? -1 : 1)),
    ]

    const primary = paid[0].currency
    const currencyOf = (id: number | null | undefined): Currency | undefined =>
        id ? [...paid, ...back].find((row) => row.currency.id === Number(id))?.currency : primary

    const written: LineInput[] = []
    const writtenRows: MoneyRow[] = []
    for (const line of values.lines ?? []) {
        const currency = currencyOf(line.currency_id)
        if (!currency) {
            return { ok: false, error: 'lineCurrency' }
        }
        const amount = roundToCurrency(Number(line.amount), currency)
        writtenRows.push({ currency, amount })
        written.push({
            category_id: line.category_id ?? null,
            currency_id: currency.id,
            amount,
            ...(line.note ? { note: line.note } : {}),
        })
    }

    const rest = remainderAmount(paid, writtenRows, primary, back)
    const lines: LineInput[] = [...written]
    if (rest > 0) {
        lines.push({
            category_id: values.default_category_id ?? null,
            currency_id: primary.id,
            amount: rest,
            ...(values.default_note ? { note: values.default_note } : {}),
        })
    }

    return {
        ok: true,
        input: {
            type: values.type,
            description: values.description?.trim() ? values.description.trim() : null,
            date: values.date ? values.date : null,
            payments,
            lines,
            tag_ids: values.tag_ids ?? [],
        },
    }
}
