import { useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { FolderInput } from 'lucide-react'
import { spacesApi } from '@/api/spaces'
import { BackupsView, BackupSource } from '@/components/features/backups/BackupsView'
import { Button } from '@/components/ui/button'
import { useCurrentSpace, useImportSpace, useSwitchSpace } from '@/hooks/use-spaces'
import { useUser } from '@/stores/auth'

/** Backups of the current space; only its admins reach the endpoints. */
export default function SpaceBackupsPage() {
    const { t } = useTranslation('settings')
    const space = useCurrentSpace()
    const user = useUser()
    const importSpace = useImportSpace()
    const switchSpace = useSwitchSpace()
    const fileInput = useRef<HTMLInputElement>(null)

    const source = useMemo<BackupSource | null>(() => {
        if (!space) return null
        const id = space.id
        return {
            key: ['space', id, 'backups'],
            list: () => spacesApi.backups(id),
            create: (note) => spacesApi.createBackup(id, note),
            upload: (file) => spacesApi.uploadBackup(id, file),
            downloadUrl: (name) => spacesApi.backupDownloadUrl(id, name),
            restore: (name) => spacesApi.restoreBackup(id, name),
            remove: (name) => spacesApi.deleteBackup(id, name),
            laravel: {
                targets: user?.role === 'guest' ? ['space'] : ['space', 'new'],
                restore: async (file, target) => {
                    if (target === 'space') return spacesApi.restoreLaravel(id, file)
                    await switchSpace((await spacesApi.importSpace(file)).id)
                },
            },
        }
    }, [space, user?.role, switchSpace])

    if (!space) return <></>
    if (space.role !== 'admin') {
        return <p className="text-sm text-muted-foreground">{t('spaces.backups.adminsOnly')}</p>
    }

    const importAction = user?.role !== 'guest' && (
        <>
            <Button variant="outline" onClick={() => fileInput.current?.click()} disabled={importSpace.isPending}>
                <FolderInput className="mr-2 size-4" />
                {t('spaces.backups.import')}
            </Button>
            <input
                ref={fileInput}
                type="file"
                accept=".zip,.sqlite"
                className="hidden"
                onChange={async (e) => {
                    const file = e.target.files?.[0]
                    e.target.value = ''
                    if (!file) return
                    const created = await importSpace.mutateAsync({ file })
                    await switchSpace(created.id)
                }}
            />
        </>
    )

    return (
        <BackupsView
            source={source!}
            title={t('spaces.backups.title', { name: space.name })}
            description={t('spaces.backups.description')}
            actions={importAction}
        />
    )
}
