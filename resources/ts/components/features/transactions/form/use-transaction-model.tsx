import { useEffect, useLayoutEffect, useMemo, type MutableRefObject, type ReactNode } from 'react'
import { useWatch, type UseFormReturn } from 'react-hook-form'
import { useAccounts, useCategories, useCurrencies, useTransactionPartyDefaults } from '@/hooks'
import { isDateInFuture } from '@/lib/utils'
import { isSplitPayment } from '@/lib/payment-input'
import { currencyDecimals, sumTransactionItems } from '@/lib/transaction-items'
import { isTransferType, TransactionFormValues } from '@/schemas/transactions'
import { toBalanceImpact, useBalancePreview } from './balance-preview'
import { TransactionBalanceHint } from './TransactionBalanceHint'
import { useTransferRate } from './transfer-rate'

export const ACCOUNT_FILTER = { active: true, exclude_debts: true }

interface TransactionModelOptions {
    form: UseFormReturn<TransactionFormValues>
    schemaOptionsRef: MutableRefObject<{ rejectFutureDate: boolean; currencyDecimals: number }>
    defaultValues?: Partial<TransactionFormValues>
    isEdit?: boolean
    originalAffectsBalance?: boolean
    onPreviewChange?: (preview: ReactNode) => void
    open: boolean
}

// What the form derives from the accounts it picked: types, rates, balance preview.
export function useTransactionModel({
    form,
    schemaOptionsRef,
    defaultValues,
    isEdit,
    originalAffectsBalance,
    onPreviewChange,
    open,
}: TransactionModelOptions) {
    const { data: accounts } = useAccounts(ACCOUNT_FILTER)
    const { data: categories } = useCategories()
    const { data: currencies } = useCurrencies()

    const [transactionType, accountId, toAccountId, toSpaceId, items, date, extraPayments, changes] = useWatch({
        control: form.control,
        name: ['type', 'account_id', 'to_account_id', 'to_space_id', 'items', 'date', 'extra_payments', 'changes'],
    })
    // A split payment is several transactions: one account's balance hint would be wrong for it.
    const isSplit = transactionType === 'expense' && isSplitPayment({ extra_payments: extraPayments, changes })
    const isTransfer = isTransferType(transactionType)
    const isSpaceTransfer = transactionType === 'transfer_out'
    const hasItems = Boolean(items?.length)
    // The transfers API that stores a transfer to another space needs a date.
    const dateRequired = Boolean(originalAffectsBalance) || isSpaceTransfer
    const isPendingDate = !date || isDateInFuture(date)

    // Where the destination account lives: undefined is this space.
    const destinationSpaceId = isSpaceTransfer ? Number(toSpaceId) || null : undefined
    const { data: destinationSpaceAccounts } = useAccounts(ACCOUNT_FILTER, destinationSpaceId ?? null)

    useTransactionPartyDefaults(form, accounts, categories)

    useEffect(() => {
        if (isTransfer) {
            form.setValue('category_id', null)
        }
    }, [isTransfer, form])

    const selectedAccount = accounts?.find((a) => a.id === Number(accountId))
    const selectedToAccount = (isSpaceTransfer ? destinationSpaceAccounts : accounts)?.find(
        (a) => a.id === Number(toAccountId),
    )
    const itemDecimals = currencyDecimals(selectedAccount?.currency)
    schemaOptionsRef.current = {
        rejectFutureDate: Boolean(originalAffectsBalance),
        currencyDecimals: itemDecimals,
    }

    const itemsTotal = sumTransactionItems(items, itemDecimals)
    useEffect(() => {
        if (hasItems && itemsTotal > 0) {
            form.setValue('amount', itemsTotal, { shouldValidate: false })
        }
    }, [itemsTotal, hasItems, form])

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
        () => (isEdit && originalAffectsBalance && defaultValues ? toBalanceImpact(defaultValues) : null),
        [isEdit, originalAffectsBalance, defaultValues],
    )
    const balances = useBalancePreview({
        control: form.control,
        from: selectedAccount,
        to: selectedToAccount,
        posted,
        willPost: Boolean(originalAffectsBalance) || (!isEdit && !isPendingDate),
    })

    const balanceHint = (
        <TransactionBalanceHint from={balances.from} to={balances.to} isPending={isPendingDate} isUndated={!date} />
    )

    useLayoutEffect(() => {
        onPreviewChange?.(isSplit ? null : balanceHint)
    }, [onPreviewChange, balances, isPendingDate, date, isSplit])

    return {
        accounts,
        transactionType,
        accountId,
        isTransfer,
        isSpaceTransfer,
        isSplit,
        hasItems,
        dateRequired,
        isPendingDate,
        destinationSpaceId,
        destinationSpaceAccounts,
        selectedAccount,
        selectedToAccount,
        rate,
        balances,
        balanceHint,
    }
}
