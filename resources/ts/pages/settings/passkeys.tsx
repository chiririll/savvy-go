import { useTranslation } from 'react-i18next'
import { Navigate } from 'react-router-dom'
import { browserSupportsWebAuthn } from '@simplewebauthn/browser'
import { Page, PageHeader, FormWrapper } from '@/components/shared'
import { UserSettingsTabs } from '@/components/features/settings/UserSettingsTabs'
import { PasskeysSection } from '@/components/features/security/PasskeysSection'
import { pages } from '@/app/pages'

export default function PasskeySettingsPage() {
    const { t } = useTranslation('settings')
    if (!browserSupportsWebAuthn()) return <Navigate to={pages.securitySettings.path} replace />

    return (
        <Page title={t('security.passkeys.title')}>
            <PageHeader title={t('security.passkeys.title')} description={t('security.passkeys.description')} />
            <UserSettingsTabs />

            <FormWrapper>
                <PasskeysSection />
            </FormWrapper>
        </Page>
    )
}
