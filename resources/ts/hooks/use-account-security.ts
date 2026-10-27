import { authApi } from '@/api'
import { useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'
import { useAuthStore } from '@/stores/auth'

export function useChangePassword() {
    return useResourceMutation({
        mutationFn: authApi.changePassword,
        // An SSO-only account stops being one once it has a password.
        onSuccess: () => void useAuthStore.getState().checkAuth(),
        successMessage: i18n.t('toasts.accountSecurity.passwordChanged'),
    })
}

export function useLogoutOthers() {
    return useResourceMutation<Awaited<ReturnType<typeof authApi.logoutOthers>>, void>({
        mutationFn: authApi.logoutOthers,
        successMessage: i18n.t('toasts.accountSecurity.loggedOutOthers'),
    })
}
