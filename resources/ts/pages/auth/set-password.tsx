import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Loader2 } from 'lucide-react'
import { LanguageSwitcher } from '@/components/shared'
import { Logo } from '@/components/shared/Logo'
import { SetPasswordForm, type SetPasswordValues } from '@/components/features/login/SetPasswordForm'
import { usePasswordTokenPreview } from '@/components/features/login/use-password-token-preview'
import { authApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

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
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <div className="flex justify-center mb-4">
                        <div
                            className={cn(
                                'flex size-12 items-center justify-center rounded-xl bg-primary',
                                'text-primary-foreground',
                            )}
                        >
                            <Logo className="size-7" />
                        </div>
                    </div>
                    <CardTitle className="text-2xl">
                        {invalid
                            ? t('setPassword.invalidTitle')
                            : preview?.isInactive
                              ? t('setPassword.inviteTitle')
                              : t('setPassword.resetTitle')}
                    </CardTitle>
                    <CardDescription>
                        {invalid
                            ? t('setPassword.invalidDescription')
                            : t('setPassword.description', { email: preview?.email ?? '' })}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    {!invalid && <SetPasswordForm submitting={submitting} onSubmit={onSubmit} />}
                    {invalid && (
                        <Button className="w-full" onClick={() => navigate('/login', { replace: true })}>
                            {t('setPassword.backToLogin')}
                        </Button>
                    )}
                </CardContent>
            </Card>
            <LanguageSwitcher variant="auth" />
        </div>
    )
}
