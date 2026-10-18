import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { TransfersNeedingAttention } from '@/components/features/spaces/TransfersNeedingAttention'
import { SyncGroupSection } from '@/components/features/spaces/SyncGroupSection'
import { useCurrentSpace } from '@/hooks'
import { SpaceBadge } from '@/components/features/spaces/SpaceBadge'
import { spaceRoleLabel } from '@/lib/labels'
import { General } from '@/components/features/spaces/settings/General'
import { Members } from '@/components/features/spaces/settings/Members'
import { Invitations } from '@/components/features/spaces/settings/Invitations'
import { Links } from '@/components/features/spaces/settings/Links'
import { Audit } from '@/components/features/spaces/settings/Audit'

export default function SpaceSettingsPage() {
    const { t } = useTranslation('settings')
    const space = useCurrentSpace()
    if (!space) return <></>
    const isAdmin = space.role === 'admin'

    return (
        <Page title={t('spaces.title')}>
            <PageHeader
                leading={<SpaceBadge space={space} />}
                title={space.name}
                description={t('spaces.description', { role: spaceRoleLabel(t, space.role) })}
            />
            <div className="grid items-start gap-6 lg:grid-cols-2">
                <div className="space-y-6">
                    <General key={space.id} space={space} isAdmin={isAdmin} />
                    <Members space={space} isAdmin={isAdmin} />
                    {isAdmin && <Invitations space={space} />}
                </div>
                <div className="space-y-6">
                    <TransfersNeedingAttention space={space} />
                    <Links space={space} isAdmin={isAdmin} />
                    <SyncGroupSection space={space} isAdmin={isAdmin} />
                    {isAdmin && <Audit space={space} />}
                </div>
            </div>
        </Page>
    )
}
