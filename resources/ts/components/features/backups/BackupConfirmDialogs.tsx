import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
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
import { Button } from '@/components/ui/button'
import { useResourceMutation } from '@/hooks/use-crud'
import i18n from '@/lib/i18n'
import { formatDateTime } from '@/lib/format'
import { backupStatusHelp } from '@/lib/labels'
import type { Backup } from '@/types/backup'
import type { BackupSource } from './source'

interface Props {
    source: BackupSource
    /** The backup asked about; kept until the dialog is closed. */
    backup: Backup | null
    open: boolean
    onClose: () => void
}

export function RestoreBackupDialog({ source, backup, open, onClose }: Props) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const restoreBackup = useResourceMutation({
        mutationFn: (filename: string) => source.restore(filename),
        invalidateAll: true,
        successMessage: i18n.t('toasts.backup.restored'),
    })

    const handleRestore = () => {
        if (!backup?.restorable) return
        restoreBackup.mutate(backup.filename, { onSuccess: onClose })
    }

    return (
        <AlertDialog open={open} onOpenChange={(next) => !next && onClose()}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{t('backups.restoreTitle')}</AlertDialogTitle>
                    <AlertDialogDescription asChild>
                        <div className="space-y-2">
                            <p>
                                {t('backups.restoreDescription', {
                                    date: backup ? formatDateTime(backup.createdAt) : '',
                                })}
                            </p>
                            {backup && backup.status !== 'current' && <p>{backupStatusHelp(t, backup.status)}</p>}
                        </div>
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel disabled={restoreBackup.isPending}>
                        {tCommon('actions.cancel')}
                    </AlertDialogCancel>
                    <Button
                        type="button"
                        variant="destructive"
                        disabled={restoreBackup.isPending}
                        onClick={handleRestore}
                    >
                        {restoreBackup.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
                        {t('backups.restore')}
                    </Button>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}

export function DeleteBackupDialog({ source, backup, open, onClose }: Props) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const deleteBackup = useResourceMutation({
        mutationFn: (filename: string) => source.remove(filename),
        invalidateKeys: [source.key],
        successMessage: i18n.t('toasts.backup.deleted'),
    })

    const handleDelete = () => {
        if (!backup) return
        deleteBackup.mutate(backup.filename, { onSuccess: onClose })
    }

    return (
        <AlertDialog open={open} onOpenChange={(next) => !next && onClose()}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{t('backups.deleteTitle')}</AlertDialogTitle>
                    <AlertDialogDescription>
                        {t('backups.deleteDescription', { date: backup ? formatDateTime(backup.createdAt) : '' })}
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>{tCommon('actions.cancel')}</AlertDialogCancel>
                    <AlertDialogAction variant="destructive" onClick={handleDelete} disabled={deleteBackup.isPending}>
                        {deleteBackup.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
                        {tCommon('actions.delete')}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}
