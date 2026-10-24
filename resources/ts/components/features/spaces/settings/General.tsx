import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Boxes, LogOut, Trash2 } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useDeleteSpace, useLeaveSpace, useUpdateSpace, useSpaceDetails } from '@/hooks'
import { formatBytes } from '@/lib/format'
import { Space } from '@/types/spaces'
import { SpaceLookFields } from '@/components/features/spaces/SpaceLookFields'
import { Confirm } from '@/components/shared/dialogs/Confirm'

export function General({ space, isAdmin }: { space: Space; isAdmin: boolean }) {
    const { t } = useTranslation('settings')
    const navigate = useNavigate()
    const { data: details } = useSpaceDetails(space.id)
    const [name, setName] = useState(space.name)
    const [look, setLook] = useState({ emoji: space.emoji ?? '', color: space.color ?? '' })
    const update = useUpdateSpace(space.id)
    const changed =
        name.trim() !== space.name || look.emoji !== (space.emoji ?? '') || look.color !== (space.color ?? '')
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
                    </div>
                </div>
                <SpaceLookFields {...look} name={name}onChange={setLook} disabled={!isAdmin} />
                {isAdmin && changed && (
                    <Button
                        disabled={!name.trim() || update.isPending}
                        onClick={() => update.mutate({ name: name.trim(), ...look })}
                    >
                        {t('actions.save', { ns: 'common' })}
                    </Button>
                )}
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
