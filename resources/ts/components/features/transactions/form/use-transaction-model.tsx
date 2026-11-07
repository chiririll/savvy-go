import { useEffect, useMemo } from 'react'
import { useWatch, type UseFormReturn } from 'react-hook-form'
import { useAccounts, useCategories } from '@/hooks'
import { pickDefaultCategory } from '@/hooks/use-transaction-party-defaults'
import { isDateInFuture } from '@/lib/utils'
import { lineCoverage, paymentCurrencies } from '@/lib/transaction-lines'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Transaction } from '@/types'
import { postedImpact, useBalancePreview } from './balance-preview'

export const ACCOUNT_FILTER = { active: true, exclude_debts: true }

interface TransactionModelOptions {
    form: UseFormReturn<TransactionFormValues>
    /** The transaction being edited, when the form is an edit. */
    transaction?: Transaction | null
    isEdit?: boolean
    originalAffectsBalance?: boolean
}

// What the form derives from what it holds: the accounts it uses, their balances, whether the date makes it pending.
export function useTransactionModel({ form, transaction, isEdit, originalAffectsBalance }: TransactionModelOptions) {
    const { data: accounts } = useAccounts(ACCOUNT_FILTER)
    const { data: categories } = useCategories()

    const [transactionType, payments, change, date, lines, defaultCategoryId] = useWatch({
        control: form.control,
        name: ['type', 'payments', 'change', 'date', 'lines', 'default_category_id'],
    })
    // A transaction that already moved balances keeps a date.
    const dateRequired = Boolean(originalAffectsBalance)
    const isPendingDate = !date || isDateInFuture(date)

    // A new transaction starts with the first account and the category of its type.
    const firstAccountId = Number(payments?.[0]?.account_id)
    useEffect(() => {
        if (!(firstAccountId > 0) && accounts && accounts.length > 0) {
            form.setValue('payments.0.account_id', accounts[0].id)
        }
    }, [firstAccountId, accounts, form])

    useEffect(() => {
        if (defaultCategoryId || lines?.length) {
            return
        }
        const preselected = pickDefaultCategory(categories, transactionType)
        if (preselected) {
            form.setValue('default_category_id', preselected.id)
        }
    }, [defaultCategoryId, lines, transactionType, categories, form])

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

    // How far the lines cover the payments: the default category takes the rest.
    const values = { payments: payments ?? [], change: change ?? [], lines: lines ?? [] }
    const coverage = lineCoverage(values, accounts ?? [])
    const lineCurrencies = paymentCurrencies(values, accounts ?? [])
    const hasLines = Boolean(lines?.length)

    return {
        accounts,
        coverage,
        lineCurrencies,
        hasLines,
        transactionType,
        dateRequired,
        isPendingDate,
        isUndated: !date,
        balances,
        usedAccountIds,
    }
}
