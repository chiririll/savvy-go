import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { ResponsiveDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useResourceMutation } from '@/hooks/use-crud'
import i18n from '@/lib/i18n'
import type { BackupSource } from './source'

interface Props {
    source: BackupSource
    open: boolean
    onOpenChange: (open: boolean) => void
}

export function CreateBackupDialog({ source, open, onOpenChange }: Props) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const [note, setNote] = useState('')
    const createBackup = useResourceMutation({
        mutationFn: (note?: string) => source.create(note),
        invalidateKeys: [source.key],
        successMessage: i18n.t('toasts.backup.created'),
    })

    const handleCreate = () => {
        createBackup.mutate(note || undefined, {
            onSuccess: () => {
                onOpenChange(false)
                setNote('')
            },
        })
    }

    return (
        <ResponsiveDialog
            open={open}
            onOpenChange={onOpenChange}
            title={t('backups.createTitle')}
            description={t('backups.createDescription')}
            footer={
                <>
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                        {tCommon('actions.cancel')}
                    </Button>
                    <Button onClick={handleCreate} disabled={createBackup.isPending}>
                        {createBackup.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
                        {tCommon('actions.create')}
                    </Button>
                </>
            }
        >
            <div className="space-y-2">
                <Label htmlFor="note">{t('backups.noteOptional')}</Label>
                <Input
                    id="note"
                    placeholder={t('backups.notePlaceholder')}
                    value={note}
                    onChange={(e) => setNote(e.target.value)}
                />
            </div>
        </ResponsiveDialog>
    )
}
