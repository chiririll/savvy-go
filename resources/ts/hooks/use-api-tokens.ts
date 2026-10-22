import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { apiTokensApi } from '@/api'
import type { CreateApiTokenInput, UpdateApiTokenInput } from '@/types'
import i18n from '@/lib/i18n'
import { getApiErrorMessage } from '@/lib/api-error'
import { isConfirmCancelled, withConfirmation } from '@/stores/confirm'

const QUERY_KEY = ['api-tokens']

export function useApiTokens() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: apiTokensApi.list,
    })
}

/** What each kind of token can be given. Fixed for the life of the server. */
export function useApiTokenCapabilities() {
    return useQuery({
        queryKey: ['api-token-capabilities'],
        queryFn: apiTokensApi.capabilities,
        staleTime: Infinity,
    })
}

export function useCreateApiToken() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (input: CreateApiTokenInput) => withConfirmation((fields) => apiTokensApi.create(input, fields)),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.apiToken.created'))
        },
        onError: (error: unknown) => {
            if (isConfirmCancelled(error)) return
            toast.error(getApiErrorMessage(error, i18n.t('toasts.apiToken.createFailed')))
        },
    })
}

export function useUpdateApiToken() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ id, input }: { id: number; input: UpdateApiTokenInput }) =>
            withConfirmation((fields) => apiTokensApi.update(id, input, fields)),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.apiToken.updated'))
        },
        onError: (error: unknown) => {
            if (isConfirmCancelled(error)) return
            toast.error(getApiErrorMessage(error, i18n.t('toasts.apiToken.updateFailed')))
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
