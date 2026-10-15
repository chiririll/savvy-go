import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { ServerChecks } from '@/components/features/admin/ServerChecks'

export default function AdminChecksPage() {
    const { t } = useTranslation('settings')

    return (
        <Page title={t('checks.pageTitle')}>
            <PageHeader title={t('checks.pageTitle')} description={t('checks.description')} />
            <ServerChecks />
        </Page>
    )
}
