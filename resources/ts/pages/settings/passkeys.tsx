import { useTranslation } from 'react-i18next'
import { Page, PageHeader, FormWrapper } from '@/components/shared'
import { PasskeysSection } from '@/components/features/security/PasskeysSection'

export default function PasskeySettingsPage() {
    const { t } = useTranslation('settings')

    return (
        <Page title={t('security.passkeys.title')}>
            <PageHeader title={t('security.passkeys.title')} description={t('security.passkeys.description')} />

            <FormWrapper>
                <PasskeysSection />
            </FormWrapper>
        </Page>
    )
}
