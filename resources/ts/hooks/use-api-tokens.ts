import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { apiTokensApi } from '@/api'
import i18n from '@/lib/i18n'
import { getApiErrorMessage } from '@/lib/api-error'

const QUERY_KEY = ['api-tokens']

export function useApiTokens() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: apiTokensApi.list,
    })
}

export function useCreateApiToken() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: apiTokensApi.create,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.apiToken.created'))
        },
        onError: (error: unknown) => {
            toast.error(getApiErrorMessage(error, i18n.t('toasts.apiToken.createFailed')))
        },
    })
}

export function useRevokeApiToken() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (id: number) => apiTokensApi.revoke(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.apiToken.revoked'))
        },
        onError: (error: unknown) => {
            toast.error(getApiErrorMessage(error, i18n.t('toasts.apiToken.revokeFailed')))
        },
    })
}
