import { useTranslation } from 'react-i18next'
import { Mail } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { InviteForm } from '@/components/features/spaces/InviteForm'
import { Button } from '@/components/ui/button'
import { useRevokeInvitation, useSpaceInvitations } from '@/hooks'
import { formatDateTime } from '@/lib/format'
import { Space } from '@/types/spaces'
import { spaceRoleLabel } from '@/lib/labels'

export function Invitations({ space }: { space: Space }) {
    const { t } = useTranslation('settings')
    const { data: invitations = [] } = useSpaceInvitations(space.id)
    const revoke = useRevokeInvitation(space.id)
    const open = invitations.filter((i) => !i.acceptedAt && (!i.expiresAt || new Date(i.expiresAt) > new Date()))

    return (
        <SettingsSection
            icon={Mail}
            title={t('spaces.invitations.title')}
            description={t('spaces.invitations.description')}
        >
            <div className="space-y-4">
                <InviteForm spaceId={space.id} />
                {open.length > 0 && (
                    <ul className="divide-y rounded-lg border">
                        {open.map((i) => (
                            <li key={i.id} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                                <div className="min-w-0">
                                    <p className="truncate">
                                        {i.email ?? i.note ?? t('spaces.invitations.anyone')} ·{' '}
                                        {spaceRoleLabel(t, i.role)}
                                    </p>
                                    <p className="truncate text-xs text-muted-foreground">
                                        {t('spaces.invitations.expires', { date: formatDateTime(i.expiresAt) })}
                                    </p>
                                </div>
                                <Button variant="ghost" size="sm" onClick={() => revoke.mutate(i.id)}>
                                    {t('spaces.invitations.revoke')}
                                </Button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </SettingsSection>
    )
}
