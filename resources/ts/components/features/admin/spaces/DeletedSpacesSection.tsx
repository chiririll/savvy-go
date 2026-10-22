import { useTranslation } from 'react-i18next'
import { ArchiveRestore } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { useDeletedSpaces, useRestoreDeletedSpace } from '@/hooks'
import { formatBytes, formatDateTime } from '@/lib/format'

export function DeletedSpacesSection() {
    const { t } = useTranslation('settings')
    const { data: deleted = [] } = useDeletedSpaces()
    const restore = useRestoreDeletedSpace()

    return (
        <SettingsSection
            icon={ArchiveRestore}
            title={t('admin.spaces.deletedTitle')}
            description={t('admin.spaces.deletedDescription')}
        >
            {deleted.length ? (
                <ul className="divide-y rounded-lg border text-sm">
                    {deleted.map((b) => (
                        <li key={b.filename} className="flex items-center justify-between gap-3 px-3 py-2">
                            <div className="min-w-0">
                                <p className="truncate font-medium">{b.spaceName || b.filename}</p>
                                <p className="text-xs text-muted-foreground">
                                    {formatDateTime(b.createdAt)} · {formatBytes(b.size)}
                                </p>
                            </div>
                            <Button
                                size="sm"
                                variant="outline"
                                disabled={restore.isPending}
                                onClick={() => restore.mutate(b.filename)}
                            >
                                {t('admin.spaces.restore')}
                            </Button>
                        </li>
                    ))}
                </ul>
            ) : (
                <p className="text-sm text-muted-foreground">{t('admin.spaces.noDeleted')}</p>
            )}
        </SettingsSection>
    )
}
