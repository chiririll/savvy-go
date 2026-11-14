import type { Account, ReconciliationInput } from '@/types'
import { isServiceAccountType } from '@/types'
import { isAmount, numberToAmount, subtractAmounts } from '@/lib/format/decimal-amount'

/** One account of the reconcile dialog. Amounts are text, so nothing is rounded on the way. */
export interface ReconcileRow {
    accountId: number
    /** What the user sees and sends: the computed balance until they type. */
    amount: string
    /** The ledger's balance at the chosen date; null while it is loading. */
    computed: string | null
    /** The user typed in the row: a new date does not overwrite it. */
    edited: boolean
}

/** Debt and space accounts are kept by the app and cannot be reconciled. Order is the order of the list. */
export function reconcilableAccounts(accounts: readonly Account[]): Account[] {
    return accounts.filter((account) => !isServiceAccountType(account.type) && account.currency !== null)
}

export function decimalsOf(account: Account): number {
    return account.currency?.decimals ?? 2
}

/** Every account is a row, with its current balance as the amount. */
export function initialRows(accounts: readonly Account[]): ReconcileRow[] {
    return reconcilableAccounts(accounts).map((account) => {
        const balance = numberToAmount(account.currentBalance, decimalsOf(account))
        return { accountId: account.id, amount: balance, computed: balance, edited: false }
    })
}

export function removeRow(rows: readonly ReconcileRow[], accountId: number): ReconcileRow[] {
    return rows.filter((row) => row.accountId !== accountId)
}

/** Accounts that are not rows (anymore), in the order of the list. */
export function removedAccounts(accounts: readonly Account[], rows: readonly ReconcileRow[]): Account[] {
    const present = new Set(rows.map((row) => row.accountId))
    return reconcilableAccounts(accounts).filter((account) => !present.has(account.id))
}

/**
 * Puts an account back where the list has it. `balance` is its computed balance at the chosen date, or null when
 * that is not known yet (the row is filled when it arrives).
 */
export function addRow(
    rows: readonly ReconcileRow[],
    accounts: readonly Account[],
    accountId: number,
    balance: string | null,
): ReconcileRow[] {
    if (rows.some((row) => row.accountId === accountId)) return [...rows]
    const row: ReconcileRow = { accountId, amount: balance ?? '', computed: balance, edited: false }
    const order = new Map(reconcilableAccounts(accounts).map((account, index) => [account.id, index]))
    const next = [...rows, row]
    return next.sort((a, b) => (order.get(a.accountId) ?? 0) - (order.get(b.accountId) ?? 0))
}

export function editRow(rows: readonly ReconcileRow[], accountId: number, amount: string): ReconcileRow[] {
    return rows.map((row) => (row.accountId === accountId ? { ...row, amount, edited: true } : row))
}

/**
 * Applies the computed balances of a date (by account id). Rows the user has not edited take the value as their
 * amount too; edited rows keep what was typed.
 */
export function applyComputed(rows: readonly ReconcileRow[], balances: ReadonlyMap<number, string>): ReconcileRow[] {
    return rows.map((row) => {
        const computed = balances.get(row.accountId)
        if (computed === undefined) return row
        return row.edited ? { ...row, computed } : { ...row, computed, amount: computed }
    })
}

/** Typed minus computed, as exact text; null while either is missing or incomplete. */
export function rowDifference(row: ReconcileRow, decimals: number): string | null {
    if (row.computed === null) return null
    return subtractAmounts(row.amount, row.computed, decimals)
}

export function isRowValid(row: ReconcileRow, decimals: number): boolean {
    return isAmount(row.amount, decimals)
}

/** Ready to save: a date that is not in the future, at least one row, every row a complete amount. */
export function canSave(
    date: string,
    today: string,
    rows: readonly ReconcileRow[],
    decimalsByAccount: ReadonlyMap<number, number>,
): boolean {
    if (!date || date > today || rows.length === 0) return false
    return rows.every((row) => isRowValid(row, decimalsByAccount.get(row.accountId) ?? 2))
}

/** The batch the API takes. The amount goes out as the JSON number of the text typed. */
export function toPayload(date: string, rows: readonly ReconcileRow[]): ReconciliationInput {
    return {
        date,
        items: rows.map((row) => ({ accountId: row.accountId, actualBalance: Number(row.amount.replace(',', '.')) })),
    }
}
