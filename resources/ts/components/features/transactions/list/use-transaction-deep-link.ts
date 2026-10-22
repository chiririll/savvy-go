import { useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTransaction } from '@/hooks'
import type { Transaction } from '@/types'
import { useTransactionFormDialog } from '../form/CreateTransactionProvider'

export function useTransactionDeepLink(items?: Transaction[]) {
    const { openCreate, openEdit } = useTransactionFormDialog()
    const [searchParams, setSearchParams] = useSearchParams()
    const editId = searchParams.get('edit')
    const { data: editTransaction, isError: editNotFound } = useTransaction(editId ?? '')

    useEffect(() => {
        if (searchParams.get('create') !== '1') {
            return
        }

        const type = searchParams.get('type')
        const accountId = searchParams.get('account_id')
        const amount = searchParams.get('amount')
        const description = searchParams.get('description')

        openCreate({
            type: type === 'income' || type === 'expense' || type === 'transfer' ? type : undefined,
            account_id: accountId ? Number(accountId) : undefined,
            amount: amount ? Number(amount) : undefined,
            description: description ?? undefined,
        })

        setSearchParams((prev) => {
            prev.delete('create')
            prev.delete('account_id')
            prev.delete('amount')
            prev.delete('description')
            prev.delete('type')
            return prev
        }, { replace: true })
    }, [openCreate, searchParams, setSearchParams])

    useEffect(() => {
        if (!editId) {
            return
        }

        const found = items?.find((item) => String(item.id) === editId)
        const transaction = found ?? editTransaction
        if (!transaction && !editNotFound) {
            return
        }

        if (transaction) {
            openEdit(transaction)
        }

        setSearchParams((prev) => {
            prev.delete('edit')
            return prev
        }, { replace: true })
    }, [editId, editNotFound, editTransaction, items, openEdit, setSearchParams])

    return { openCreate, openEdit }
}
