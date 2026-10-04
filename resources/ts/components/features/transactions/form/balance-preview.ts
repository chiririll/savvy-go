import { useMemo } from 'react'
import { useWatch, type Control } from 'react-hook-form'
import { isTransferType, type TransactionFormValues } from '@/schemas/transactions'
import { collectNegativeBalanceWarnings, warningIfResultNegative } from '@/lib/negative-balance'
import type { formatCurrency } from '@/lib/utils'
import type { Account } from '@/types'

export type BalancePreview = {
    currentBalance: number
    newBalance: number
    insufficientFunds?: boolean
    currency: Parameters<typeof formatCurrency>[1]
}

/** What a transaction does to account balances. */
export type BalanceImpact = {
    type?: TransactionFormValues['type']
    amount: number
    accountId: number | null
    toAccountId: number | null
    toAmount: number
}

const SOURCE_SIGN = { income: 1, expense: -1, transfer: -1, transfer_out: -1 } as const

export function toBalanceImpact(values: Partial<TransactionFormValues>): BalanceImpact {
    return {
        type: values.type,
        amount: Number(values.amount) || 0,
        accountId: Number(values.account_id) || null,
        toAccountId: Number(values.to_account_id) || null,
        toAmount: Number(values.to_amount) || Number(values.amount) || 0,
    }
}

function impactOn(accountId: number, tx: BalanceImpact, side: 'source' | 'destination'): number {
    if (side === 'destination') {
        return isTransferType(tx.type) && tx.toAccountId === accountId ? tx.toAmount : 0
    }

    return tx.accountId === accountId && tx.type
        ? SOURCE_SIGN[tx.type] * tx.amount
        : 0
}

function project(
    account: Account,
    next: BalanceImpact,
    posted: BalanceImpact | null,
    side: 'source' | 'destination',
): BalancePreview {
    const currentBalance = account.currentBalance
    return {
        currentBalance,
        newBalance: currentBalance
            - (posted ? impactOn(account.id, posted, side) : 0)
            + impactOn(account.id, next, side),
        currency: account.currency,
    }
}

function negativeWarning(account?: Account, preview?: BalancePreview | null) {
    return account && preview
        ? warningIfResultNegative({
            accountName: account.name,
            currentBalance: preview.currentBalance,
            resultingBalance: preview.newBalance,
            currency: account.currency,
        })
        : null
}

interface BalancePreviewOptions {
    control: Control<TransactionFormValues>
    from?: Account
    to?: Account
    /** The edited transaction as it is posted now, if it affects balances. */
    posted: BalanceImpact | null
    /** Whether saving changes balances, so negative results need confirming. */
    willPost: boolean
}

/** Account balances before and after saving the form, and what goes negative. */
export function useBalancePreview({ control, from, to, posted, willPost }: BalancePreviewOptions) {
    const [type, amount, accountId, toAccountId, toAmount] = useWatch({
        control,
        name: ['type', 'amount', 'account_id', 'to_account_id', 'to_amount'],
    })

    return useMemo(() => {
        const next = toBalanceImpact({
            type,
            amount,
            account_id: accountId,
            to_account_id: toAccountId,
            to_amount: toAmount,
        })
        const source = from ? project(from, next, posted, 'source') : null
        const fromPreview = source && {
            ...source,
            insufficientFunds: (type === 'expense' || isTransferType(type)) && source.newBalance < 0,
        }
        const toPreview = to && isTransferType(type) ? project(to, next, posted, 'destination') : null
        const warnings = willPost
            ? collectNegativeBalanceWarnings([negativeWarning(from, fromPreview), negativeWarning(to, toPreview)])
            : []

        return { from: fromPreview, to: toPreview, warnings }
    }, [type, amount, accountId, toAccountId, toAmount, from, to, posted, willPost])
}
