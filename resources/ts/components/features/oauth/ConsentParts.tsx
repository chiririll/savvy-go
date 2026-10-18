import { useTranslation } from 'react-i18next'
import { CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
import { apiResourceLabel } from '@/lib/labels'
import { testId } from '@/lib/test-id'
import type { ApiResource } from '@/types'

export function ConsentHeader({ title, description }: { title: string; description: string }) {
    return (
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
}

interface ConsentConfirmDialogProps {
    open: boolean
    client: string
    changes: ApiResource[]
    deletes: ApiResource[]
    onOpenChange: (open: boolean) => void
    onAllow: () => void
}

/** The extra confirmation before an app is allowed to change or delete data. */
export function ConsentConfirmDialog({
    open,
    client,
    changes,
    deletes,
    onOpenChange,
    onAllow,
}: ConsentConfirmDialogProps) {
    const { t } = useTranslation('auth')
    const { t: tSettings } = useTranslation('settings')
    const list = (rs: ApiResource[]) => rs.map((r) => apiResourceLabel(tSettings, r)).join(', ')

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{t('oauth.confirmTitle', { client })}</AlertDialogTitle>
                    <AlertDialogDescription asChild>
                        <div className="space-y-2">
                            <p>{t('oauth.confirmChange', { client, resources: list(changes) })}</p>
                            {deletes.length > 0 && (
                                <p className="font-medium text-destructive">
                                    {t('oauth.confirmDelete', { resources: list(deletes) })}
                                </p>
                            )}
                            <p>{t('oauth.confirmAsYou')}</p>
                        </div>
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel {...testId('oauth-confirm-cancel')}>{t('oauth.confirmBack')}</AlertDialogCancel>
                    <AlertDialogAction onClick={onAllow} {...testId('oauth-confirm-allow')}>
                        {t('oauth.confirmAllow')}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}
