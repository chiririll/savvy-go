import { useTranslation } from 'react-i18next'
import { Confirm } from '@/components/shared/dialogs/Confirm'
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
        <Confirm
            open={open}
            onOpenChange={(next) => !next && onClose()}
            title={t('backups.restoreTitle')}
            description={
                <>
                    <p>{t('backups.restoreDescription', { date: backup ? formatDateTime(backup.createdAt) : '' })}</p>
                    {backup && backup.status !== 'current' && <p>{backupStatusHelp(t, backup.status)}</p>}
                </>
            }
            action={t('backups.restore')}
            destructive
            pending={restoreBackup.isPending}
            onConfirm={handleRestore}
        />
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
        <Confirm
            open={open}
            onOpenChange={(next) => !next && onClose()}
            title={t('backups.deleteTitle')}
            description={t('backups.deleteDescription', { date: backup ? formatDateTime(backup.createdAt) : '' })}
            action={tCommon('actions.delete')}
            destructive
            pending={deleteBackup.isPending}
            onConfirm={handleDelete}
        />
    )
}
