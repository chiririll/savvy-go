import { Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { useSettings } from '@/hooks/use-settings'

function CopyableUrl({ label, value }: { label: string; value: string }) {
    const { t } = useTranslation('forms')

    return (
        <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground">{label}</p>
            <div className="flex items-center gap-2 rounded-md border bg-background px-3 py-2">
                <code className="flex-1 truncate font-mono text-xs">{value}</code>
                <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-7 shrink-0 text-muted-foreground"
                    onClick={() => {
                        navigator.clipboard?.writeText(value)
                        toast.success(t('sso.copied'))
                    }}
                >
                    <Copy className="size-3.5" />
                </Button>
            </div>
        </div>
    )
}

export function RedirectUrls({ slug, protocol }: { slug: string; protocol: string }) {
    const { t } = useTranslation('forms')
    const { data: settings } = useSettings()
    // The server builds redirect URIs from the public URL, not from the address this page is opened at.
    const appUrl = settings?.app_url?.trim().replace(/\/+$/, '')
    const base = `${appUrl}/api/auth/sso/${slug}`

    if (!appUrl) {
        return (
            <div className="rounded-lg border border-dashed bg-muted/30 p-4">
                <p className="text-xs text-muted-foreground">{t('sso.needAppUrl')}</p>
            </div>
        )
    }

    return (
        <div className="space-y-3 rounded-lg border border-dashed bg-muted/30 p-4">
            <p className="text-xs text-muted-foreground">{t('sso.registerUrls')}</p>
            {protocol === 'saml' ? (
                <>
                    <CopyableUrl label={t('sso.acsUrl')} value={`${base}/acs`} />
                    <CopyableUrl label={t('sso.spEntityId')} value={`${base}/metadata`} />
                </>
            ) : (
                <CopyableUrl label={t('sso.redirectUrl')} value={`${base}/callback`} />
            )}
        </div>
    )
}
