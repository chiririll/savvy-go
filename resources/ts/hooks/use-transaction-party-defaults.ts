import { useEffect } from 'react'
import { isTransferType } from '@/schemas/transactions'
import type { Account, Category } from '@/types'

interface TransactionPartyForm {
    watch: (name: 'type' | 'account_id' | 'category_id') => unknown
    setValue: (name: 'account_id' | 'category_id', value: number | null) => void
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

        const popular = [...(categories ?? [])]
            .filter((category) => category.type === type)
            .sort((a, b) => (b.transactionsCount ?? 0) - (a.transactionsCount ?? 0))

        if (popular[0]) {
            form.setValue('category_id', popular[0].id)
        }
    }, [categoryId, type, categories, form])
}
