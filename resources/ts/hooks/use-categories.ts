import { useQuery } from '@tanstack/react-query'
import { categoriesApi } from '@/api'
import { CategoryType } from '@/types'
import { CategoryFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['categories']

export function useCategories(type?: CategoryType) {
    return useQuery({
        queryKey: type ? [...QUERY_KEY, type] : QUERY_KEY,
        queryFn: () => categoriesApi.getAll(type ? { type } : undefined),
    })
}

export function useCategorySummary(params: {
    type: 'income' | 'expense'
    start_date?: string
    end_date?: string
}) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'summary', params],
        queryFn: () => categoriesApi.getSummary(params),
        enabled: !!params.type,
    })
}

export function useCategory(id: string | number) {
    return useResourceItem(QUERY_KEY, () => categoriesApi.getById(id), id)
}

export function useCreateCategory(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: CategoryFormData) => categoriesApi.create(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.category.created'),
        redirectTo,
    })
}

export function useUpdateCategory(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<CategoryFormData> }) =>
            categoriesApi.update(id, data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.category.updated'),
        redirectTo,
    })
}

export function useDeleteCategory() {
    return useResourceMutation({
        mutationFn: ({ id, successorId }: { id: string | number; successorId?: number }) =>
            categoriesApi.delete(id, successorId),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.category.deleted'),
    })
}

export function useSetDefaultCategory() {
    return useResourceMutation({
        mutationFn: (id: string | number) => categoriesApi.setDefault(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.category.defaultUpdated'),
    })
}
