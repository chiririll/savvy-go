import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Download, RotateCcw, Trash2, Plus, Upload, Loader2 } from 'lucide-react'
import { useBackups, useCreateBackup, useUploadBackup, useRestoreBackup, useDeleteBackup } from '@/hooks/use-backups'
import { backupsApi } from '@/api/backups'
import { Backup } from '@/types/backup'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { intlLocale } from '@/lib/i18n'

function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString(intlLocale())
}

function statusBadgeVariant(backup: Backup): 'secondary' | 'outline' | 'destructive' {
    if (backup.status === 'current') return 'secondary'
    return backup.restorable ? 'outline' : 'destructive'
}

export default function BackupsPage() {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const isReadOnly = useReadOnly()
    const { data: backups, isLoading } = useBackups()
    const createBackup = useCreateBackup()
    const uploadBackup = useUploadBackup()
    const restoreBackup = useRestoreBackup()
    const deleteBackup = useDeleteBackup()

    const [createDialogOpen, setCreateDialogOpen] = useState(false)
    const [uploadDialogOpen, setUploadDialogOpen] = useState(false)
    const [restoreDialogOpen, setRestoreDialogOpen] = useState(false)
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
    const [selectedBackup, setSelectedBackup] = useState<Backup | null>(null)
    const [note, setNote] = useState('')
    const [uploadFile, setUploadFile] = useState<File | null>(null)

    const handleCreate = () => {
        createBackup.mutate(note || undefined, {
            onSuccess: () => {
                setCreateDialogOpen(false)
                setNote('')
            },
        })
    }

    const handleUpload = () => {
        if (!uploadFile) return
        uploadBackup.mutate({ file: uploadFile, note: note || undefined }, {
            onSuccess: () => {
                setUploadDialogOpen(false)
                setNote('')
                setUploadFile(null)
            },
        })
    }

    const openRestore = (backup: Backup) => {
        setSelectedBackup(backup)
        setRestoreDialogOpen(true)
    }

    const closeRestore = () => {
        setRestoreDialogOpen(false)
        setSelectedBackup(null)
    }

    const handleRestore = () => {
        if (!selectedBackup?.restorable) return
        restoreBackup.mutate(selectedBackup.filename, {
            onSuccess: closeRestore,
        })
    }

    const handleDelete = () => {
        if (!selectedBackup) return
        deleteBackup.mutate(selectedBackup.filename, {
            onSuccess: () => {
                setDeleteDialogOpen(false)
                setSelectedBackup(null)
            },
        })
    }

    const handleDownload = (backup: Backup) => {
        const token = localStorage.getItem('token')
        const url = backupsApi.download(backup.filename)

        fetch(url, {
            headers: { Authorization: `Bearer ${token}` }
        })
            .then(res => res.blob())
            .then(blob => {
                const a = document.createElement('a')
                a.href = URL.createObjectURL(blob)
                a.download = backup.filename
                a.click()
                URL.revokeObjectURL(a.href)
            })
    }

    return (
        <Page title={t('backups.title')}>
            <PageHeader
                title={t('backups.title')}
                description={t('backups.description')}
            />

            <div className="flex gap-2 mb-6">
                <Button onClick={() => setCreateDialogOpen(true)} disabled={isReadOnly}>
                    <Plus className="size-4 mr-2" />
                    {t('backups.create')}
                </Button>
                <Button variant="outline" onClick={() => setUploadDialogOpen(true)} disabled={isReadOnly}>
                    <Upload className="size-4 mr-2" />
                    {t('backups.upload')}
                </Button>
            </div>

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
                            Array.from({ length: 3 }).map((_, i) => (
                                <TableRow key={i}>
                                    <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                                    <TableCell><Skeleton className="h-4 w-24 ml-auto" /></TableCell>
                                </TableRow>
                            ))
                        ) : backups?.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                                    {t('backups.empty')}
                                </TableCell>
                            </TableRow>
                        ) : (
                            backups?.map((backup) => (
                                <TableRow key={backup.filename}>
                                    <TableCell>{formatDate(backup.createdAt)}</TableCell>
                                    <TableCell>{formatBytes(backup.size)}</TableCell>
                                    <TableCell>
                                        <div className="flex flex-col items-start gap-1">
                                            <span className="font-mono text-sm">
                                                {backup.appVersion
                                                    || (backup.status === 'raw'
                                                        ? t('backups.versionRaw')
                                                        : t('backups.versionUnknown'))}
                                            </span>
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <Badge variant={statusBadgeVariant(backup)}>
                                                        {t(`backups.status.${backup.status}`)}
                                                    </Badge>
                                                </TooltipTrigger>
                                                <TooltipContent>
                                                    {t(`backups.statusHelp.${backup.status}`)}
                                                </TooltipContent>
                                            </Tooltip>
                                        </div>
                                    </TableCell>
                                    <TableCell className="text-muted-foreground">
                                        {backup.note || '-'}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-1">
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                onClick={() => handleDownload(backup)}
                                                title={t('backups.download')}
                                            >
                                                <Download className="size-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                onClick={() => openRestore(backup)}
                                                title={backup.restorable
                                                    ? t('backups.restore')
                                                    : t(`backups.statusHelp.${backup.status}`)}
                                                disabled={isReadOnly || !backup.restorable}
                                            >
                                                <RotateCcw className="size-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                onClick={() => {
                                                    setSelectedBackup(backup)
                                                    setDeleteDialogOpen(true)
                                                }}
                                                title={tCommon('actions.delete')}
                                                disabled={isReadOnly}
                                            >
                                                <Trash2 className="size-4" />
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('backups.createTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('backups.createDescription')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="note">{t('backups.noteOptional')}</Label>
                            <Input
                                id="note"
                                placeholder={t('backups.notePlaceholder')}
                                value={note}
                                onChange={(e) => setNote(e.target.value)}
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button onClick={handleCreate} disabled={createBackup.isPending}>
                            {createBackup.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
                            {tCommon('actions.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('backups.uploadTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('backups.uploadDescription')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="file">{t('backups.backupFile')}</Label>
                            <Input
                                id="file"
                                type="file"
                                accept=".zip,.sqlite"
                                onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
                            />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="upload-note">{t('backups.noteOptional')}</Label>
                            <Input
                                id="upload-note"
                                placeholder={t('backups.uploadNotePlaceholder')}
                                value={note}
                                onChange={(e) => setNote(e.target.value)}
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setUploadDialogOpen(false)}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button onClick={handleUpload} disabled={!uploadFile || uploadBackup.isPending}>
                            {uploadBackup.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
                            {t('backups.upload')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <AlertDialog
                open={restoreDialogOpen}
                onOpenChange={(open) => {
                    if (!open) {
                        closeRestore()
                        return
                    }
                    setRestoreDialogOpen(true)
                }}
            >
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('backups.restoreTitle')}</AlertDialogTitle>
                        <AlertDialogDescription asChild>
                            <div className="space-y-2">
                                <p>
                                    {t('backups.restoreDescription', {
                                        date: selectedBackup ? formatDate(selectedBackup.createdAt) : '',
                                    })}
                                </p>
                                {selectedBackup && selectedBackup.status !== 'current' && (
                                    <p>
                                        {t(`backups.statusHelp.${selectedBackup.status}`)}
                                    </p>
                                )}
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

            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('backups.deleteTitle')}</AlertDialogTitle>
                        <AlertDialogDescription>
                            {t('backups.deleteDescription', {
                                date: selectedBackup ? formatDate(selectedBackup.createdAt) : '',
                            })}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>{tCommon('actions.cancel')}</AlertDialogCancel>
                        <AlertDialogAction
                            variant="destructive"
                            onClick={handleDelete}
                            disabled={deleteBackup.isPending}
                        >
                            {deleteBackup.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
                            {tCommon('actions.delete')}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </Page>
    )
}
