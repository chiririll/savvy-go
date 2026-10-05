import { useTranslation } from 'react-i18next'
import { Languages, Palette } from 'lucide-react'
import { LanguageSwitcher, Page, PageHeader, SettingsSection, ThemeSwitcher } from '@/components/shared'
import { UserSettingsTabs } from '@/components/features/spaces/UserSettingsTabs'

export default function UserSettingsPage() {
    const { t } = useTranslation('settings')

    return (
        <Page title={t('user.title')}>
            <PageHeader title={t('user.title')} description={t('user.description')} />
            <UserSettingsTabs />
            <div className="grid items-start gap-6 lg:grid-cols-2">
                <SettingsSection icon={Languages} title={t('system.language.title')} description={t('system.language.description')}>
                    <div className="rounded-lg border">
                        <LanguageSwitcher variant="select" />
                    </div>
                </SettingsSection>
                <SettingsSection icon={Palette} title={t('user.theme.title')} description={t('user.theme.description')}>
                    <ThemeSwitcher />
                </SettingsSection>
            </div>
        </Page>
    )
}
