import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { startRegistration } from '@simplewebauthn/browser'
import { webauthnApi } from '@/api'
import { toast } from 'sonner'
import i18n from '@/lib/i18n'
import { resolveUserFacingText } from '@/lib/api-error'
import { isConfirmCancelled, withConfirmation } from '@/stores/confirm'

const QUERY_KEY = ['webauthn-credentials']

export function isPasskeyDomainSupported(): boolean {
    if (typeof window === 'undefined' || !window.isSecureContext) return false

    const host = window.location.hostname
    if (host === 'localhost') return true

    const isIpv4 = /^\d{1,3}(\.\d{1,3}){3}$/.test(host)
    const isIpv6 = host.includes(':')
    return !isIpv4 && !isIpv6
}

export function useWebauthnCredentials() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: webauthnApi.listCredentials,
    })
}

export function useRegisterPasskey() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: async (name: string | null) => {
            const { token, options } = await withConfirmation(webauthnApi.registerOptions)
            const attestation = await startRegistration({ optionsJSON: options })
            return webauthnApi.registerVerify(token, attestation, name)
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.passkey.added'))
        },
        onError: (error: unknown) => {
            if (isConfirmCancelled(error)) return
            toast.error(passkeyErrorMessage(error, i18n.t('toasts.passkey.addFailed')))
        },
    })
}

export function useRenamePasskey() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ id, name }: { id: number; name: string }) =>
            webauthnApi.renameCredential(id, name),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.passkey.renamed'))
        },
        onError: (error: Error) => {
            toast.error(error.message || i18n.t('toasts.passkey.renameFailed'))
        },
    })
}

export function useDeletePasskey() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (id: number) => withConfirmation((fields) => webauthnApi.deleteCredential(id, fields)),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: QUERY_KEY })
            toast.success(i18n.t('toasts.passkey.removed'))
        },
        onError: (error: Error) => {
            if (isConfirmCancelled(error)) return
            toast.error(error.message || i18n.t('toasts.passkey.removeFailed'))
        },
    })
}

export function passkeyErrorMessage(error: unknown, fallback: string): string {
    if (error instanceof Error) {
        if (error.name === 'NotAllowedError' || error.name === 'AbortError') {
            return i18n.t('toasts.passkey.cancelled')
        }
        if (error.name === 'InvalidStateError') {
            return i18n.t('toasts.passkey.alreadyRegistered')
        }
        return resolveUserFacingText(error.message || fallback)
    }
    if (error && typeof error === 'object' && 'message' in error) {
        return resolveUserFacingText((error as { message: string }).message)
    }
    return fallback
}
