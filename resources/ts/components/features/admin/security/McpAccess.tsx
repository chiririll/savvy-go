import { testId } from '@/lib/test-id'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Bot, Copy, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import { SettingsSection } from '@/components/shared'
import { Confirm } from '@/components/shared/dialogs/Confirm'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
    useAddOAuthHost,
    useCreateOAuthClient,
    useOAuthClients,
    useOAuthHosts,
    useRemoveOAuthHost,
    useOAuthRegistration,
    useRemoveOAuthClient,
    useSetOAuthRegistration,
} from '@/hooks/use-oauth-registration'
import type { CreatedOAuthClient, OAuthClient } from '@/api/oauth'
import { intlLocale } from '@/lib/i18n/i18n'

/** One registered AI app, with what the server saw of its registration. */
export function McpApp({ app }: { app: OAuthClient }) {
    const { t } = useTranslation('settings')
    const remove = useRemoveOAuthClient()
    const when = new Date(app.registered_at).toLocaleString(intlLocale(), {
        dateStyle: 'medium',
        timeStyle: 'short',
    })
    const kinds = {
        dcr: t('admin.mcp.kind.dcr'),
        cimd: t('admin.mcp.kind.cimd'),
        manual: t('admin.mcp.kind.manual'),
    }
    const facts: [string, string][] = [
        [t('admin.mcp.apps.ip'), app.ip],
        [t('admin.mcp.apps.userAgent'), app.user_agent],
        [t('admin.mcp.apps.origin'), app.origin],
        [t('admin.mcp.apps.redirects'), app.redirect_uris.join(', ')],
        [t('admin.mcp.apps.registeredAt'), when],
    ]

    return (
        <li className="space-y-3 px-3 py-3" {...testId(`mcp-app-${app.client_id}`)}>
            <div className="flex items-center justify-between gap-3">
                <div className="flex min-w-0 items-center gap-2">
                    <p className="truncate text-sm font-medium">{app.name}</p>
                    <Badge variant="secondary">{kinds[app.kind]}</Badge>
                    {app.confidential && <Badge variant="outline">{t('admin.mcp.confidentialBadge')}</Badge>}
                </div>
                <div className="flex shrink-0 gap-2">
                    <Confirm
                        trigger={
                            <Button variant="outline" size="sm" disabled={remove.isPending}>
                                <Trash2 className="mr-2 size-4" />
                                {t('admin.mcp.apps.disconnect')}
                            </Button>
                        }
                        title={t('admin.mcp.apps.removeTitle', {
                            name: app.name,
                        })}
                        description={t('admin.mcp.apps.removeDescription')}
                        action={t('admin.mcp.apps.disconnect')}
                        onConfirm={() => remove.mutate(app.client_id)}
                    />
                </div>
            </div>
            <dl className="grid gap-x-3 gap-y-1 text-xs sm:grid-cols-[8rem_1fr]">
                {facts.map(([label, value]) => (
                    <div key={label} className="contents">
                        <dt className="text-muted-foreground">{label}</dt>
                        <dd className="break-all font-mono">{value || t('admin.mcp.apps.unknown')}</dd>
                    </div>
                ))}
            </dl>
        </li>
    )
}

/** Copies a value the admin has to paste into the app. */
function CopyField({ label, value }: { label: string; value: string }) {
    const { t } = useTranslation('settings')
    return (
        <div className="space-y-1.5">
            <Label>{label}</Label>
            <div className="flex gap-2">
                <Input readOnly value={value} className="font-mono text-xs" />
                <Button
                    variant="outline"
                    size="icon"
                    title={t('admin.mcp.copy')}
                    onClick={async () => {
                        await navigator.clipboard.writeText(value)
                        toast.success(t('admin.mcp.copied'))
                    }}
                >
                    <Copy className="size-4" />
                </Button>
            </div>
        </div>
    )
}

