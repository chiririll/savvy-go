import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { ResponsiveDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useResourceMutation } from '@/hooks/use-crud'
import i18n from '@/lib/i18n'
import { isLaravelFile, laravelActionLabel, LaravelRestoreOptions, type LaravelTarget } from './LaravelRestoreOptions'
import type { BackupSource } from './source'

interface Props {
    source: BackupSource
    open: boolean
    onOpenChange: (open: boolean) => void
}

export function UploadBackupDialog({ source, open, onOpenChange }: Props) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const [note, setNote] = useState('')
    const [file, setFile] = useState<File | null>(null)
    const [appKey, setAppKey] = useState('')
    const [laravelTarget, setLaravelTarget] = useState<LaravelTarget | null>(null)

    const uploadBackup = useResourceMutation({
        mutationFn: ({ file, note }: { file: File; note?: string }) => source.upload(file, note),
        invalidateKeys: [source.key],
        successMessage: i18n.t('toasts.backup.uploaded'),
    })
    const restoreLaravel = useResourceMutation({
        mutationFn: ({ file, target, appKey }: { file: File; target: LaravelTarget; appKey?: string }) =>
            source.laravel!.restore(file, target, appKey),
        invalidateAll: true,
        successMessage: (_, { target }) =>
            target === 'new' ? i18n.t('toasts.backup.restored') : i18n.t('toasts.backup.upgraded'),
    })

    const laravelUpload = !!source.laravel && isLaravelFile(file)
    const target = laravelTarget ?? source.laravel?.targets[0]
    const pending = uploadBackup.isPending || restoreLaravel.isPending

    const close = () => {
        onOpenChange(false)
        setNote('')
        setFile(null)
        setAppKey('')
        setLaravelTarget(null)
    }

    const handleUpload = () => {
        if (!file) return
        if (laravelUpload) {
            restoreLaravel.mutate(
                { file, target: target!, appKey: target === 'server' ? appKey.trim() || undefined : undefined },
                { onSuccess: close },
            )
            return
        }
        uploadBackup.mutate({ file, note: note || undefined }, { onSuccess: close })
    }

    return (
        <ResponsiveDialog
            open={open}
            onOpenChange={(next) => (next ? onOpenChange(true) : close())}
            title={t('backups.uploadTitle')}
            description={t('backups.uploadDescription')}
            footer={
                <>
                    <Button variant="outline" onClick={close}>
                        {tCommon('actions.cancel')}
                    </Button>
                    <Button onClick={handleUpload} disabled={!file || pending}>
                        {pending && <Loader2 className="size-4 mr-2 animate-spin" />}
                        {laravelUpload ? laravelActionLabel(t, target!) : t('backups.upload')}
                    </Button>
                </>
            }
        >
            <div className="space-y-4">
                <div className="space-y-2">
                    <Label htmlFor="file">{t('backups.backupFile')}</Label>
                    <Input
                        id="file"
                        type="file"
                        accept=".zip,.sqlite"
                        onChange={(e) => setFile(e.target.files?.[0] || null)}
                    />
                </div>
                {laravelUpload ? (
                    <LaravelRestoreOptions
                        targets={source.laravel!.targets}
                        target={target!}
                        onTarget={setLaravelTarget}
                        appKey={appKey}
                        onAppKey={setAppKey}
                    />
                ) : (
                    <div className="space-y-2">
                        <Label htmlFor="upload-note">{t('backups.noteOptional')}</Label>
                        <Input
                            id="upload-note"
                            placeholder={t('backups.uploadNotePlaceholder')}
                            value={note}
                            onChange={(e) => setNote(e.target.value)}
                        />
                    </div>
                )}
            </div>
        </ResponsiveDialog>
    )
}
