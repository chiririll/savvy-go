import { useQuery, useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import i18n from '@/lib/i18n/i18n'
import { ssoApi } from '@/api/sso'
import type { IdentityProviderFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'

const QUERY_KEY = ['identity-providers']
const PRESETS_KEY = ['sso-presets']
const PUBLIC_KEY = ['sso-providers']

export function useSsoProviders() {
    return useQuery({
        queryKey: PUBLIC_KEY,
        queryFn: () => ssoApi.providers(),
        staleTime: 60_000,
    })
}

export function useSsoPresets() {
    return useQuery({
        queryKey: PRESETS_KEY,
        queryFn: () => ssoApi.presets(),
        staleTime: Infinity,
    })
}

export function useIdentityProviders() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: () => ssoApi.list(),
    })
}

export function useIdentityProvider(id: string | number) {
    return useResourceItem(QUERY_KEY, () => ssoApi.getById(id), id)
}

export function useCreateIdentityProvider(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: IdentityProviderFormData) => ssoApi.create(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.sso.created'),
        redirectTo,
    })
}

export function useUpdateIdentityProvider(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<IdentityProviderFormData> }) =>
            ssoApi.update(id, data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.sso.updated'),
        redirectTo,
    })
}

export function useDeleteIdentityProvider() {
    return useResourceMutation({
        mutationFn: (id: string | number) => ssoApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.sso.deleted'),
    })
}

export function useTestIdentityProvider() {
    return useMutation({
        mutationFn: (id: string | number) => ssoApi.test(id),
        onSuccess: (result) => {
            if (result.status === 'ok') {
                toast.success(i18n.t('toasts.sso.testOk'))
                return
            }

            toast.error(result.message || i18n.t('toasts.sso.testFailed'))
        },
    })
}
