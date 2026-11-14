import { useQuery } from '@tanstack/react-query'
import { tagsApi } from '@/api'
import { TagFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n/i18n'

const QUERY_KEY = ['tags']

export function useTags() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: () => tagsApi.getAll(),
    })
}

export function useTag(id: string | number) {
    return useResourceItem(QUERY_KEY, () => tagsApi.getById(id), id)
}

export function useCreateTag(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: TagFormData) => tagsApi.create(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.tag.created'),
        redirectTo,
    })
}

export function useUpdateTag(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<TagFormData> }) =>
            tagsApi.update(id, data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.tag.updated'),
        redirectTo,
    })
}

export function useDeleteTag() {
    return useResourceMutation({
        mutationFn: (id: string | number) => tagsApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.tag.deleted'),
    })
}
