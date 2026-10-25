import { toast } from 'sonner'
import { Lock, ShieldCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SettingsSection, ToggleRow, SettingsRowSkeleton } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { useSettings, useUpdateSettings, useSsoProviders, useMailStatus } from '@/hooks'
import { authApi } from '@/api'
import { useUser } from '@/stores/auth'
import type { Settings } from '@/types'
import { cn } from '@/lib/utils'

interface RowsProps {
    settings: Settings | undefined
    pending: boolean
    save: (data: Partial<Settings>) => void
}

function LockedBadge() {
    const { t } = useTranslation('settings')
    return (
        <span
            className={cn(
                'inline-flex items-center gap-1 rounded-full border bg-muted px-1.5 py-0.5',
                'text-[10px] font-medium uppercase tracking-wide text-muted-foreground',
            )}
        >
            <Lock className="size-2.5" />
            {t('system.authentication.locked')}
        </span>
    )
}

function PasswordLoginRow({ settings, pending, save }: RowsProps) {
    const { t } = useTranslation('settings')
    const { data: ssoProviders } = useSsoProviders()
    const hasSso = (ssoProviders?.length ?? 0) > 0
    const enabled = settings?.password_login_enabled ?? true
    const locked = enabled && !hasSso

    const handleChange = (checked: boolean) => {
        if (!checked && !hasSso) {
            toast.error(t('system.authentication.passwordLoginError'))
            return
        }
        save({ password_login_enabled: checked })
    }

    return (
        <ToggleRow
            id="password-login"
            label={t('system.authentication.passwordLogin')}
            description={
                locked
                    ? t('system.authentication.passwordLoginLocked')
                    : t('system.authentication.passwordLoginOpen')
            }
            checked={enabled}
            disabled={pending || locked}
            badge={locked ? <LockedBadge /> : undefined}
            onChange={handleChange}
        />
    )
}

function SsoRows({ settings, pending, save }: RowsProps) {
    const { t } = useTranslation('settings')
    const allowSignup = settings?.sso_allow_signup ?? true
    return (
        <>
            <ToggleRow
                id="sso-signup"
                label={t('system.authentication.jit')}
                description={t('system.authentication.jitDescription')}
                checked={allowSignup}
                disabled={pending}
                onChange={(checked) => save({ sso_allow_signup: checked })}
            />
            <ToggleRow
                id="sso-verified-email"
                label={t('system.authentication.verifiedEmail')}
                description={
                    allowSignup
                        ? t('system.authentication.verifiedEmailOn')
                        : t('system.authentication.verifiedEmailOff')
                }
                checked={settings?.sso_require_verified_email ?? false}
                disabled={pending || !allowSignup}
                onChange={(checked) => save({ sso_require_verified_email: checked })}
            />
        </>
    )
}

function EmailVerificationRow({ settings, pending, save }: RowsProps) {
    const { t } = useTranslation('settings')
    const { data: mail } = useMailStatus()
    const me = useUser()
    const required = settings?.require_email_verification ?? false
    // Links go out by email, and the admin switching it on must not lock themselves out.
    const mailReady = !!mail?.enabled && !!mail.appUrlSet
    const adminConfirmed = me?.emailVerified === true
    const blocker = !mailReady
        ? t('system.authentication.requireVerificationNeedsMail')
        : !adminConfirmed
          ? t('system.authentication.requireVerificationNeedsYou')
          : null

    const sendConfirmation = () => {
        authApi
            .resendVerification()
            .then(() => toast.success(t('system.authentication.confirmationSent')))
            .catch((error: { message?: string }) => toast.error(error.message ?? ''))
    }

    return (
        <>
            <ToggleRow
                id="require-email-verification"
                label={t('system.authentication.requireVerification')}
                description={(!required && blocker) || t('system.authentication.requireVerificationOn')}
                checked={required}
                disabled={pending || (!required && blocker !== null)}
                onChange={(checked) => save({ require_email_verification: checked })}
            />
            {!required && mailReady && !adminConfirmed && (
                <div className="px-4 pb-3.5">
                    <Button size="sm" variant="outline" onClick={sendConfirmation}>
                        {t('system.authentication.sendConfirmation')}
                    </Button>
                </div>
            )}
        </>
    )
}

export function AuthenticationSection() {
    const { t } = useTranslation('settings')
    const { data: settings, isLoading } = useSettings()
    const updateSettings = useUpdateSettings()
    const rows: RowsProps = {
        settings,
        pending: updateSettings.isPending,
        save: (data) => updateSettings.mutate(data),
    }

    return (
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
                        <PasswordLoginRow {...rows} />
                        <SsoRows {...rows} />
                        <EmailVerificationRow {...rows} />
                    </>
                )}
            </div>
        </SettingsSection>
    )
}
