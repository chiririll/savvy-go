import { Boxes, Globe } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SettingsSection, ToggleRow, SettingsRowSkeleton } from '@/components/shared'
import { useSettings, useUpdateSettings } from '@/hooks'
import { NumberRow, TextRow } from './SettingRows'

export function ServerSection() {
    const { t } = useTranslation('settings')
    const { data: settings, isLoading } = useSettings()
    const updateSettings = useUpdateSettings()

    return (
        <SettingsSection icon={Globe} title={t('system.server.title')} description={t('system.server.description')}>
            <div className="divide-y rounded-lg border">
                {isLoading || !settings ? (
                    <SettingsRowSkeleton />
                ) : (
                    <TextRow
                        id="app-url"
                        label={t('system.server.appUrl')}
                        description={
                            settings.config_writable === false
                                ? t('system.server.configReadOnly')
                                : t('system.server.appUrlDescription')
                        }
                        readOnly={settings.config_writable === false}
                        placeholder={t('system.server.appUrlPlaceholder')}
                        suggestion={window.location.origin}
                        value={settings.app_url ?? ''}
                        onSave={(v) => updateSettings.mutate({ app_url: v })}
                    />
                )}
            </div>
        </SettingsSection>
    )
}

export function SpacesLimitsSection() {
    const { t } = useTranslation('settings')
    const { data: settings, isLoading } = useSettings()
    const updateSettings = useUpdateSettings()

    return (
        <SettingsSection icon={Boxes} title={t('system.spaces.title')} description={t('system.spaces.description')}>
            <div className="divide-y rounded-lg border">
                {isLoading || !settings ? (
                    <>
                        <SettingsRowSkeleton />
                        <SettingsRowSkeleton />
                        <SettingsRowSkeleton />
                    </>
                ) : (
                    <>
                        <ToggleRow
                            id="invites-register"
                            label={t('system.spaces.invitesRegister')}
                            description={t('system.spaces.invitesRegisterDescription')}
                            checked={settings.space_invites_can_register}
                            disabled={updateSettings.isPending}
                            onChange={(checked) => updateSettings.mutate({ space_invites_can_register: checked })}
                        />
                        <NumberRow
                            id="max-spaces"
                            label={t('system.spaces.maxSpaces')}
                            description={t('system.spaces.maxSpacesDescription')}
                            value={settings.max_spaces_per_user}
                            nullable
                            onSave={(v) => updateSettings.mutate({ max_spaces_per_user: v })}
                        />
                        <NumberRow
                            id="quota"
                            label={t('system.spaces.quota')}
                            description={t('system.spaces.quotaDescription')}
                            value={settings.space_quota_mb}
                            onSave={(v) => updateSettings.mutate({ space_quota_mb: v ?? 0 })}
                        />
                        <NumberRow
                            id="backups-max"
                            label={t('system.spaces.backupsMax')}
                            description={t('system.spaces.backupsMaxDescription')}
                            value={settings.space_backups_max}
                            onSave={(v) => updateSettings.mutate({ space_backups_max: v ?? 0 })}
                        />
                    </>
                )}
            </div>
        </SettingsSection>
    )
}
