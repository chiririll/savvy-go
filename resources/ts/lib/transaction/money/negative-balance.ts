import type { Currency } from '@/types'

export type NegativeBalanceWarning = {
    accountName: string
    currentBalance: number
    resultingBalance: number
    currency?: Currency | string | null
}

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

/** A duplicate of a confirmed transaction takes the same money again: warn for each account its payments take below zero. */
export function warningsForConfirmedDuplicate(transaction: {
    status: string
    payments: Array<{
        amount: number
        account: { name: string; currentBalance: number; currency?: Currency | null } | null
    }>
}): NegativeBalanceWarning[] {
    if (transaction.status !== 'confirmed') {
        return []
    }

    return transaction.payments.flatMap((payment) =>
        warningsForAccountOutflow(payment.account ?? undefined, payment.amount < 0 ? -payment.amount : 0),
    )
}
