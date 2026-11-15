import { fromUnits, toUnits } from '@/lib/format/decimal-amount'
import type { Account, Currency } from '@/types'
import { accountOf } from '@/lib/transaction/account-lookup'

interface Row {
    account_id: number
    amount: number
}

/** What the lock reads of the form. Amounts may still be empty, or text, while the user types. */
export interface LockValues {
    payments: Row[]
    change?: Row[]
    lines?: Array<{ currency_id?: number | null; amount: number }>
}

/** The amount in units of the currency; zero for an empty, broken or negative one. */
function units(amount: unknown, decimals: number): bigint {
    const value = toUnits(String(amount ?? ''), decimals)
    return value !== null && value > 0n ? value : 0n
}

type Line = NonNullable<LockValues['lines']>[number]

interface RowLock {
    /**
     * Why the row cannot follow the lines: another payment is in its currency, no line is, or money came back in a
     * currency no payment covers (the real payment is unknown without a rate). Null when it can.
     */
    blocker: 'several' | 'noLine' | 'foreignChange' | null
    currency?: Currency
    /** The lines in the currency of the row; a line that names none is in the first payment's currency. */
    lines: Line[]
}

/** What the lock of a payment row needs to know: its currency, the lines it would follow, and what stops it. */
function rowLock(values: LockValues, accounts: Account[], index: number): RowLock {
    const currency = accountOf(accounts, values.payments[index]?.account_id)?.currency
    const primary = accountOf(accounts, values.payments[0]?.account_id)?.currency
    if (!currency || !primary) {
        return { blocker: 'noLine', lines: [] }
    }
    const shared = values.payments.some((row, i) => i !== index && accountOf(accounts, row?.account_id)?.currency?.id === currency.id)
    const lines = (values.lines ?? []).filter((line) => (line?.currency_id ? Number(line.currency_id) : primary.id) === currency.id)
    const foreign = (values.change ?? []).some((row) => {
        const other = accountOf(accounts, row?.account_id)?.currency
        if (!other || other.id === currency.id || units(row.amount, other.decimals) === 0n) {
            return false
        }
        // Change in the currency of another payment is that payment's to follow; only change nobody pays in is foreign.
        return !values.payments.some((payment) => accountOf(accounts, payment?.account_id)?.currency?.id === other.id)
    })
    const blocker = shared ? 'several' : lines.length === 0 ? 'noLine' : foreign ? 'foreignChange' : null
    return { blocker, currency, lines }
}

/** Why a payment row stays locked, or null when it can follow the lines. */
export const lockBlocker = (values: LockValues, accounts: Account[], index: number): RowLock['blocker'] =>
    rowLock(values, accounts, index).blocker

/**
 * What an unlocked payment is: the lines in its currency, plus the change in that currency (the payments net to the
 * lines). Exact text with the decimals of the currency; empty while those lines have no amount. Change in another
 * currency is not converted (no rate is stored): it blocks the row instead.
 *
 * Null when the row cannot follow the lines and stays locked, see `lockBlocker`.
 */
export function followedAmount(values: LockValues, accounts: Account[], index: number): string | null {
    const { blocker, currency, lines } = rowLock(values, accounts, index)
    if (blocker || !currency) {
        return null
    }
    let total = lines.reduce((sum, line) => sum + units(line.amount, currency.decimals), 0n)
    for (const row of values.change ?? []) {
        if (accountOf(accounts, row?.account_id)?.currency?.id === currency.id) {
            total += units(row.amount, currency.decimals)
        }
    }
    return total > 0n ? fromUnits(total, currency.decimals) : ''
}
