import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useAdminSpaces } from '@/hooks'
import { formatBytes, formatDateTime } from '@/lib/format'
import { QuotaCell, AssignAdmin } from '@/components/features/admin/spaces/SpaceCells'
import { DeleteSpaceButton } from '@/components/features/admin/spaces/DeleteSpaceButton'
import { DeletedSpacesSection } from '@/components/features/admin/spaces/DeletedSpacesSection'

export default function AdminSpacesPage() {
    const { t } = useTranslation('settings')
    const { data: spaces = [] } = useAdminSpaces()

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
                                        <div className="text-xs text-muted-foreground">
                                            {formatDateTime(s.createdAt)}
                                        </div>
                                        {s.unavailable && (
                                            <Badge variant="destructive" className="mt-1">
                                                {s.unavailable}
                                            </Badge>
                                        )}
                                    </TableCell>
                                    <TableCell>
                                        {t('admin.spaces.membersCount', { members: s.members, admins: s.admins })}
                                    </TableCell>
                                    <TableCell>{formatBytes(s.size)}</TableCell>
                                    <TableCell>
                                        <QuotaCell space={s} />
                                    </TableCell>
                                    <TableCell>
                                        <AssignAdmin space={s} />
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <DeleteSpaceButton space={s} />
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </div>
                <p className="text-xs text-muted-foreground">{t('admin.spaces.privacy')}</p>

                <DeletedSpacesSection />
            </div>
        </Page>
    )
}
