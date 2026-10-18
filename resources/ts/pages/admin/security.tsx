import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { SigningKeys } from '@/components/features/admin/security/SigningKeys'
import { McpAccess } from '@/components/features/admin/security/McpAccess'

/** How this server signs backups and transfers, and whose signatures it trusts. */
export default function AdminSecurityPage() {
    const { t } = useTranslation('settings')
    return (
        <Page title={t('admin.security.title')}>
            <PageHeader title={t('admin.security.title')} description={t('admin.security.description')} />
            <div className="max-w-3xl space-y-6">
                <McpAccess />
                <SigningKeys />
            </div>
        </Page>
    )
}
