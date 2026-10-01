import { useQuery, useMutation, useQueryClient, QueryKey } from '@tanstack/react-query'
import { accountsApi } from '@/api'
import { Account } from '@/types'
import { AccountFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['accounts']

export function useAccounts(params?: { active?: boolean; exclude_debts?: boolean }) {
    return useQuery({
        queryKey: params ? [...QUERY_KEY, params] : QUERY_KEY,
        queryFn: () => accountsApi.getAll(params),
    })
}

export function useTotalBalance() {
    return useQuery({
        queryKey: [...QUERY_KEY, 'summary'],
        queryFn: () => accountsApi.getAllWithSummary({ active: true }),
        select: (data) => data.summary,
    })
}

export function useBalanceHistory(params?: { start_date?: string; end_date?: string }) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'balance-history', params],
        queryFn: () => accountsApi.getBalanceHistory(params),
    })
}

export function useBalanceComparison(params?: { start_date?: string; end_date?: string }) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'balance-comparison', params],
        queryFn: () => accountsApi.getBalanceComparison(params),
    })
}

export function useAccount(id: string | number) {
    return useResourceItem(QUERY_KEY, () => accountsApi.getById(id), id)
}

export function useCreateAccount(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: AccountFormData) => accountsApi.create(data),
        invalidateKeys: [QUERY_KEY, ['currencies']],
        successMessage: i18n.t('toasts.account.created'),
        redirectTo,
    })
}

export function useUpdateAccount(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<AccountFormData> }) =>
            accountsApi.update(id, data),
        invalidateKeys: [QUERY_KEY, ['currencies']],
        successMessage: i18n.t('toasts.account.updated'),
        redirectTo,
    })
}

export function useReorderAccounts() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (ids: number[]) => accountsApi.reorder(ids),
        onMutate: async (ids) => {
            await queryClient.cancelQueries({ queryKey: QUERY_KEY })
            const previous = queryClient.getQueriesData({ queryKey: QUERY_KEY })

            queryClient.setQueriesData<Account[]>({ queryKey: QUERY_KEY }, (old) => {
                if (!Array.isArray(old)) {
                    return old
                }

                const byId = new Map(old.map((account) => [account.id, account]))
                const reordered = ids
                    .map((id) => byId.get(id))
                    .filter((account): account is Account => account !== undefined)
                const rest = old.filter((account) => !ids.includes(account.id))

                return [...reordered, ...rest]
            })

            return { previous }
        },
        onError: (_error, _ids, context) => {
            context?.previous.forEach(([key, data]: [QueryKey, unknown]) => {
                queryClient.setQueryData(key, data)
            })
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
        },
    })
}

export function useDeleteAccount() {
    return useResourceMutation({
        mutationFn: (id: string | number) => accountsApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.account.deleted'),
    })
}
