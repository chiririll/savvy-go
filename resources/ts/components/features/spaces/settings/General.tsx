import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Boxes, LogOut, Trash2 } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useDeleteSpace, useLeaveSpace, useRenameSpace, useSpaceDetails } from '@/hooks'
import { formatBytes } from '@/lib/format'
import { Space } from '@/types/spaces'
import { Confirm } from '@/components/features/spaces/settings/Confirm'

export function General({ space, isAdmin }: { space: Space; isAdmin: boolean }) {
    const { t } = useTranslation('settings')
    const navigate = useNavigate()
    const { data: details } = useSpaceDetails(space.id)
    const [name, setName] = useState(space.name)
    const rename = useRenameSpace(space.id)
    const remove = useDeleteSpace(space.id)
    const leave = useLeaveSpace(space.id)

    return (
        <SettingsSection icon={Boxes} title={t('spaces.general.title')} description={t('spaces.general.description')}>
            <div className="space-y-4">
                <div className="space-y-1.5">
                    <Label htmlFor="space-name">{t('spaces.fields.name')}</Label>
                    <div className="flex gap-2">
                        <Input
                            id="space-name"
                            value={name}
                            disabled={!isAdmin}
                            maxLength={100}
                            onChange={(e) => setName(e.target.value)}
                        />
                        {isAdmin && name.trim() !== space.name && (
                            <Button
                                disabled={!name.trim() || rename.isPending}
                                onClick={() => rename.mutate(name.trim())}
                            >
                                {t('actions.save', { ns: 'common' })}
                            </Button>
                        )}
                    </div>
                </div>
                {details && (
                    <p className="text-sm text-muted-foreground">
                        {details.quota > 0
                            ? t('spaces.general.usageQuota', {
                                  size: formatBytes(details.size),
                                  quota: formatBytes(details.quota),
                              })
                            : t('spaces.general.usage', { size: formatBytes(details.size) })}
                    </p>
                )}
                <div className="flex flex-wrap gap-2">
                    <Confirm
                        trigger={
                            <Button variant="outline" size="sm">
                                <LogOut className="mr-2 size-4" />
                                {t('spaces.general.leave')}
                            </Button>
                        }
                        title={t('spaces.general.leaveTitle', { name: space.name })}
                        description={t('spaces.general.leaveDescription')}
                        action={t('spaces.general.leave')}
                        onConfirm={() => leave.mutate(undefined, { onSuccess: () => navigate('/') })}
                    />
                    {isAdmin && (
                        <Confirm
                            trigger={
                                <Button variant="destructive" size="sm">
                                    <Trash2 className="mr-2 size-4" />
                                    {t('spaces.general.delete')}
                                </Button>
                            }
                            title={t('spaces.general.deleteTitle', { name: space.name })}
                            description={t('spaces.general.deleteDescription')}
                            action={t('spaces.general.delete')}
                            onConfirm={() => remove.mutate(undefined, { onSuccess: () => navigate('/') })}
                        />
                    )}
                </div>
            </div>
        </SettingsSection>
    )
}
