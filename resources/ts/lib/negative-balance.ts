import type { Currency } from '@/types'

export type NegativeBalanceWarning = {
    accountName: string
    currentBalance: number
    resultingBalance: number
    currency?: Currency | string | null
}

const OUTFLOW_TYPES = new Set(['expense', 'transfer', 'debt_payment', 'debt_lend'])

export function warningIfResultNegative(options: {
    accountName: string
    currentBalance: number
    resultingBalance: number
    currency?: Currency | string | null
}): NegativeBalanceWarning | null {
    const delta = options.resultingBalance - options.currentBalance
    if (options.resultingBalance >= 0 || delta >= 0 || !options.accountName) {
        return null
    }

    return {
        accountName: options.accountName,
        currentBalance: options.currentBalance,
        resultingBalance: options.resultingBalance,
        currency: options.currency,
    }
}

export function collectNegativeBalanceWarnings(
    candidates: Array<NegativeBalanceWarning | null | undefined>,
): NegativeBalanceWarning[] {
    return candidates.filter((item): item is NegativeBalanceWarning => item != null)
}

export function warningsForAccountOutflow(
    account: { name: string; currentBalance: number; currency?: Currency | null } | undefined,
    amount: number,
): NegativeBalanceWarning[] {
    if (!account || !(amount > 0)) {
        return []
    }

    const warning = warningIfResultNegative({
        accountName: account.name,
        currentBalance: account.currentBalance,
        resultingBalance: account.currentBalance - amount,
        currency: account.currency,
    })

    return warning ? [warning] : []
}

export function warningsForTransactionOutflow(transaction: {
    type: string
    amount: number
    account: { name: string; currentBalance: number; currency?: Currency | null }
}): NegativeBalanceWarning[] {
    if (!OUTFLOW_TYPES.has(transaction.type)) {
        return []
    }

    return warningsForAccountOutflow(transaction.account, transaction.amount)
}

export function warningsForConfirmedDuplicate(transaction: {
    type: string
    amount: number
    status: string
    account: { name: string; currentBalance: number; currency?: Currency | null }
}): NegativeBalanceWarning[] {
    if (transaction.status !== 'confirmed') {
        return []
    }

    return warningsForTransactionOutflow(transaction)
}
