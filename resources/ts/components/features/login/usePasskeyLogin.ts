import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { browserSupportsWebAuthn } from '@simplewebauthn/browser'
import { passkeyErrorMessage, isPasskeyDomainSupported } from '@/hooks'
import { useAuthStore } from '@/stores/auth'

/** Signing in with a passkey, alone or as the second step after a password or SSO. */
export function usePasskeyLogin(afterLoginPath: string) {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const loginWithPasskey = useAuthStore((state) => state.loginWithPasskey)
    const [loading, setLoading] = useState(false)
    const [supported] = useState(() => browserSupportsWebAuthn() && isPasskeyDomainSupported())

    const signIn = async (stepUpToken?: string) => {
        setLoading(true)
        try {
            await loginWithPasskey({ twoFactorToken: stepUpToken })
            toast.success(t('welcomeToast'))
            navigate(afterLoginPath)
        } catch (error: unknown) {
            toast.error(passkeyErrorMessage(error, t('passkeyFailed')))
        } finally {
            setLoading(false)
        }
    }

    return { supported, loading, signIn }
}
