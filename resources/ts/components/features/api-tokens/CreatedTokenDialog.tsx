import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Check, Copy } from 'lucide-react'
import { ResponsiveDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'

interface CreatedTokenDialogProps {
    /** The plain token; the dialog is open while there is one. */
    token: string | null
    onClose: () => void
}

export function CreatedTokenDialog({ token, onClose }: CreatedTokenDialogProps) {
    const { t } = useTranslation('settings')
    const [copied, setCopied] = useState(false)

    const handleCopy = async () => {
        if (!token) return
        try {
            await navigator.clipboard.writeText(token)
            setCopied(true)
        } catch {
            toast.error(t('api.copyFailed'))
        }
    }

    const close = () => {
        setCopied(false)
        onClose()
    }

    return (
        <ResponsiveDialog
            open={token !== null}
            onOpenChange={(open) => !open && close()}
            title={t('api.created.title')}
            description={t('api.created.description')}
            className="sm:max-w-lg"
            footer={<Button onClick={close}>{t('api.created.done')}</Button>}
        >
            <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 break-all rounded-md border bg-muted px-3 py-2 text-xs">{token}</code>
                <Button variant="outline" size="icon" onClick={handleCopy} aria-label={t('api.created.copy')}>
                    {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
            </div>
        </ResponsiveDialog>
    )
}
