import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-react'
import { AuthCardShell } from '@/components/features/login/AuthCardShell'
import { SetPasswordForm, type SetPasswordValues } from '@/components/features/login/SetPasswordForm'
import { usePasswordTokenPreview } from '@/components/features/login/use-password-token-preview'
import { authApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { toast } from 'sonner'

export default function SetPasswordPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const { token } = useParams<{ token: string }>()
    const applySession = useAuthStore((state) => state.applySession)
    const { preview, loading, invalid } = usePasswordTokenPreview(token)
    const [submitting, setSubmitting] = useState(false)

    const onSubmit = async (data: SetPasswordValues) => {
        if (!token) return
        setSubmitting(true)
        try {
            const response = await authApi.acceptPasswordToken(token, data.password, data.password_confirmation)
            if ('requires_2fa' in response && response.requires_2fa) {
                toast.success(t('setPassword.updated'))
                navigate('/login', {
                    replace: true,
                    state: { twoFactorToken: response.two_factor_token, rememberMe: true },
                })
                return
            }
            if ('user' in response) {
                applySession(response)
            }
            toast.success(t('setPassword.updated'))
            navigate('/', { replace: true })
        } catch (error: unknown) {
            const message =
                error && typeof error === 'object' && 'message' in error
                    ? (error as { message: string }).message
                    : t('setPassword.failed')
            toast.error(message)
        } finally {
            setSubmitting(false)
        }
    }

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
        )
    }

    return (
        <AuthCardShell
            title={
                invalid
                    ? t('setPassword.invalidTitle')
                    : preview?.isInactive
                      ? t('setPassword.inviteTitle')
                      : t('setPassword.resetTitle')
            }
            description={
                invalid
                    ? t('setPassword.invalidDescription')
                    : t('setPassword.description', { email: preview?.email ?? '' })
            }
        >
            {!invalid && <SetPasswordForm submitting={submitting} onSubmit={onSubmit} />}
            {invalid && (
                <Button className="w-full" onClick={() => navigate('/login', { replace: true })}>
                    {t('setPassword.backToLogin')}
                </Button>
            )}
        </AuthCardShell>
    )
}
