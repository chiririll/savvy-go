import { useTranslation } from 'react-i18next'
import { backupsApi } from '@/api/backups'
import { BackupsView, BackupSource } from '@/components/features/backups/BackupsView'

const serverBackups: BackupSource = {
    key: ['backups'],
    list: backupsApi.getAll,
    create: backupsApi.create,
    upload: backupsApi.upload,
    downloadUrl: backupsApi.download,
    restore: backupsApi.restore,
    remove: backupsApi.delete,
}

/** Backups of the whole server: every space and the signing key. */
export default function SystemBackupsPage() {
    const { t } = useTranslation('settings')
    return <BackupsView source={serverBackups} title={t('admin.backups.title')} description={t('admin.backups.description')} />
}
