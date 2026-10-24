import { useTranslation } from 'react-i18next'
import { Page, PageHeader, FormWrapper } from '@/components/shared'
import { MailSection } from '@/components/features/mail'
import { ClearInvitationsSection } from '@/components/features/spaces/ClearInvitationsSection'
import { AuthenticationSection } from '@/components/features/admin/system/AuthenticationSection'
import { ServerSection, SpacesLimitsSection } from '@/components/features/admin/system/ServerAndSpacesSections'

export default function AdminSystemPage() {
    const { t } = useTranslation('settings')

    return (
        <Page title={t('system.title')}>
            <PageHeader title={t('system.heading')} description={t('system.description')} />

            <FormWrapper>
                <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-2">
                    <AuthenticationSection />
                    <ServerSection />
                    <MailSection />
                    <ClearInvitationsSection />
                    <SpacesLimitsSection />
                </div>
            </FormWrapper>
        </Page>
    )
}
