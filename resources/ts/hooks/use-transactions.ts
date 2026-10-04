import { useMemo } from 'react'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { transactionsApi, type ConfirmTransactionOptions } from '@/api'
import { Transaction, TransactionFilters } from '@/types'
import { TransactionFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['transactions']
const TRANSACTION_INVALIDATE = [
    QUERY_KEY,
    ['accounts'],
    ['budgets'],
    ['categories'],
    ['reports'],
] as const
const TRANSACTION_CONFIRM_INVALIDATE = [
    ...TRANSACTION_INVALIDATE,
    ['recurring'],
] as const
const TRANSACTION_INFINITE_RESET = [[...QUERY_KEY, 'infinite']] as const

type TransactionQueryFilters = TransactionFilters & { with_summary?: boolean }

function withoutPage(filters?: TransactionQueryFilters) {
    if (!filters) {
        return undefined
    }

    const { page: _page, ...rest } = filters
    return rest
}

export function useTransactions(filters?: TransactionQueryFilters) {
    return useQuery({
        queryKey: filters ? [...QUERY_KEY, filters] : QUERY_KEY,
        queryFn: () => transactionsApi.getAll(filters),
    })
}

export function useInfiniteTransactions(filters?: TransactionQueryFilters) {
    const listFilters = withoutPage(filters)
    const query = useInfiniteQuery({
        queryKey: [...QUERY_KEY, 'infinite', listFilters],
        queryFn: ({ pageParam }) => transactionsApi.getAll({ ...listFilters, page: pageParam }),
        initialPageParam: 1,
        getNextPageParam: (lastPage) => {
            const meta = lastPage.meta
            if (!meta || meta.current_page >= meta.last_page) {
                return undefined
            }

            return meta.current_page + 1
        },
    })

    const transactions = useMemo(
        () => query.data?.pages.flatMap((page) => page.data) ?? [],
        [query.data],
    )

    return {
        ...query,
        transactions,
        meta: query.data?.pages.at(-1)?.meta,
    }
}

export function useTransaction(id: string | number) {
    return useResourceItem(QUERY_KEY, () => transactionsApi.getById(id), id)
}

export function useCreateTransaction(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: TransactionFormData) => transactionsApi.create(data),
        invalidateKeys: [...TRANSACTION_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: (transaction: Transaction) =>
            transaction.status === 'pending'
                ? i18n.t('toasts.transaction.pendingCreated')
                : i18n.t('toasts.transaction.created'),
        redirectTo,
    })
}

export function useUpdateTransaction(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<TransactionFormData> }) =>
            transactionsApi.update(id, data),
        invalidateKeys: [...TRANSACTION_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: i18n.t('toasts.transaction.updated'),
        redirectTo,
    })
}

export function useDeleteTransaction() {
    return useResourceMutation({
        mutationFn: (id: string | number) => transactionsApi.delete(id),
        invalidateKeys: [...TRANSACTION_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: i18n.t('toasts.transaction.deleted'),
    })
}

export function useDuplicateTransaction() {
    return useResourceMutation({
        mutationFn: (id: string | number) => transactionsApi.duplicate(id),
        invalidateKeys: [...TRANSACTION_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: i18n.t('toasts.transaction.duplicated'),
    })
}

export function useConfirmTransaction() {
    return useResourceMutation({
        mutationFn: ({ id, ...options }: { id: string | number } & ConfirmTransactionOptions) =>
            transactionsApi.confirm(id, options),
        invalidateKeys: [...TRANSACTION_CONFIRM_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: i18n.t('toasts.transaction.confirmed'),
    })
}

export function usePendingSummary() {
    return useQuery({
        queryKey: [...QUERY_KEY, 'pending-summary'],
        queryFn: () => transactionsApi.getPendingSummary(),
    })
}

export function useSkipTransaction() {
    return useResourceMutation({
        mutationFn: (id: string | number) => transactionsApi.skip(id),
        invalidateKeys: [...TRANSACTION_CONFIRM_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: i18n.t('toasts.transaction.skipped'),
    })
}

export function useTransactionSummary(filters?: TransactionFilters) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'summary', filters],
        queryFn: () => transactionsApi.getSummary(filters),
    })
}
