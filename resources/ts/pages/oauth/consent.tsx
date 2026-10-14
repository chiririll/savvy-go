import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Check, Loader2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Logo } from '@/components/shared/Logo'
import { oauthApi, type OAuthRequest } from '@/api/oauth'
import { useUser } from '@/stores/auth'
import { getApiErrorMessage } from '@/lib/api-error'

/** Where an AI app (Claude web, ...) sends the browser to ask for read access. */
export default function OAuthConsentPage() {
    const { t } = useTranslation('auth')
    const [params] = useSearchParams()
    const user = useUser()
    const [busy, setBusy] = useState<'allow' | 'deny' | null>(null)
    const [done, setDone] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const request = useMemo<OAuthRequest | null>(() => {
        const get = (key: string) => params.get(key) ?? ''
        const req = {
            client_id: get('client_id'),
            redirect_uri: get('redirect_uri'),
            state: get('state'),
            code_challenge: get('code_challenge'),
            code_challenge_method: get('code_challenge_method'),
        }
        return req.client_id && req.redirect_uri && req.code_challenge ? req : null
    }, [params])

    const { data: info, isPending, isError } = useQuery({
        queryKey: ['oauth-client', request?.client_id, request?.redirect_uri],
        queryFn: () => oauthApi.info(request!.client_id, request!.redirect_uri),
        enabled: request !== null,
        retry: false,
    })

    const host = (() => {
        try {
            return new URL(request?.redirect_uri ?? '').host
        } catch {
            return ''
        }
    })()
    const client = info?.client_name ?? t('oauth.unknownClient')
    const invalid = request === null || isError

    const decide = async (approve: boolean) => {
        if (!request) return
        setBusy(approve ? 'allow' : 'deny')
        setError(null)
        try {
            const url = await oauthApi.decide(request, approve)
            setDone(true)
            window.location.assign(url)
        } catch (e) {
            setError(getApiErrorMessage(e, t('oauth.failed')))
            setBusy(null)
        }
    }

    const header = (title: string, description: string) => (
        <CardHeader className="text-center">
            <div className="flex justify-center mb-4">
                <div className="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                    <Logo className="size-7" />
                </div>
            </div>
            <CardTitle className="text-2xl">{title}</CardTitle>
            <CardDescription>{description}</CardDescription>
        </CardHeader>
    )

    return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                {request !== null && isPending ? (
                    <CardContent className="flex justify-center py-12">
                        <Loader2 className="size-8 animate-spin text-muted-foreground" />
                    </CardContent>
                ) : invalid ? (
                    header(t('oauth.invalidTitle'), t('oauth.invalidDescription'))
                ) : done ? (
                    header(t('oauth.title', { client }), t('oauth.redirecting', { client }))
                ) : (
                    <>
                        {header(t('oauth.title', { client }), t('oauth.description', { client }))}
                        <CardContent className="space-y-4 text-sm">
                            <div>
                                <p className="font-medium mb-1">{t('oauth.canTitle')}</p>
                                <ul className="space-y-1 text-muted-foreground">
                                    <li className="flex gap-2">
                                        <Check className="size-4 mt-0.5 shrink-0 text-primary" />
                                        {t('oauth.can1')}
                                    </li>
                                    <li className="flex gap-2">
                                        <Check className="size-4 mt-0.5 shrink-0 text-primary" />
                                        {t('oauth.can2')}
                                    </li>
                                </ul>
                            </div>
                            <div>
                                <p className="font-medium mb-1">{t('oauth.cannotTitle')}</p>
                                <ul className="space-y-1 text-muted-foreground">
                                    <li className="flex gap-2">
                                        <X className="size-4 mt-0.5 shrink-0 text-destructive" />
                                        {t('oauth.cannot1')}
                                    </li>
                                    <li className="flex gap-2">
                                        <X className="size-4 mt-0.5 shrink-0 text-destructive" />
                                        {t('oauth.cannot2')}
                                    </li>
                                </ul>
                            </div>
                            <p className="text-muted-foreground">{t('oauth.revoke')}</p>
                            {host && <p className="font-medium">{t('oauth.returnsTo', { host })}</p>}
                            {user && <p className="text-xs text-muted-foreground">{t('oauth.signedInAs', { email: user.email })}</p>}
                            {error && (
                                <Alert variant="destructive">
                                    <AlertDescription>{error}</AlertDescription>
                                </Alert>
                            )}
                        </CardContent>
                        <CardFooter className="gap-2">
                            <Button variant="outline" className="flex-1" disabled={busy !== null} onClick={() => decide(false)}>
                                {busy === 'deny' && <Loader2 className="size-4 animate-spin" />}
                                {t('oauth.deny')}
                            </Button>
                            <Button className="flex-1" disabled={busy !== null} onClick={() => decide(true)}>
                                {busy === 'allow' && <Loader2 className="size-4 animate-spin" />}
                                {t('oauth.allow')}
                            </Button>
                        </CardFooter>
                    </>
                )}
            </Card>
        </div>
    )
}
