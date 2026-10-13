import { useTranslation } from 'react-i18next'
import { backupsApi } from '@/api/backups'
import { spacesApi } from '@/api/spaces'
import { BackupsView, BackupSource } from '@/components/features/backups/BackupsView'
import { useSwitchSpace } from '@/hooks/use-spaces'

/** Backups of the whole server: every space and the signing key. */
export default function SystemBackupsPage() {
    const { t } = useTranslation('settings')
    const switchSpace = useSwitchSpace()

    const source: BackupSource = {
        key: ['backups'],
        list: backupsApi.getAll,
        create: backupsApi.create,
        upload: backupsApi.upload,
        downloadUrl: backupsApi.download,
        restore: backupsApi.restore,
        remove: backupsApi.delete,
        laravel: {
            targets: ['server', 'new'],
            restore: async (file, target, appKey) => {
                if (target === 'server') return backupsApi.restoreLaravel(file, appKey)
                await switchSpace((await spacesApi.importSpace(file)).id)
            },
        },
    }

    return <BackupsView source={source} title={t('admin.backups.title')} description={t('admin.backups.description')} />
}
