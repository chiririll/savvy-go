import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { oauthApi } from '@/api/oauth'
import { useApiTokenCapabilities } from '@/hooks'
import { type SpaceChoice } from '@/components/features/api-tokens/SpacePicker'
import { kindOptions, type ApiPermission, type ApiResource } from '@/types'
import { isConfirmCancelled, withConfirmation } from '@/stores/confirm'
import { getApiErrorMessage } from '@/lib/api-error'
import { ConsentConfirmDialog, ConsentHeader } from '@/components/features/oauth/ConsentParts'
import { ConsentForm } from '@/components/features/oauth/ConsentForm'
import { changedResources, parseOAuthRequest, redirectHost } from '@/components/features/oauth/consent'

/** Where an AI app (Claude web, ...) sends the browser to ask for access. Read unless the user ticks more. */
export default function OAuthConsentPage() {
    const { t } = useTranslation('auth')
    const [params] = useSearchParams()
    const [busy, setBusy] = useState<'allow' | 'deny' | null>(null)
    const [done, setDone] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [confirming, setConfirming] = useState(false)
    const [spaceChoice, setSpaceChoice] = useState<SpaceChoice>({ all: false, ids: [] })
    // Until the user changes it, the app gets read access to everything it can show.
    const [chosen, setChosen] = useState<ApiPermission[] | null>(null)

    const request = useMemo(() => parseOAuthRequest(params), [params])

    const {
        data: info,
        isPending,
        isError,
    } = useQuery({
        queryKey: ['oauth-client', request?.client_id, request?.redirect_uri],
        queryFn: () => oauthApi.info(request!.client_id, request!.redirect_uri),
        enabled: request !== null,
        retry: false,
    })

    // What a connected app can be given comes from the server; until the user changes it, it gets the lowest preset.
    const { data: capabilities } = useApiTokenCapabilities()
    const options = capabilities ? kindOptions(capabilities, 'mcp') : null
    const resources: ApiResource[] = options?.resources ?? []
    const permissions = chosen ?? options?.presets[0]?.permissions ?? []

    // Changing data is never the default: when the user ticked it, they confirm once more.
    const { changes, deletes } = changedResources(resources, permissions)
    const host = redirectHost(request?.redirect_uri)
    const client = info?.client_name ?? t('oauth.unknownClient')
    const invalid = request === null || isError

    const decide = async (approve: boolean) => {
        if (!request) return
        setBusy(approve ? 'allow' : 'deny')
        setError(null)
        try {
            // Allowing mints the app a token, so it asks for the password or code; denying asks for nothing.
            const url = approve
                ? await withConfirmation((fields) => oauthApi.decide(request, true, spaceChoice, permissions, fields))
                : await oauthApi.decide(request, false, spaceChoice, permissions)
            setDone(true)
            window.location.assign(url)
        } catch (e) {
            if (!isConfirmCancelled(e)) setError(getApiErrorMessage(e, t('oauth.failed')))
            setBusy(null)
        }
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-lg">
                {request !== null && isPending ? (
                    <CardContent className="flex justify-center py-12">
                        <Loader2 className="size-8 animate-spin text-muted-foreground" />
                    </CardContent>
                ) : invalid ? (
                    <ConsentHeader title={t('oauth.invalidTitle')} description={t('oauth.invalidDescription')} />
                ) : done ? (
                    <ConsentHeader
                        title={t('oauth.title', { client })}
                        description={t('oauth.redirecting', { client })}
                    />
                ) : (
                    <>
                        <ConsentHeader
                            title={t('oauth.title', { client })}
                            description={t('oauth.description', { client })}
                        />
                        <ConsentForm
                            spaces={info?.spaces ?? []}
                            options={options}
                            permissions={permissions}
                            onPermissionsChange={setChosen}
                            spaceChoice={spaceChoice}
                            onSpaceChoiceChange={setSpaceChoice}
                            changes={changes}
                            host={host}
                            error={error}
                            busy={busy}
                            onDecide={decide}
                            onAsk={() => setConfirming(true)}
                        />
                    </>
                )}
            </Card>
            <ConsentConfirmDialog
                open={confirming}
                client={client}
                changes={changes}
                deletes={deletes}
                onOpenChange={setConfirming}
                onAllow={() => decide(true)}
            />
        </div>
    )
}