/** Hosts whose apps may identify themselves with a metadata document URL. Starts empty, so the feature is off. */
export function McpHosts() {
    const { t } = useTranslation('settings')
    const { data: hosts } = useOAuthHosts()
    const add = useAddOAuthHost()
    const remove = useRemoveOAuthHost()
    const [host, setHost] = useState('')

    return (
        <div className="space-y-2" {...testId('mcp-hosts')}>
            <Label htmlFor="mcp-host">{t('admin.mcp.hosts.title')}</Label>
            <p className="text-xs text-muted-foreground">{t('admin.mcp.hosts.help')}</p>
            {hosts?.length ? (
                <ul className="flex flex-wrap gap-2">
                    {hosts.map((h) => (
                        <li key={h}>
                            <Badge variant="secondary" className="gap-1 font-mono">
                                {h}
                                <button
                                    type="button"
                                    title={t('admin.mcp.hosts.remove', {
                                        host: h,
                                    })}
                                    aria-label={t('admin.mcp.hosts.remove', {
                                        host: h,
                                    })}
                                    onClick={() => remove.mutate(h)}
                                >
                                    <X className="size-3" />
                                </button>
                            </Badge>
                        </li>
                    ))}
                </ul>
            ) : (
                <p className="text-sm text-muted-foreground">{t('admin.mcp.hosts.none')}</p>
            )}
            <form
                className="flex gap-2"
                onSubmit={(e) => {
                    e.preventDefault()
                    if (host.trim())
                        add.mutate(host.trim(), {
                            onSuccess: () => setHost(''),
                        })
                }}
            >
                <Input
                    id="mcp-host"
                    value={host}
                    onChange={(e) => setHost(e.target.value)}
                    placeholder={t('admin.mcp.hosts.placeholder')}
                    className="font-mono text-xs"
                />
                <Button type="submit" variant="outline" disabled={add.isPending || !host.trim()}>
                    {t('admin.mcp.hosts.add')}
                </Button>
            </form>
        </div>
    )
}

/** Creates a client for apps that ask for a client ID (and a secret) in their own settings. */
export function McpOwnClient() {
    const { t } = useTranslation('settings')
    const create = useCreateOAuthClient()
    const [name, setName] = useState('')
    const [redirects, setRedirects] = useState('')
    const [confidential, setConfidential] = useState(false)
    const [created, setCreated] = useState<CreatedOAuthClient | null>(null)
    const uris = redirects
        .split('\n')
        .map((u) => u.trim())
        .filter(Boolean)

    if (created) {
        return (
            <div className="space-y-3 rounded-lg border p-3" {...testId('mcp-own-created')}>
                <p className="text-sm font-medium">{t('admin.mcp.own.createdTitle')}</p>
                <CopyField label={t('admin.mcp.own.clientId')} value={created.client_id} />
                {created.client_secret && (
                    <>
                        <CopyField label={t('admin.mcp.own.clientSecret')} value={created.client_secret} />
                        <p className="text-xs text-muted-foreground">{t('admin.mcp.own.secretOnce')}</p>
                    </>
                )}
                <Button variant="outline" size="sm" onClick={() => setCreated(null)}>
                    {t('admin.mcp.own.done')}
                </Button>
            </div>
        )
    }

    return (
        <form
            className="space-y-3"
            {...testId('mcp-own-form')}
            onSubmit={(e) => {
                e.preventDefault()
                create.mutate(
                    { name: name.trim(), redirect_uris: uris, confidential },
                    {
                        onSuccess: (client) => {
                            setCreated(client)
                            setName('')
                            setRedirects('')
                            setConfidential(false)
                        },
                    },
                )
            }}
        >
            <Label>{t('admin.mcp.own.title')}</Label>
            <p className="text-xs text-muted-foreground">{t('admin.mcp.own.help')}</p>
            <div className="space-y-1.5">
                <Label htmlFor="mcp-own-name">{t('admin.mcp.own.name')}</Label>
                <Input id="mcp-own-name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-1.5">
                <Label htmlFor="mcp-own-redirects">{t('admin.mcp.own.redirects')}</Label>
                <Textarea
                    id="mcp-own-redirects"
                    value={redirects}
                    onChange={(e) => setRedirects(e.target.value)}
                    className="font-mono text-xs"
                    rows={3}
                />
            </div>
            <div className="flex items-center gap-2">
                <Checkbox
                    id="mcp-own-confidential"
                    checked={confidential}
                    onCheckedChange={(v) => setConfidential(v === true)}
                />
                <Label htmlFor="mcp-own-confidential">{t('admin.mcp.own.confidential')}</Label>
            </div>
            <Button type="submit" variant="outline" size="sm" disabled={create.isPending || uris.length === 0}>
                {t('admin.mcp.own.create')}
            </Button>
        </form>
    )
}

