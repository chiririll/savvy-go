import { useTranslation } from 'react-i18next'
import { Page, PageHeader, FormWrapper } from '@/components/shared'
import { UserSettingsTabs } from '@/components/features/settings/UserSettingsTabs'
import { ChangePasswordSection } from '@/components/features/security/ChangePasswordSection'
import { SessionsSection } from '@/components/features/security/SessionsSection'
import { TwoFactorSection, RecoverySection } from '@/components/features/security/TwoFactorSections'
import { TwoFactorDialogs } from '@/components/features/security/TwoFactorDialogs'
import { useTwoFactorSetup } from '@/components/features/security/useTwoFactorSetup'

export default function SecuritySettingsPage() {
    const { t } = useTranslation('settings')
    const twoFactor = useTwoFactorSetup()

    return (
        <Page title={t('security.title')}>
            <PageHeader title={t('security.heading')} description={t('security.description')} />
            <UserSettingsTabs />

            <FormWrapper>
                <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-2">
                    <ChangePasswordSection />
                    <SessionsSection />
                    <TwoFactorSection setup={twoFactor} />
                    <RecoverySection setup={twoFactor} />
                </div>
            </FormWrapper>

            <TwoFactorDialogs setup={twoFactor} />
        </Page>
    )
}
