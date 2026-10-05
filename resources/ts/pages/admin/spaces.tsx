import { testId } from '@/lib/test-id'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArchiveRestore, ShieldPlus, Trash2 } from 'lucide-react'
import { Page, PageHeader, SettingsSection } from '@/components/shared'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import {
    useAdminDeleteSpace,
    useAdminSpaces,
    useAssignSpaceAdmin,
    useDeletedSpaces,
    useRestoreDeletedSpace,
    useSetSpaceQuota,
    useUsers,
} from '@/hooks'
import { formatBytes, formatDateTime } from '@/lib/format'
import { SpaceOverview } from '@/types/spaces'

const MB = 1024 * 1024

function QuotaCell({ space }: { space: SpaceOverview }) {
    const { t } = useTranslation('settings')
    const setQuota = useSetSpaceQuota()
    const [draft, setDraft] = useState(space.quota ? String(space.quota / MB) : '')
    const value = draft.trim() === '' ? null : Number(draft)
    const changed = (value ?? 0) * MB !== space.quota
    return (
        <div className="flex items-center gap-2">
            <Input className="h-8 w-24" type="number" min={0} step="any" placeholder={t('admin.spaces.defaultQuota')} value={draft} onChange={(e) => setDraft(e.target.value)} />
            <span className="text-xs text-muted-foreground">MB</span>
            {changed && (
                <Button size="sm" variant="outline" onClick={() => setQuota.mutate({ id: space.id, quotaBytes: value ? Math.round(value * MB) : null })}>
                    {t('actions.save', { ns: 'common' })}
                </Button>
            )}
        </div>
    )
}

function AssignAdmin({ space }: { space: SpaceOverview }) {
    const { t } = useTranslation('settings')
    const { data: users = [] } = useUsers()
    const assign = useAssignSpaceAdmin()
    const [userId, setUserId] = useState('')
    return (
        <div className="flex items-center gap-2">
            <Select value={userId} onValueChange={setUserId}>
                <SelectTrigger className="h-8 w-44">
                    <SelectValue placeholder={t('admin.spaces.chooseUser')} />
                </SelectTrigger>
                <SelectContent>
                    {users.filter((u) => u.role !== 'guest').map((u) => (
                        <SelectItem key={u.id} value={String(u.id)}>{u.name}</SelectItem>
                    ))}
                </SelectContent>
            </Select>
            <Button size="icon" variant="outline" disabled={!userId} title={t('admin.spaces.assign')}
                onClick={() => assign.mutate({ id: space.id, userId: Number(userId) }, { onSuccess: () => setUserId('') })}>
                <ShieldPlus className="size-4" />
            </Button>
        </div>
    )
}

export default function AdminSpacesPage() {
    const { t } = useTranslation('settings')
    const { data: spaces = [] } = useAdminSpaces()
    const { data: deleted = [] } = useDeletedSpaces()
    const remove = useAdminDeleteSpace()
    const restore = useRestoreDeletedSpace()

    return (
        <Page title={t('admin.spaces.title')}>
            <PageHeader title={t('admin.spaces.title')} description={t('admin.spaces.description')} />

            <div className="space-y-6">
                <div className="rounded-lg border">
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>{t('spaces.fields.name')}</TableHead>
                                <TableHead>{t('admin.spaces.members')}</TableHead>
                                <TableHead>{t('admin.spaces.size')}</TableHead>
                                <TableHead>{t('admin.spaces.quota')}</TableHead>
                                <TableHead>{t('admin.spaces.assign')}</TableHead>
                                <TableHead />
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {spaces.map((s) => (
                                <TableRow key={s.id}>
                                    <TableCell>
                                        <div className="font-medium">{s.name}</div>
                                        <div className="text-xs text-muted-foreground">{formatDateTime(s.createdAt)}</div>
                                        {s.unavailable && <Badge variant="destructive" className="mt-1">{s.unavailable}</Badge>}
                                    </TableCell>
                                    <TableCell>{t('admin.spaces.membersCount', { members: s.members, admins: s.admins })}</TableCell>
                                    <TableCell>{formatBytes(s.size)}</TableCell>
                                    <TableCell><QuotaCell space={s} /></TableCell>
                                    <TableCell><AssignAdmin space={s} /></TableCell>
                                    <TableCell className="text-right">
                                        <AlertDialog>
                                            <AlertDialogTrigger asChild {...testId('confirm')}>
                                                <Button variant="ghost" size="icon" title={t('actions.delete', { ns: 'common' })}>
                                                    <Trash2 className="size-4" />
                                                </Button>
                                            </AlertDialogTrigger>
                                            <AlertDialogContent>
                                                <AlertDialogHeader>
                                                    <AlertDialogTitle>{t('spaces.general.deleteTitle', { name: s.name })}</AlertDialogTitle>
                                                    <AlertDialogDescription>{t('spaces.general.deleteDescription')}</AlertDialogDescription>
                                                </AlertDialogHeader>
                                                <AlertDialogFooter>
                                                    <AlertDialogCancel>{t('actions.cancel', { ns: 'common' })}</AlertDialogCancel>
                                                    <AlertDialogAction onClick={() => remove.mutate(s.id)}>{t('actions.delete', { ns: 'common' })}</AlertDialogAction>
                                                </AlertDialogFooter>
                                            </AlertDialogContent>
                                        </AlertDialog>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </div>
                <p className="text-xs text-muted-foreground">{t('admin.spaces.privacy')}</p>

                <SettingsSection icon={ArchiveRestore} title={t('admin.spaces.deletedTitle')} description={t('admin.spaces.deletedDescription')}>
                    {deleted.length ? (
                        <ul className="divide-y rounded-lg border text-sm">
                            {deleted.map((b) => (
                                <li key={b.filename} className="flex items-center justify-between gap-3 px-3 py-2">
                                    <div className="min-w-0">
                                        <p className="truncate font-medium">{b.spaceName || b.filename}</p>
                                        <p className="text-xs text-muted-foreground">{formatDateTime(b.createdAt)} · {formatBytes(b.size)}</p>
                                    </div>
                                    <Button size="sm" variant="outline" disabled={restore.isPending} onClick={() => restore.mutate(b.filename)}>
                                        {t('admin.spaces.restore')}
                                    </Button>
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <p className="text-sm text-muted-foreground">{t('admin.spaces.noDeleted')}</p>
                    )}
                </SettingsSection>
            </div>
        </Page>
    )
}
