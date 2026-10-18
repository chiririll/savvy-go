import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ResponsiveDialog } from '@/components/shared/ResponsiveDialog'
import { formatDateTime } from '@/lib/format'

interface UserPasswordLinkDialogProps {
    url: string | null
    /** When the link stops working: an invitation lasts days, a reset an hour by default. */
    expiresAt?: string | null
    open: boolean
    onOpenChange: (open: boolean) => void
}

export function UserPasswordLinkDialog({
    url,
    expiresAt,
    open,
    onOpenChange,
}: UserPasswordLinkDialogProps) {
    const { t } = useTranslation(['pages', 'common'])
    const [copied, setCopied] = useState(false)

    const handleCopy = async () => {
        if (!url) return
        try {
            await navigator.clipboard.writeText(url)
            setCopied(true)
            window.setTimeout(() => setCopied(false), 2000)
        } catch {
            // clipboard may be unavailable over plain HTTP
        }
    }

    return (
        <ResponsiveDialog
            open={open}
            onOpenChange={onOpenChange}
            title={t('users.inviteLinkTitle')}
            description={t('users.inviteLinkDescription', { date: expiresAt ? formatDateTime(expiresAt) : '' })}
            className="sm:max-w-lg"
            footer={
                <Button type="button" onClick={() => onOpenChange(false)}>
                    {t('common:actions.done')}
                </Button>
            }
        >
            <div className="flex gap-2">
                <Input readOnly value={url ?? ''} className="font-mono text-sm" />
                <Button type="button" variant="outline" onClick={handleCopy}>
                    {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
            </div>
        </ResponsiveDialog>
    )
}
