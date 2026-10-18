import { testId } from '@/lib/test-id'
import { ReactNode, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { Plus, Upload } from 'lucide-react'
import { Page, PageHeader } from '@/components/shared'
import { Button } from '@/components/ui/button'
import type { Backup } from '@/types/backup'
import { BackupsTable } from './BackupsTable'
import { DeleteBackupDialog, RestoreBackupDialog } from './BackupConfirmDialogs'
import { CreateBackupDialog } from './CreateBackupDialog'
import { UploadBackupDialog } from './UploadBackupDialog'
import type { BackupSource } from './source'

export type { BackupSource } from './source'

interface BackupsViewProps {
    source: BackupSource
    title: string
    description: string
    /** Shown next to the create and upload buttons. */
    actions?: ReactNode
}

function downloadBackup(url: string, filename: string) {
    fetch(url, { credentials: 'same-origin' })
        .then((res) => res.blob())
        .then((blob) => {
            const a = document.createElement('a')
            a.href = URL.createObjectURL(blob)
            a.download = filename
            a.click()
            URL.revokeObjectURL(a.href)
        })
}

export function BackupsView({ source, title, description, actions }: BackupsViewProps) {
    const { t } = useTranslation('settings')
    const { data: backups, isLoading } = useQuery({ queryKey: source.key, queryFn: source.list })

    const [createOpen, setCreateOpen] = useState(false)
    const [uploadOpen, setUploadOpen] = useState(false)
    const [restoreOpen, setRestoreOpen] = useState(false)
    const [deleteOpen, setDeleteOpen] = useState(false)
    const [selected, setSelected] = useState<Backup | null>(null)

    const closeRestore = () => {
        setRestoreOpen(false)
        setSelected(null)
    }
    const closeDelete = () => {
        setDeleteOpen(false)
        setSelected(null)
    }

    return (
        <Page title={title}>
            <PageHeader title={title} description={description} />

            <div className="flex gap-2 mb-6">
                <Button onClick={() => setCreateOpen(true)} {...testId('create')}>
                    <Plus className="size-4 mr-2" />
                    {t('backups.create')}
                </Button>
                <Button variant="outline" onClick={() => setUploadOpen(true)} {...testId('upload')}>
                    <Upload className="size-4 mr-2" />
                    {t('backups.upload')}
                </Button>
                {actions}
            </div>

            <BackupsTable
                backups={backups}
                isLoading={isLoading}
                onDownload={(backup) => downloadBackup(source.downloadUrl(backup.filename), backup.filename)}
                onRestore={(backup) => {
                    setSelected(backup)
                    setRestoreOpen(true)
                }}
                onDelete={(backup) => {
                    setSelected(backup)
                    setDeleteOpen(true)
                }}
            />

            <CreateBackupDialog source={source} open={createOpen} onOpenChange={setCreateOpen} />
            <UploadBackupDialog source={source} open={uploadOpen} onOpenChange={setUploadOpen} />
            <RestoreBackupDialog source={source} backup={selected} open={restoreOpen} onClose={closeRestore} />
            <DeleteBackupDialog source={source} backup={selected} open={deleteOpen} onClose={closeDelete} />
        </Page>
    )
}
