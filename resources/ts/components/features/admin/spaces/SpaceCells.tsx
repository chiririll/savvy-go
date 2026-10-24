import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ShieldPlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useAssignSpaceAdmin, useSetSpaceQuota, useUsers } from '@/hooks'
import { SpaceOverview } from '@/types/spaces'

const MB = 1024 * 1024

export function QuotaCell({ space }: { space: SpaceOverview }) {
    const { t } = useTranslation('settings')
    const setQuota = useSetSpaceQuota()
    const [draft, setDraft] = useState(space.quota ? String(space.quota / MB) : '')
    const value = draft.trim() === '' ? null : Number(draft)
    const changed = (value ?? 0) * MB !== space.quota
    return (
        <div className="flex items-center gap-2">
            <Input
                className="h-8 w-24"
                aria-label={t('admin.spaces.quota')}
                type="number"
                min={0}
                step="any"
                placeholder={t('admin.spaces.defaultQuota')}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
            />
            <span className="text-xs text-muted-foreground">MB</span>
            {changed && (
                <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setQuota.mutate({ id: space.id, quotaBytes: value ? Math.round(value * MB) : null })}
                >
                    {t('actions.save', { ns: 'common' })}
                </Button>
            )}
        </div>
    )
}

export function AssignAdmin({ space }: { space: SpaceOverview }) {
    const { t } = useTranslation('settings')
    const { data: users = [] } = useUsers()
    const assign = useAssignSpaceAdmin()
    const [userId, setUserId] = useState('')
    return (
        <div className="flex items-center gap-2">
            <Select value={userId} onValueChange={setUserId}>
                <SelectTrigger className="h-8 w-44" aria-label={t('admin.spaces.chooseUser')}>
                    <SelectValue placeholder={t('admin.spaces.chooseUser')} />
                </SelectTrigger>
                <SelectContent>
                    {users
                        .filter((u) => u.role !== 'guest')
                        .map((u) => (
                            <SelectItem key={u.id} value={String(u.id)}>
                                {u.name}
                            </SelectItem>
                        ))}
                </SelectContent>
            </Select>
            <Button
                size="icon"
                variant="outline"
                disabled={!userId}
                title={t('admin.spaces.assign')}
                onClick={() =>
                    assign.mutate({ id: space.id, userId: Number(userId) }, { onSuccess: () => setUserId('') })
                }
            >
                <ShieldPlus className="size-4" />
            </Button>
        </div>
    )
}
