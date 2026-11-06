import { useMemo } from 'react'
import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData, type QueryKey } from '@tanstack/react-query'
import { toast } from 'sonner'
import { transactionsApi, transactionGroupsApi, type ConfirmTransactionOptions, type TransactionsResponse } from '@/api'
import type { TransactionsResponse } from '@/api/transactions'
import { Transaction, TransactionFilters } from '@/types'
import { TransactionFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'
import { getApiErrorMessage } from '@/lib/api-error'
import { dayOrderForApi, placeInSlots } from '@/lib/transaction-order'
import type { PaymentLeg } from '@/lib/split-payment'

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
    const queryClient = useQueryClient()

    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<TransactionFormData> }) =>
            transactionsApi.update(id, data),
        // No reset: dropping the loaded pages collapses the list and loses scroll position.
        // The saved row is put into the loaded pages at once; invalidation then refetches
        // them in place, which also re-sorts a row whose date or amount changed.
        onSuccess: (saved) => {
            queryClient.setQueriesData<InfiniteData<TransactionsResponse>>(
                { queryKey: [...QUERY_KEY, 'infinite'] },
                (cached) =>
                    cached && {
                        ...cached,
                        pages: cached.pages.map((page) => ({
                            ...page,
                            data: page.data.map((item) => (item.id === saved.id ? saved : item)),
                        })),
                    },
            )
        },
        invalidateKeys: [...TRANSACTION_INVALIDATE],
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

interface ReorderVariables {
    date: string
    /** The day's transactions as the list shows them. */
    shownIds: number[]
    direction: 'asc' | 'desc'
}

/** Drags within one day: the lists show the new order at once and the server confirms it. */
export function useReorderTransactions() {
    const queryClient = useQueryClient()
    const listKey = [...QUERY_KEY, 'infinite']

    return useMutation({
        mutationFn: ({ date, shownIds, direction }: ReorderVariables) =>
            transactionsApi.reorder(date, dayOrderForApi(shownIds, direction)),
        onMutate: async ({ shownIds }) => {
            await queryClient.cancelQueries({ queryKey: listKey })
            const previous = queryClient.getQueriesData({ queryKey: listKey })

            queryClient.setQueriesData<InfiniteData<TransactionsResponse>>({ queryKey: listKey }, (old) => {
                if (!old) {
                    return old
                }

                const flat = old.pages.flatMap((page) => page.data)
                const placed = placeInSlots(flat, shownIds)
                let offset = 0

                return {
                    ...old,
                    pages: old.pages.map((page) => {
                        const data = placed.slice(offset, offset + page.data.length)
                        offset += page.data.length
                        return { ...page, data }
                    }),
                }
            })

            return { previous }
        },
        onError: (error, _variables, context) => {
            context?.previous.forEach(([key, data]: [QueryKey, unknown]) => {
                queryClient.setQueryData(key, data)
            })
            toast.error(getApiErrorMessage(error, i18n.t('toasts.transaction.reorderFailed')))
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
        },
    })
}

export function useTransactionGroup(uuid: string | null) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'group', uuid],
        queryFn: () => transactionGroupsApi.getByUuid(uuid as string),
        enabled: uuid !== null,
    })
}

export function useCreateTransactionGroup() {
    return useResourceMutation({
        mutationFn: (legs: PaymentLeg[]) => transactionGroupsApi.create(legs),
        invalidateKeys: [...TRANSACTION_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: i18n.t('toasts.transaction.paymentCreated'),
    })
}

export function useUpdateTransactionGroup() {
    return useResourceMutation({
        mutationFn: ({ uuid, legs }: { uuid: string; legs: PaymentLeg[] }) => transactionGroupsApi.update(uuid, legs),
        invalidateKeys: [...TRANSACTION_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: i18n.t('toasts.transaction.paymentUpdated'),
    })
}

export function useDeleteTransactionGroup() {
    return useResourceMutation({
        mutationFn: (uuid: string) => transactionGroupsApi.delete(uuid),
        invalidateKeys: [...TRANSACTION_INVALIDATE],
        resetKeys: [...TRANSACTION_INFINITE_RESET],
        successMessage: i18n.t('toasts.transaction.paymentDeleted'),
    })
}
