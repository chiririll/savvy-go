import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { QRCode } from 'react-qrcode-logo'
import { Copy } from 'lucide-react'
import { ResponsiveDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/hooks'
import { OtpField } from './SecurityLayout'
import type { SetupStep, TwoFactorSetup } from './useTwoFactorSetup'

function stepText(t: TFunction<'settings'>, step: SetupStep) {
    switch (step) {
        case 'qr':
            return {
                title: t('security.dialogs.enableTitle'),
                description: t('security.dialogs.enableDescription'),
            }
        case 'verify':
            return {
                title: t('security.dialogs.verifyTitle'),
                description: t('security.dialogs.verifyDescription'),
            }
        case 'recovery':
            return {
                title: t('security.dialogs.recoveryTitle'),
                description: t('security.dialogs.recoveryDescription'),
            }
    }
}

function QrStep({ setup }: { setup: TwoFactorSetup }) {
    const { t } = useTranslation('settings')
    const { theme } = useTheme()
    const { qrData, copySecret } = setup

    if (!qrData) return null

    return (
        <div className="flex flex-col items-center gap-4">
            <QRCode
                value={qrData.qr_code_url}
                size={240}
                ecLevel="M"
                bgColor="transparent"
                fgColor={theme === 'dark' ? '#ffffff' : '#0f172a'}
                qrStyle="dots"
                eyeRadius={6}
                quietZone={0}
                style={{ maxWidth: '100%', height: 'auto' }}
            />
            <div className="w-full">
                <p className="text-sm text-muted-foreground mb-2 text-center">{t('security.dialogs.manualCode')}</p>
                <div className="flex items-center gap-2 bg-muted p-2 rounded-md">
                    <code className="flex-1 text-sm font-mono break-all">{qrData.secret}</code>
                    <Button variant="ghost" size="sm" onClick={copySecret}>
                        <Copy className="h-4 w-4" />
                    </Button>
                </div>
            </div>
        </div>
    )
}

function RecoveryStep({ setup }: { setup: TwoFactorSetup }) {
    const { t } = useTranslation('settings')
    const { recoveryCodes, copyRecoveryCodes } = setup

    if (recoveryCodes.length === 0) return null

    return (
        <>
            <div className="bg-muted p-4 rounded-md font-mono text-sm grid grid-cols-2 gap-2">
                {recoveryCodes.map((code, index) => (
                    <div key={index} className="text-center">
                        {code}
                    </div>
                ))}
            </div>
            <Button variant="outline" className="w-full mt-4" onClick={copyRecoveryCodes}>
                <Copy className="h-4 w-4 mr-2" />
                {t('security.dialogs.copyCodes')}
            </Button>
        </>
    )
}

function EnableDialog({ setup }: { setup: TwoFactorSetup }) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const { setupStep, setSetupStep, otpValue, setOtpValue, confirmMutation, handleConfirm, closeEnableDialog } = setup

    const text = stepText(t, setupStep)
    const footer =
        setupStep === 'qr' ? (
            <Button onClick={() => setSetupStep('verify')}>{tCommon('actions.continue')}</Button>
        ) : setupStep === 'verify' ? (
            <>
                <Button variant="outline" onClick={() => setSetupStep('qr')}>
                    {tCommon('actions.back')}
                </Button>
                <Button onClick={handleConfirm} disabled={otpValue.length !== 6 || confirmMutation.isPending}>
                    {confirmMutation.isPending ? t('security.dialogs.verifying') : t('security.dialogs.verify')}
                </Button>
            </>
        ) : (
            <Button onClick={closeEnableDialog}>{tCommon('actions.done')}</Button>
        )

    return (
        <ResponsiveDialog
            open={setup.showEnableDialog}
            onOpenChange={closeEnableDialog}
            title={text.title}
            description={text.description}
            footer={footer}
        >
            {setupStep === 'qr' && <QrStep setup={setup} />}
            {setupStep === 'verify' && <OtpField value={otpValue} onChange={setOtpValue} />}
            {setupStep === 'recovery' && <RecoveryStep setup={setup} />}
        </ResponsiveDialog>
    )
}

function DisableDialog({ setup }: { setup: TwoFactorSetup }) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const { otpValue, setOtpValue, disableMutation, handleDisable, setShowDisableDialog, closeWithOtpReset } = setup

    return (
        <ResponsiveDialog
            open={setup.showDisableDialog}
            onOpenChange={setShowDisableDialog}
            title={t('security.dialogs.disableTitle')}
            description={t('security.dialogs.disableDescription')}
            footer={
                <>
                    <Button variant="outline" onClick={closeWithOtpReset(setShowDisableDialog)}>
                        {tCommon('actions.cancel')}
                    </Button>
                    <Button
                        variant="destructive"
                        onClick={handleDisable}
                        disabled={otpValue.length !== 6 || disableMutation.isPending}
                    >
                        {disableMutation.isPending ? t('security.dialogs.disabling') : t('security.dialogs.disable')}
                    </Button>
                </>
            }
        >
            <OtpField value={otpValue} onChange={setOtpValue} />
        </ResponsiveDialog>
    )
}

function RegenerateDialog({ setup }: { setup: TwoFactorSetup }) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const { otpValue, setOtpValue, regenerateMutation, handleRegenerate, setShowRegenerateDialog, closeWithOtpReset } =
        setup

    return (
        <ResponsiveDialog
            open={setup.showRegenerateDialog}
            onOpenChange={setShowRegenerateDialog}
            title={t('security.recovery.regenerateTitle')}
            description={t('security.recovery.regenerateDescription')}
            footer={
                <>
                    <Button variant="outline" onClick={closeWithOtpReset(setShowRegenerateDialog)}>
                        {tCommon('actions.cancel')}
                    </Button>
                    <Button onClick={handleRegenerate} disabled={otpValue.length !== 6 || regenerateMutation.isPending}>
                        {regenerateMutation.isPending
                            ? t('security.recovery.regenerating')
                            : t('security.recovery.regenerate')}
                    </Button>
                </>
            }
        >
            <OtpField value={otpValue} onChange={setOtpValue} />
        </ResponsiveDialog>
    )
}

export function TwoFactorDialogs({ setup }: { setup: TwoFactorSetup }) {
    return (
        <>
            <EnableDialog setup={setup} />
            <DisableDialog setup={setup} />
            <RegenerateDialog setup={setup} />
        </>
    )
}