/** Dynamic registration: the window in which one new AI app (Claude, ...) may register itself; closed it looks like it does not exist. */
export function McpRegistration() {
    const { t } = useTranslation('settings')
    const { data } = useOAuthRegistration()
    const setOpen = useSetOAuthRegistration()
    const open = data?.open ?? false
    const until = data?.expires_at
        ? new Date(data.expires_at).toLocaleTimeString(intlLocale(), {
              hour: '2-digit',
              minute: '2-digit',
          })
        : ''

    return (
        <div className="space-y-3" {...testId('mcp-dcr')}>
            <p className="text-xs text-muted-foreground">{t('admin.mcp.dcr.help')}</p>
            <p className="text-sm text-muted-foreground">
                {open ? t('admin.mcp.open', { time: until }) : t('admin.mcp.closed')}
            </p>
            <Button
                variant={open ? 'outline' : 'default'}
                size="sm"
                disabled={setOpen.isPending}
                onClick={() => setOpen.mutate(!open)}
                {...testId('mcp-registration')}
            >
                {open ? t('admin.mcp.close') : t('admin.mcp.allow')}
            </Button>
        </div>
    )
}

/** The three ways an AI app can identify itself, one tab each, then the apps that did. */
export function McpAccess() {
    const { t } = useTranslation('settings')
    const { data: apps } = useOAuthClients()
    const endpoint = `${window.location.origin}/mcp`

    return (
        <SettingsSection icon={Bot} title={t('admin.mcp.title')} description={t('admin.mcp.description')}>
            <div className="space-y-4">
                <div className="space-y-1.5">
                    <Label htmlFor="mcp-endpoint">{t('admin.mcp.endpoint')}</Label>
                    <div className="flex gap-2">
                        <Input id="mcp-endpoint" readOnly value={endpoint} className="font-mono text-xs" />
                        <Button
                            variant="outline"
                            size="icon"
                            title={t('admin.mcp.copy')}
                            onClick={async () => {
                                await navigator.clipboard.writeText(endpoint)
                                toast.success(t('admin.mcp.copied'))
                            }}
                        >
                            <Copy className="size-4" />
                        </Button>
                    </div>
                </div>
                <Tabs defaultValue="dcr" {...testId('mcp-modes')}>
                    <TabsList className="h-auto w-full flex-wrap sm:w-fit">
                        <TabsTrigger value="dcr">{t('admin.mcp.tabs.dcr')}</TabsTrigger>
                        <TabsTrigger value="cimd">{t('admin.mcp.tabs.cimd')}</TabsTrigger>
                        <TabsTrigger value="own">{t('admin.mcp.tabs.own')}</TabsTrigger>
                    </TabsList>
                    <TabsContent value="dcr" className="rounded-lg border p-3">
                        <McpRegistration />
                    </TabsContent>
                    <TabsContent value="cimd" className="rounded-lg border p-3">
                        <McpHosts />
                    </TabsContent>
                    <TabsContent value="own" className="rounded-lg border p-3">
                        <McpOwnClient />
                    </TabsContent>
                </Tabs>
                <div className="space-y-2">
                    <Label>{t('admin.mcp.apps.title')}</Label>
                    {apps?.length ? (
                        <ul className="divide-y rounded-lg border">
                            {apps.map((app) => (
                                <McpApp key={app.client_id} app={app} />
                            ))}
                        </ul>
                    ) : (
                        <p className="text-sm text-muted-foreground">{t('admin.mcp.apps.none')}</p>
                    )}
                    <p className="text-xs text-muted-foreground">{t('admin.mcp.apps.help')}</p>
                </div>
            </div>
        </SettingsSection>
    )
}
