import { testId } from '@/lib/test-id'
import { useTranslation } from 'react-i18next'
import { Bot, Copy, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import {
    useOAuthClients,
    useOAuthRegistration,
    useRemoveOAuthClient,
    useSetOAuthRegistration,
} from '@/hooks/use-oauth-registration'
import type { OAuthClient } from '@/api/oauth'
import { intlLocale } from '@/lib/i18n'

/** One registered AI app, with what the server saw of its registration. */
export function McpApp({ app }: { app: OAuthClient }) {
    const { t } = useTranslation('settings')
    const remove = useRemoveOAuthClient()
    const when = new Date(app.registered_at).toLocaleString(intlLocale(), { dateStyle: 'medium', timeStyle: 'short' })
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
                </div>
                <div className="flex shrink-0 gap-2">
                    <AlertDialog>
                        <AlertDialogTrigger asChild>
                            <Button variant="outline" size="sm" disabled={remove.isPending}>
                                <Trash2 className="mr-2 size-4" />
                                {t('admin.mcp.apps.disconnect')}
                            </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                            <AlertDialogHeader>
                                <AlertDialogTitle>
                                    {t('admin.mcp.apps.removeTitle', { name: app.name })}
                                </AlertDialogTitle>
                                <AlertDialogDescription>{t('admin.mcp.apps.removeDescription')}</AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                                <AlertDialogCancel>{t('actions.cancel', { ns: 'common' })}</AlertDialogCancel>
                                <AlertDialogAction onClick={() => remove.mutate(app.client_id)}>
                                    {t('admin.mcp.apps.disconnect')}
                                </AlertDialogAction>
                            </AlertDialogFooter>
                        </AlertDialogContent>
                    </AlertDialog>
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

/** The window in which one new AI app (Claude, ...) may register itself; closed it looks like it does not exist. */
export function McpAccess() {
    const { t } = useTranslation('settings')
    const { data } = useOAuthRegistration()
    const setOpen = useSetOAuthRegistration()
    const open = data?.open ?? false
    const { data: apps } = useOAuthClients()
    const endpoint = `${window.location.origin}/mcp`
    const until = data?.expires_at
        ? new Date(data.expires_at).toLocaleTimeString(intlLocale(), { hour: '2-digit', minute: '2-digit' })
        : ''

    return (
        <SettingsSection
            icon={Bot}
            title={t('admin.mcp.title')}
            description={t('admin.mcp.description')}
            actions={
                <Button
                    variant={open ? 'outline' : 'default'}
                    size="sm"
                    disabled={setOpen.isPending}
                    onClick={() => setOpen.mutate(!open)}
                    {...testId('mcp-registration')}
                >
                    {open ? t('admin.mcp.close') : t('admin.mcp.allow')}
                </Button>
            }
        >
            <div className="space-y-4">
                <p className="text-sm text-muted-foreground">
                    {open ? t('admin.mcp.open', { time: until }) : t('admin.mcp.closed')}
                </p>
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
