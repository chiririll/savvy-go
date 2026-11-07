import { useEffect } from 'react'
import { isTransferType } from '@/lib/transaction-type'
import type { Account, Category } from '@/types'

interface TransactionPartyForm {
    watch: (name: 'type' | 'account_id' | 'category_id') => unknown
    setValue: (name: 'account_id' | 'category_id', value: number | null) => void
}

/** The category a new income or expense starts with: the default one, else the one used most. */
export function pickDefaultCategory(categories: Category[] | undefined, type: string): Category | undefined {
    const ofType = (categories ?? []).filter((category) => category.type === type)
    return ofType.find((category) => category.isDefault)
        ?? [...ofType].sort((a, b) => (b.transactionsCount ?? 0) - (a.transactionsCount ?? 0))[0]
}

export function useTransactionPartyDefaults(
    form: TransactionPartyForm,
    accounts: Account[] | undefined,
    categories: Category[] | undefined,
) {
    const type = form.watch('type') as string
    const accountId = form.watch('account_id') as number | null | undefined
    const categoryId = form.watch('category_id') as number | null | undefined

    useEffect(() => {
        if (accountId === null) {
            return
        }
        if (!accountId && accounts && accounts.length > 0) {
            form.setValue('account_id', accounts[0].id)
        }
    }, [accountId, accounts, form])

    useEffect(() => {
        if (categoryId || isTransferType(type)) {
            return
        }

        const preselected = pickDefaultCategory(categories, type)

        if (preselected) {
            form.setValue('category_id', preselected.id)
        }
    }, [categoryId, type, categories, form])
}
