import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Logo } from '@/components/shared/Logo'
import { oauthApi, type OAuthRequest } from '@/api/oauth'
import { SpacePicker, hasSpaces, type SpaceChoice } from '@/components/features/api-tokens/SpacePicker'
import { PermissionPicker } from '@/components/features/api-tokens/PermissionPicker'
import { permissionLevel, presetPermissions, type ApiPermission, type ApiResource } from '@/types'
import { apiResourceLabel } from '@/lib/labels'
import { useUser } from '@/stores/auth'
import { getApiErrorMessage } from '@/lib/api-error'
import { testId } from '@/lib/test-id'

/** Where an AI app (Claude web, ...) sends the browser to ask for access. Read unless the user ticks more. */
export default function OAuthConsentPage() {
    const { t } = useTranslation('auth')
    const { t: tSettings } = useTranslation('settings')
    const [params] = useSearchParams()
    const user = useUser()
    const [busy, setBusy] = useState<'allow' | 'deny' | null>(null)
    const [done, setDone] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [confirming, setConfirming] = useState(false)
    const [spaceChoice, setSpaceChoice] = useState<SpaceChoice>({ all: false, ids: [] })
    // Until the user changes it, the app gets read access to everything it can show.
    const [chosen, setChosen] = useState<ApiPermission[] | null>(null)

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

    const resources = (info?.resources ?? []) as ApiResource[]
    const permissions = chosen ?? presetPermissions('viewer', resources)

    // Changing data is never the default: when the user ticked it, they confirm once more.
    const changes = resources.filter((r) => permissionLevel(permissions, r) !== null && permissionLevel(permissions, r) !== 'read')
    const deletes = resources.filter((r) => permissionLevel(permissions, r) === 'delete')
    const list = (rs: ApiResource[]) => rs.map((r) => apiResourceLabel(tSettings, r)).join(', ')

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
            const url = await oauthApi.decide(request, approve, spaceChoice, permissions)
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
            <Card className="w-full max-w-lg">
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
                        <CardContent className="space-y-5 text-sm">
                            <div className="space-y-2">
                                <p className="font-medium">{t('oauth.canTitle')}</p>
                                <PermissionPicker value={permissions} onChange={setChosen} resources={resources} />
                                <p className="text-xs text-muted-foreground">
                                    {changes.length === 0 && `${t('oauth.readOnly')} `}
                                    {t('oauth.cannot')}
                                </p>
                            </div>
                            <div>
                                <p className="font-medium mb-2">{t('oauth.spacesTitle')}</p>
                                <SpacePicker
                                    idPrefix="oauth-space"
                                    spaces={info?.spaces ?? []}
                                    value={spaceChoice}
                                    onChange={setSpaceChoice}
                                />
                                {!hasSpaces(spaceChoice) && (
                                    <p className="mt-1 text-xs text-muted-foreground">{t('oauth.needSpace')}</p>
                                )}
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
                            <Button variant="outline" className="flex-1" disabled={busy !== null} onClick={() => decide(false)} {...testId('oauth-deny')}>
                                {busy === 'deny' && <Loader2 className="size-4 animate-spin" />}
                                {t('oauth.deny')}
                            </Button>
                            <Button className="flex-1" disabled={busy !== null || !hasSpaces(spaceChoice) || permissions.length === 0} onClick={() => (changes.length > 0 ? setConfirming(true) : decide(true))} {...testId('oauth-allow')}>
                                {busy === 'allow' && <Loader2 className="size-4 animate-spin" />}
                                {t('oauth.allow')}
                            </Button>
                        </CardFooter>
                    </>
                )}
            </Card>
            <AlertDialog open={confirming} onOpenChange={setConfirming}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('oauth.confirmTitle', { client })}</AlertDialogTitle>
                        <AlertDialogDescription asChild>
                            <div className="space-y-2">
                                <p>{t('oauth.confirmChange', { client, resources: list(changes) })}</p>
                                {deletes.length > 0 && (
                                    <p className="font-medium text-destructive">{t('oauth.confirmDelete', { resources: list(deletes) })}</p>
                                )}
                                <p>{t('oauth.confirmAsYou')}</p>
                            </div>
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel {...testId('oauth-confirm-cancel')}>{t('oauth.confirmBack')}</AlertDialogCancel>
                        <AlertDialogAction onClick={() => decide(true)} {...testId('oauth-confirm-allow')}>
                            {t('oauth.confirmAllow')}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}
