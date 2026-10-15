import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Loader2, MailCheck, MailX } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { authApi } from '@/api'
import { useAuthStore } from '@/stores/auth'

type State = 'working' | 'done' | 'invalid'

/** Opened from the link in a confirmation email; works with or without a session. */
export default function VerifyEmailPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const { token = '' } = useParams<{ token: string }>()
    const checkAuth = useAuthStore((state) => state.checkAuth)
    const [state, setState] = useState<State>('working')
    const copy: Record<State, { title: string; description: string }> = {
        working: { title: t('verifyEmail.workingTitle'), description: t('verifyEmail.workingDescription') },
        done: { title: t('verifyEmail.doneTitle'), description: t('verifyEmail.doneDescription') },
        invalid: { title: t('verifyEmail.invalidTitle'), description: t('verifyEmail.invalidDescription') },
    }

    useEffect(() => {
        let cancelled = false
        authApi.verifyEmail(token)
            .then(async () => {
                // A session already open in this browser learns about it at once.
                await checkAuth()
                if (!cancelled) setState('done')
            })
            .catch(() => {
                if (!cancelled) setState('invalid')
            })
        return () => {
            cancelled = true
        }
    }, [token, checkAuth])

    return (
        <div className="flex min-h-screen items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <div className="mb-2 flex justify-center">
                        {state === 'working' && <Loader2 className="size-8 animate-spin text-muted-foreground" />}
                        {state === 'done' && <MailCheck className="size-8 text-primary" />}
                        {state === 'invalid' && <MailX className="size-8 text-destructive" />}
                    </div>
                    <CardTitle className="text-2xl">{copy[state].title}</CardTitle>
                    <CardDescription>{copy[state].description}</CardDescription>
                </CardHeader>
                {state !== 'working' && (
                    <CardContent>
                        <Button className="w-full" onClick={() => navigate('/', { replace: true })}>
                            {t('verifyEmail.continue')}
                        </Button>
                    </CardContent>
                )}
            </Card>
        </div>
    )
}
