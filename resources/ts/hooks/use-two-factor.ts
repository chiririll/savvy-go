import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { authApi } from '@/api'
import { toast } from 'sonner'
import i18n from '@/lib/i18n'
import { isConfirmCancelled, withConfirmation } from '@/stores/confirm'

const QUERY_KEY = ['two-factor-status']

export function useTwoFactorStatus() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: authApi.twoFactorStatus,
    })
}

export function useEnableTwoFactor() {
    return useMutation({
        mutationFn: () => withConfirmation(authApi.twoFactorEnable),
        onError: (error: Error) => {
            if (isConfirmCancelled(error)) return
            toast.error(error.message || i18n.t('toasts.twoFactor.enableFailed'))
        },
    })
}

export function useConfirmTwoFactor() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (code: string) => authApi.twoFactorConfirm(code),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.twoFactor.enabled'))
        },
        onError: (error: Error) => {
            toast.error(error.message || i18n.t('toasts.twoFactor.invalidCode'))
        },
    })
}

export function useDisableTwoFactor() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (code: string) => authApi.twoFactorDisable(code),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.twoFactor.disabled'))
        },
        onError: (error: Error) => {
            toast.error(error.message || i18n.t('toasts.twoFactor.invalidCode'))
        },
    })
}

export function useVerifyTwoFactor() {
    return useMutation({
        mutationFn: ({ token, code }: { token: string; code: string }) =>
            authApi.twoFactorVerify(token, code),
        onError: (error: Error) => {
            toast.error(error.message || i18n.t('toasts.twoFactor.invalidCode'))
        },
    })
}

export function useRecoveryCodes() {
    return useQuery({
        queryKey: [...QUERY_KEY, 'recovery-codes'],
        queryFn: authApi.twoFactorRecoveryCodes,
    })
}

export function useRegenerateRecoveryCodes() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (code: string) => authApi.twoFactorRegenerateRecoveryCodes(code),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.twoFactor.codesRegenerated'))
        },
        onError: (error: Error) => {
            toast.error(error.message || i18n.t('toasts.twoFactor.invalidCode'))
        },
    })
}
