import { useEffect, useMemo } from 'react'
import { useWatch, type UseFormReturn } from 'react-hook-form'
import { useAccounts, useCategories, useCurrencies } from '@/hooks'
import { pickDefaultCategory } from '@/hooks/use-transaction-party-defaults'
import { isDateInFuture } from '@/lib/utils'
import { isTransferType } from '@/lib/transaction-type'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Transaction } from '@/types'
import { postedImpact, useBalancePreview } from './balance-preview'
import { useLineHints } from './use-line-hints'
import { useTransferRate } from './transfer-rate'

export const ACCOUNT_FILTER = { active: true, exclude_debts: true }

interface TransactionModelOptions {
    form: UseFormReturn<TransactionFormValues>
    /** The transaction being edited, when the form is an edit. */
    transaction?: Transaction | null
    isEdit?: boolean
    originalAffectsBalance?: boolean
    /** The form (re)opens: a transfer starts its exchange rate over. */
    open: boolean
}

// What the form derives from what it holds: the accounts it uses, their balances, whether the date makes it pending.
export function useTransactionModel({ form, transaction, isEdit, originalAffectsBalance, open }: TransactionModelOptions) {
    const { data: accounts } = useAccounts(ACCOUNT_FILTER)
    const { data: categories } = useCategories()
    const { data: currencies } = useCurrencies()

    const [transactionType, payments, change, date, lines, defaultCategoryId, toAccountId, toSpaceId] = useWatch({
        control: form.control,
        name: ['type', 'payments', 'change', 'date', 'lines', 'default_category_id', 'to_account_id', 'to_space_id'],
    })
    const isTransfer = isTransferType(transactionType)
    const isSpaceTransfer = transactionType === 'transfer_out'
    // A transaction that already moved balances keeps a date, and so does a transfer to another space.
    const dateRequired = Boolean(originalAffectsBalance) || isSpaceTransfer
    const isPendingDate = !date || isDateInFuture(date)

    // A new transaction starts with the first account and the category of its type.
    const firstAccountId = Number(payments?.[0]?.account_id)
    useEffect(() => {
        if (!(firstAccountId > 0) && accounts && accounts.length > 0) {
            form.setValue('payments.0.account_id', accounts[0].id)
        }
    }, [firstAccountId, accounts, form])

    useEffect(() => {
        if (isTransfer || defaultCategoryId || lines?.length) {
            return
        }
        const preselected = pickDefaultCategory(categories, transactionType)
        if (preselected) {
            form.setValue('default_category_id', preselected.id)
        }
    }, [isTransfer, defaultCategoryId, lines, transactionType, categories, form])

    // Where the destination lives: undefined is this space, null is another one that is not chosen yet.
    const destinationSpaceId = isSpaceTransfer ? Number(toSpaceId) || null : undefined
    const { data: destinationSpaceAccounts } = useAccounts(ACCOUNT_FILTER, destinationSpaceId ?? null)
    const selectedAccount = accounts?.find((a) => a.id === Number(payments?.[0]?.account_id))
    const selectedToAccount = (isSpaceTransfer ? destinationSpaceAccounts : accounts)?.find(
        (a) => a.id === Number(toAccountId),
    )
    const rate = useTransferRate({
        form,
        from: selectedAccount,
        to: selectedToAccount,
        // Rates are relative to this space's base currency, so look the destination up by code.
        toRate: currencies?.find((c) => c.code === selectedToAccount?.currency?.code)?.rate,
        open,
        ratesReady: currencies !== undefined,
    })

    const posted = useMemo(
        () => (isEdit && originalAffectsBalance && transaction ? postedImpact(transaction) : null),
        [isEdit, originalAffectsBalance, transaction],
    )
    const balances = useBalancePreview({
        control: form.control,
        accounts,
        posted,
        willPost: Boolean(originalAffectsBalance) || (!isEdit && !isPendingDate),
    })

    const usedAccountIds = [...(payments ?? []), ...(change ?? [])].map((row) => Number(row?.account_id)).filter((id) => id > 0)

    const lineHints = useLineHints({ form, accounts, payments, change, lines, defaultCategoryId })

    return {
        accounts,
        ...lineHints,
        transactionType,
        isTransfer,
        isSpaceTransfer,
        destinationSpaceId,
        destinationSpaceAccounts,
        selectedAccount,
        selectedToAccount,
        rate,
        dateRequired,
        isPendingDate,
        isUndated: !date,
        balances,
        usedAccountIds,
    }
}
