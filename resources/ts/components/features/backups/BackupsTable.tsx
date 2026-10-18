import { useTranslation } from 'react-i18next'
import { Download, RotateCcw, Trash2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatBytes, formatDateTime } from '@/lib/format'
import { backupStatusHelp, backupStatusLabel } from '@/lib/labels'
import type { Backup } from '@/types/backup'

function statusBadgeVariant(backup: Backup): 'secondary' | 'outline' | 'destructive' {
    if (backup.status === 'current') return 'secondary'
    return backup.restorable ? 'outline' : 'destructive'
}

function SkeletonRows() {
    return Array.from({ length: 3 }).map((_, i) => (
        <TableRow key={i}>
            <TableCell>
                <Skeleton className="h-4 w-32" />
            </TableCell>
            <TableCell>
                <Skeleton className="h-4 w-16" />
            </TableCell>
            <TableCell>
                <Skeleton className="h-4 w-28" />
            </TableCell>
            <TableCell>
                <Skeleton className="h-4 w-24" />
            </TableCell>
            <TableCell>
                <Skeleton className="h-4 w-24 ml-auto" />
            </TableCell>
        </TableRow>
    ))
}

interface BackupRowProps {
    backup: Backup
    onDownload: (backup: Backup) => void
    onRestore: (backup: Backup) => void
    onDelete: (backup: Backup) => void
}

function BackupRow({ backup, onDownload, onRestore, onDelete }: BackupRowProps) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')

    return (
        <TableRow>
            <TableCell>{formatDateTime(backup.createdAt)}</TableCell>
            <TableCell>{formatBytes(backup.size)}</TableCell>
            <TableCell>
                <div className="flex flex-col items-start gap-1">
                    <span className="font-mono text-sm">{backup.appVersion || t('backups.versionUnknown')}</span>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Badge variant={statusBadgeVariant(backup)}>{backupStatusLabel(t, backup.status)}</Badge>
                        </TooltipTrigger>
                        <TooltipContent>{backupStatusHelp(t, backup.status)}</TooltipContent>
                    </Tooltip>
                </div>
            </TableCell>
            <TableCell className="text-muted-foreground">{backup.note || '-'}</TableCell>
            <TableCell className="text-right">
                <div className="flex justify-end gap-1">
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDownload(backup)}
                        title={t('backups.download')}
                    >
                        <Download className="size-4" />
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onRestore(backup)}
                        title={backup.restorable ? t('backups.restore') : backupStatusHelp(t, backup.status)}
                        disabled={!backup.restorable}
                    >
                        <RotateCcw className="size-4" />
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDelete(backup)}
                        title={tCommon('actions.delete')}
                    >
                        <Trash2 className="size-4" />
                    </Button>
                </div>
            </TableCell>
        </TableRow>
    )
}

interface BackupsTableProps extends Omit<BackupRowProps, 'backup'> {
    backups?: Backup[]
    isLoading: boolean
}

export function BackupsTable({ backups, isLoading, ...actions }: BackupsTableProps) {
    const { t } = useTranslation('settings')

    return (
        <div className="border rounded-lg">
            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead>{t('backups.date')}</TableHead>
                        <TableHead>{t('backups.size')}</TableHead>
                        <TableHead>{t('backups.version')}</TableHead>
                        <TableHead>{t('backups.note')}</TableHead>
                        <TableHead className="text-right">{t('backups.actions')}</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {isLoading ? (
                        <SkeletonRows />
                    ) : backups?.length === 0 ? (
                        <TableRow>
                            <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                                {t('backups.empty')}
                            </TableCell>
                        </TableRow>
                    ) : (
                        backups?.map((backup) => <BackupRow key={backup.filename} backup={backup} {...actions} />)
                    )}
                </TableBody>
            </Table>
        </div>
    )
}
