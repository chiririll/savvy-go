import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation, useSearchParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, Fingerprint } from 'lucide-react'
import { SsoButtons } from '@/components/features/sso'
import { AuthCardShell } from '@/components/features/login/AuthCardShell'
import { PasswordLoginForm } from '@/components/features/login/PasswordLoginForm'
import { TwoFactorStep } from '@/components/features/login/TwoFactorStep'
import { usePasskeyLogin } from '@/components/features/login/usePasskeyLogin'
import { safeReturnPath, type LoginLocationState } from '@/components/features/login/return-path'
import { authApi } from '@/api'
import { toast } from 'sonner'

export default function LoginPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const location = useLocation()
    const [searchParams] = useSearchParams()
    const [checkingStatus, setCheckingStatus] = useState(true)
    const [passwordLoginEnabled, setPasswordLoginEnabled] = useState(true)
    const [twoFactor, setTwoFactor] = useState<{ token: string; rememberMe: boolean } | null>(null)
    const loginState = (location.state as LoginLocationState | null) ?? null
    const sessionExpiredNotice = Boolean(loginState?.sessionExpired)
    const afterLoginPath = safeReturnPath(loginState?.from)
    const passkey = usePasskeyLogin(afterLoginPath)

    useEffect(() => {
        authApi
            .status()
            .then((status) => {
                if (status.needs_registration) {
                    navigate('/setup', { replace: true })
                }
                setPasswordLoginEnabled(status.password_login_enabled)
            })
            .finally(() => setCheckingStatus(false))
    }, [navigate])

    // Resume the 2FA step when arriving from the SSO callback.
    useEffect(() => {
        const incoming = location.state as LoginLocationState | null
        if (incoming?.twoFactorToken) {
            setTwoFactor({ token: incoming.twoFactorToken, rememberMe: Boolean(incoming.rememberMe) })
            navigate(location.pathname, { replace: true, state: null })
        }
    }, [location.state, location.pathname, navigate])

    // Surface SSO errors bounced back to the login screen.
    useEffect(() => {
        if (searchParams.get('sso_error')) {
            toast.error(t('ssoFailed'))
        }
    }, [searchParams, t])

    if (checkingStatus) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
        )
    }

    if (twoFactor) {
        return (
            <TwoFactorStep
                token={twoFactor.token}
                rememberMe={twoFactor.rememberMe}
                afterLoginPath={afterLoginPath}
                onBack={() => setTwoFactor(null)}
            />
        )
    }

    return (
        <AuthCardShell title={t('welcomeBack')} description={t('signInDescription')} contentClassName="space-y-4">
            {sessionExpiredNotice && (
                <Alert>
                    <AlertDescription>{t('sessionExpired.banner')}</AlertDescription>
                </Alert>
            )}
            {passkey.supported && (
                <Button
                    variant="outline"
                    className="w-full"
                    onClick={() => passkey.signIn()}
                    disabled={passkey.loading}
                >
                    {passkey.loading ? (
                        <Loader2 className="mr-2 size-4 animate-spin" />
                    ) : (
                        <Fingerprint className="mr-2 size-4" />
                    )}
                    {t('signInPasskey')}
                </Button>
            )}
            <SsoButtons showDivider={passwordLoginEnabled} />
            {passwordLoginEnabled && (
                <PasswordLoginForm
                    afterLoginPath={afterLoginPath}
                    onTwoFactor={(token, rememberMe) => setTwoFactor({ token, rememberMe })}
                />
            )}
        </AuthCardShell>
    )
}
