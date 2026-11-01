import { useQuery } from '@tanstack/react-query'
import { budgetsApi } from '@/api'
import { BudgetFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['budgets']

export function useBudgets({ enabled = true }: { enabled?: boolean } = {}) {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: () => budgetsApi.getAll(),
        enabled,
    })
}

export function useBudget(id: string | number) {
    return useResourceItem(QUERY_KEY, () => budgetsApi.getById(id), id)
}

export function useCreateBudget(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: BudgetFormData) => budgetsApi.create(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.budget.created'),
        redirectTo,
    })
}

export function useUpdateBudget(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<BudgetFormData> }) =>
            budgetsApi.update(id, data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.budget.updated'),
        redirectTo,
    })
}

export function useDeleteBudget() {
    return useResourceMutation({
        mutationFn: (id: string | number) => budgetsApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.budget.deleted'),
    })
}
