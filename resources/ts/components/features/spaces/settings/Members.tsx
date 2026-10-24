import { useTranslation } from 'react-i18next'
import { Trash2, UserRound, Users } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useRemoveMember, useSetMemberRole, useSpaceMembers } from '@/hooks'
import { useUser } from '@/stores/auth'
import { Space, SpaceRole } from '@/types/spaces'
import { spaceRoleLabel } from '@/lib/labels'

const ROLES: SpaceRole[] = ['admin', 'editor', 'viewer']

export function Members({ space, isAdmin }: { space: Space; isAdmin: boolean }) {
    const { t } = useTranslation('settings')
    const me = useUser()
    const { data: members = [] } = useSpaceMembers(space.id)
    const setRole = useSetMemberRole(space.id)
    const remove = useRemoveMember(space.id)

    return (
        <SettingsSection icon={Users} title={t('spaces.members.title')} description={t('spaces.members.description')}>
            <ul className="divide-y rounded-lg border">
                {members.map((m) => (
                    <li key={m.userId} className="flex flex-wrap items-center gap-x-3 gap-y-2 px-3 py-2">
                        <div
                            aria-hidden
                            className="flex size-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground"
                        >
                            <UserRound className="size-4" />
                        </div>
                        <div className="min-w-0 flex-1 basis-40">
                            <p className="truncate text-sm font-medium">
                                {m.name}{' '}
                                {m.userId === me?.id && (
                                    <span className="text-muted-foreground">({t('spaces.members.you')})</span>
                                )}
                            </p>
                            <p className="truncate text-xs text-muted-foreground">{m.email}</p>
                        </div>
                        <div className="ml-auto flex items-center gap-2">
                            <div className="flex w-32 justify-end">
                                {isAdmin ? (
                                    <Select
                                        value={m.role}
                                        onValueChange={(role) =>
                                            setRole.mutate({ userId: m.userId, role: role as SpaceRole })
                                        }
                                    >
                                        <SelectTrigger
                                            className="h-8 w-32"
                                            aria-label={`${t('common:fields.role')}: ${m.name}`}
                                        >
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {ROLES.map((r) => (
                                                <SelectItem key={r} value={r}>
                                                    {spaceRoleLabel(t, r)}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                ) : (
                                    <Badge variant="secondary">{spaceRoleLabel(t, m.role)}</Badge>
                                )}
                            </div>
                            {isAdmin &&
                                (m.userId !== me?.id ? (
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        title={t('spaces.members.remove')}
                                        onClick={() => remove.mutate(m.userId)}
                                    >
                                        <Trash2 className="size-4" />
                                    </Button>
                                ) : (
                                    <span aria-hidden data-testid="member-action-placeholder" className="size-9" />
                                ))}
                        </div>
                    </li>
                ))}
            </ul>
        </SettingsSection>
    )
}
