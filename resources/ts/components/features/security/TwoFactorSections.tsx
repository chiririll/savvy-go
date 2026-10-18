import { useTranslation } from 'react-i18next'
import { Key, RefreshCw, ShieldCheck } from 'lucide-react'
import { testId } from '@/lib/test-id'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { ActionRow, RowSkeleton, Section } from './SecurityLayout'
import type { TwoFactorSetup } from './useTwoFactorSetup'

export function TwoFactorSection({ setup }: { setup: TwoFactorSetup }) {
    const { t } = useTranslation('settings')
    const isReadOnly = useReadOnly()
    const { status, isLoading, enableMutation, handleEnable, setShowDisableDialog } = setup

    return (
        <Section
            icon={ShieldCheck}
            title={t('security.twoFactor.title')}
            description={t('security.twoFactor.description')}
        >
            <div className="rounded-lg border">
                {isLoading ? (
                    <RowSkeleton />
                ) : (
                    <ActionRow
                        label={t('security.twoFactor.authenticator')}
                        description={t('security.twoFactor.authenticatorDescription')}
                    >
                        <Switch
                            checked={status?.enabled ?? false}
                            disabled={isReadOnly || enableMutation.isPending}
                            onCheckedChange={(checked) => (checked ? handleEnable() : setShowDisableDialog(true))}
                            aria-label={t('security.twoFactor.toggleAria')}
                            {...testId('two-factor')}
                        />
                    </ActionRow>
                )}
            </div>
        </Section>
    )
}

export function RecoverySection({ setup }: { setup: TwoFactorSetup }) {
    const { t } = useTranslation('settings')
    const isReadOnly = useReadOnly()
    const { status, setShowRegenerateDialog } = setup

    if (!status?.enabled) return null

    return (
        <Section
            icon={Key}
            title={t('security.recovery.title')}
            description={t('security.recovery.description')}
        >
            <div className="rounded-lg border">
                <ActionRow
                    label={t('security.recovery.backup')}
                    description={t('security.recovery.remaining', { count: status.recovery_codes_remaining })}
                >
                    <Button
                        variant="outline"
                        onClick={() => setShowRegenerateDialog(true)}
                        {...testId('regenerate-codes')}
                        disabled={isReadOnly}
                    >
                        <RefreshCw className="mr-2 size-4" />
                        {t('security.recovery.regenerate')}
                    </Button>
                </ActionRow>
            </div>
        </Section>
    )
}
