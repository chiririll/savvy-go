import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Boxes, ShieldCheck, Lock } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Page, PageHeader, FormWrapper, SettingsSection, ToggleRow, SettingsRowSkeleton } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useSettings, useUpdateSettings, useSsoProviders } from '@/hooks'
import { Settings } from '@/types'

/** A number field saved on blur; empty means null when nullable. */
function NumberRow({
    id,
    label,
    description,
    value,
    nullable,
    onSave,
}: {
    id: string
    label: string
    description: string
    value: number | null
    nullable?: boolean
    onSave: (value: number | null) => void
}) {
    const { t } = useTranslation('settings')
    const [draft, setDraft] = useState(value === null ? '' : String(value))
    useEffect(() => setDraft(value === null ? '' : String(value)), [value])
    const parsed = draft.trim() === '' ? null : Number(draft)
    const valid = (parsed === null && nullable) || (parsed !== null && Number.isFinite(parsed) && parsed >= 0)
    const changed = parsed !== value

    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="space-y-1">
                <Label htmlFor={id} className="text-sm font-medium">{label}</Label>
                <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
            </div>
            <div className="flex items-center gap-2">
                <Input id={id} type="number" min={0} step="any" className="w-28" value={draft} onChange={(e) => setDraft(e.target.value)} />
                {changed && (
                    <Button size="sm" disabled={!valid} onClick={() => onSave(parsed)}>
                        {t('actions.save', { ns: 'common' })}
                    </Button>
                )}
            </div>
        </div>
    )
}

export default function AdminSystemPage() {
    const { t } = useTranslation('settings')
    const { data: settings, isLoading } = useSettings()
    const { data: ssoProviders } = useSsoProviders()
    const updateSettings = useUpdateSettings()

    const hasSso = (ssoProviders?.length ?? 0) > 0
    const passwordLoginEnabled = settings?.password_login_enabled ?? true
    const passwordLoginLocked = passwordLoginEnabled && !hasSso
    const ssoAllowSignup = settings?.sso_allow_signup ?? true
    const requireVerifiedEmail = settings?.sso_require_verified_email ?? false

    const save = (data: Partial<Settings>) => updateSettings.mutate(data)

    const handlePasswordLoginChange = (checked: boolean) => {
        if (!checked && !hasSso) {
            toast.error(t('system.authentication.passwordLoginError'))
            return
        }
        save({ password_login_enabled: checked })
    }

    return (
        <Page title={t('system.title')}>
            <PageHeader title={t('system.heading')} description={t('system.description')} />

            <FormWrapper>
                <div className="grid items-start gap-6 lg:grid-cols-2">
                    <SettingsSection
                        icon={ShieldCheck}
                        title={t('system.authentication.title')}
                        description={t('system.authentication.description')}
                    >
                        <div className="divide-y rounded-lg border">
                            {isLoading ? (
                                <>
                                    <SettingsRowSkeleton />
                                    <SettingsRowSkeleton />
                                    <SettingsRowSkeleton />
                                </>
                            ) : (
                                <>
                                    <ToggleRow
                                        id="password-login"
                                        label={t('system.authentication.passwordLogin')}
                                        description={
                                            passwordLoginLocked
                                                ? t('system.authentication.passwordLoginLocked')
                                                : t('system.authentication.passwordLoginOpen')
                                        }
                                        checked={passwordLoginEnabled}
                                        disabled={updateSettings.isPending || passwordLoginLocked}
                                        badge={
                                            passwordLoginLocked ? (
                                                <span className="inline-flex items-center gap-1 rounded-full border bg-muted px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                                                    <Lock className="size-2.5" />
                                                    {t('system.authentication.locked')}
                                                </span>
                                            ) : undefined
                                        }
                                        onChange={handlePasswordLoginChange}
                                    />
                                    <ToggleRow
                                        id="sso-signup"
                                        label={t('system.authentication.jit')}
                                        description={t('system.authentication.jitDescription')}
                                        checked={ssoAllowSignup}
                                        disabled={updateSettings.isPending}
                                        onChange={(checked) => save({ sso_allow_signup: checked })}
                                    />
                                    <ToggleRow
                                        id="sso-verified-email"
                                        label={t('system.authentication.verifiedEmail')}
                                        description={
                                            ssoAllowSignup
                                                ? t('system.authentication.verifiedEmailOn')
                                                : t('system.authentication.verifiedEmailOff')
                                        }
                                        checked={requireVerifiedEmail}
                                        disabled={updateSettings.isPending || !ssoAllowSignup}
                                        onChange={(checked) => save({ sso_require_verified_email: checked })}
                                    />
                                </>
                            )}
                        </div>
                    </SettingsSection>

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
                                        onChange={(checked) => save({ space_invites_can_register: checked })}
                                    />
                                    <NumberRow
                                        id="max-spaces"
                                        label={t('system.spaces.maxSpaces')}
                                        description={t('system.spaces.maxSpacesDescription')}
                                        value={settings.max_spaces_per_user}
                                        nullable
                                        onSave={(v) => save({ max_spaces_per_user: v })}
                                    />
                                    <NumberRow
                                        id="quota"
                                        label={t('system.spaces.quota')}
                                        description={t('system.spaces.quotaDescription')}
                                        value={settings.space_quota_mb}
                                        onSave={(v) => save({ space_quota_mb: v ?? 0 })}
                                    />
                                    <NumberRow
                                        id="backups-max"
                                        label={t('system.spaces.backupsMax')}
                                        description={t('system.spaces.backupsMaxDescription')}
                                        value={settings.space_backups_max}
                                        onSave={(v) => save({ space_backups_max: v ?? 0 })}
                                    />
                                </>
                            )}
                        </div>
                    </SettingsSection>
                </div>
            </FormWrapper>
        </Page>
    )
}
