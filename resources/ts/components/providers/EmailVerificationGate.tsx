import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, MailCheck } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { authApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { ApiError } from '@/types'
import { cn } from '@/lib/utils'

/**
 * Closes the app to a user whose address is not confirmed while the instance
 * requires it. They may ask for a new link, check again once they used it,
 * or sign out; the server refuses everything else.
 */
export function EmailVerificationGate({ children }: { children: ReactNode }) {
    const required = useAuthStore((state) => state.emailVerificationRequired)
    const user = useAuthStore((state) => state.user)

    if (!required || !user || user.emailVerified !== false) {
        return <>{children}</>
    }
    return <UnverifiedEmailScreen email={user.email} />
}

function UnverifiedEmailScreen({ email }: { email: string }) {
    const { t } = useTranslation('auth')
    const checkAuth = useAuthStore((state) => state.checkAuth)
    const logout = useAuthStore((state) => state.logout)
    const [sending, setSending] = useState(false)
    const [checking, setChecking] = useState(false)

    const resend = async () => {
        setSending(true)
        try {
            await authApi.resendVerification()
            toast.success(t('verifyEmail.sent', { email }))
        } catch (error) {
            toast.error((error as ApiError).message)
        } finally {
            setSending(false)
        }
    }

    const check = async () => {
        setChecking(true)
        try {
            await checkAuth()
        } finally {
            setChecking(false)
        }
    }

    return (
        <div className="flex min-h-screen items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <div className="mb-2 flex justify-center">
                        <div
                            className={cn(
                                'flex size-12 items-center justify-center rounded-xl bg-primary',
                                'text-primary-foreground',
                            )}
                        >
                            <MailCheck className="size-6" />
                        </div>
                    </div>
                    <CardTitle className="text-2xl">{t('verifyEmail.requiredTitle')}</CardTitle>
                    <CardDescription>{t('verifyEmail.requiredDescription', { email })}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-2">
                    <Button className="w-full" onClick={check} disabled={checking}>
                        {checking && <Loader2 className="mr-2 size-4 animate-spin" />}
                        {t('verifyEmail.confirmed')}
                    </Button>
                    <Button variant="outline" className="w-full" onClick={resend} disabled={sending}>
                        {sending && <Loader2 className="mr-2 size-4 animate-spin" />}
                        {t('verifyEmail.resend')}
                    </Button>
                    <Button variant="ghost" className="w-full" onClick={() => void logout()}>
                        {t('verifyEmail.signOut')}
                    </Button>
                </CardContent>
            </Card>
        </div>
    )
}
