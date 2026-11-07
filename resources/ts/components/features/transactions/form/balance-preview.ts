import { useMemo } from 'react'
import { useWatch, type Control } from 'react-hook-form'
import type { TransactionFormValues } from '@/schemas/transactions'
import { collectNegativeBalanceWarnings, warningIfResultNegative } from '@/lib/negative-balance'
import { paymentSign } from '@/lib/transaction-payload'
import type { Account, Currency, Transaction } from '@/types'

/** Money that moves on one account: signed, in the account's currency. */
export type AccountImpact = { accountId: number; delta: number }

export type BalancePreview = {
    account: Account
    currentBalance: number
    newBalance: number
    insufficientFunds: boolean
    currency: Currency | null
}

type ImpactValues = {
    type?: TransactionFormValues['type']
    payments?: Array<{ account_id?: number | string | null; amount?: number | string | null }>
    change?: Array<{ account_id?: number | string | null; amount?: number | string | null }>
}

/** What the form does to the accounts: the payments by the sign of the type, the change by the opposite one. */
export function formImpact(values: ImpactValues): AccountImpact[] {
    const sign = values.type ? paymentSign(values.type) : 0
    const rows = (list: ImpactValues['payments'], direction: number) =>
        (list ?? []).flatMap((row) => {
            const accountId = Number(row.account_id)
            const amount = Number(row.amount)
            return accountId > 0 && amount > 0 ? [{ accountId, delta: direction * amount }] : []
        })

    return [...rows(values.payments, sign), ...rows(values.change, -sign)]
}

/** What a saved transaction did to the accounts, if it counts in their balances. */
export function postedImpact(transaction: Pick<Transaction, 'payments'>): AccountImpact[] {
    return transaction.payments.map((p) => ({ accountId: p.accountId, delta: p.amount }))
}

const sumOn = (impact: AccountImpact[] | null, accountId: number) =>
    (impact ?? []).filter((i) => i.accountId === accountId).reduce((sum, i) => sum + i.delta, 0)

/** Balances before and after saving, for each account the next payments touch. */
export function projectBalances(
    accounts: Account[],
    next: AccountImpact[],
    posted: AccountImpact[] | null,
): BalancePreview[] {
    const ids = [...new Set(next.map((i) => i.accountId))]

    return ids.flatMap((id) => {
        const account = accounts.find((a) => a.id === id)
        if (!account) {
            return []
        }
        const decimals = account.currency?.decimals ?? 2
        const factor = 10 ** decimals
        const units = Math.round(account.currentBalance * factor)
            - Math.round(sumOn(posted, id) * factor)
            + Math.round(sumOn(next, id) * factor)
        const newBalance = units / factor

        return [{
            account,
            currentBalance: account.currentBalance,
            newBalance,
            insufficientFunds: sumOn(next, id) < 0 && newBalance < 0,
            currency: account.currency,
        }]
    })
}

interface BalancePreviewOptions {
    control: Control<TransactionFormValues>
    accounts?: Account[]
    /** The edited transaction as it is posted now, if it affects balances. */
    posted: AccountImpact[] | null
    /** Whether saving changes balances, so negative results need confirming. */
    willPost: boolean
}

/** Account balances before and after saving the form, and what goes negative. */
export function useBalancePreview({ control, accounts, posted, willPost }: BalancePreviewOptions) {
    const [type, payments, change] = useWatch({ control, name: ['type', 'payments', 'change'] })

    return useMemo(() => {
        const rows = projectBalances(accounts ?? [], formImpact({ type, payments, change }), posted)
        const warnings = willPost
            ? collectNegativeBalanceWarnings(
                rows.map((row) =>
                    warningIfResultNegative({
                        accountName: row.account.name,
                        currentBalance: row.currentBalance,
                        resultingBalance: row.newBalance,
                        currency: row.currency,
                    }),
                ),
            )
            : []

        return { rows, warnings }
    }, [type, payments, change, accounts, posted, willPost])
}
