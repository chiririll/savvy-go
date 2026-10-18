import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Card } from '@/components/ui/card'
import { useEnableTwoFactor, useConfirmTwoFactor } from '@/hooks'
import { LanguageSwitcher } from '@/components/shared'
import { IntroStep } from '@/components/features/security/setup2fa/IntroStep'
import { QrStep, type TwoFactorSecret } from '@/components/features/security/setup2fa/QrStep'
import { VerifyStep } from '@/components/features/security/setup2fa/VerifyStep'
import { RecoveryStep } from '@/components/features/security/setup2fa/RecoveryStep'

type SetupStep = 'intro' | 'qr' | 'verify' | 'recovery'

export default function Setup2FAPage() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()

    const [step, setStep] = useState<SetupStep>('intro')
    const [qrData, setQrData] = useState<TwoFactorSecret | null>(null)
    const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
    const [otpValue, setOtpValue] = useState('')

    const enableMutation = useEnableTwoFactor()
    const confirmMutation = useConfirmTwoFactor()

    useEffect(() => {
        if (sessionStorage.getItem('just_registered') !== 'true') {
            navigate('/', { replace: true })
        }
    }, [navigate])

    const handleFinish = () => {
        sessionStorage.removeItem('just_registered')
        navigate('/')
    }

    const handleSkip = () => {
        toast.info(t('setup2fa.skipToast'))
        handleFinish()
    }

    const handleComplete = () => {
        toast.success(t('setup2fa.enabledToast'))
        handleFinish()
    }

    const handleEnable = async () => {
        try {
            const data = await enableMutation.mutateAsync()
            setQrData(data)
            setStep('qr')
        } catch {
            // toasted by the mutation, or the user cancelled the confirmation
        }
    }

    const handleConfirm = async () => {
        const data = await confirmMutation.mutateAsync(otpValue)
        setRecoveryCodes(data.recovery_codes)
        setStep('recovery')
        setOtpValue('')
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                {step === 'intro' && (
                    <IntroStep enabling={enableMutation.isPending} onEnable={handleEnable} onSkip={handleSkip} />
                )}
                {step === 'qr' && qrData && (
                    <QrStep data={qrData} onBack={() => setStep('intro')} onContinue={() => setStep('verify')} />
                )}
                {step === 'verify' && (
                    <VerifyStep
                        code={otpValue}
                        verifying={confirmMutation.isPending}
                        onCodeChange={setOtpValue}
                        onBack={() => setStep('qr')}
                        onVerify={handleConfirm}
                    />
                )}
                {step === 'recovery' && recoveryCodes.length > 0 && (
                    <RecoveryStep codes={recoveryCodes} onDone={handleComplete} />
                )}
            </Card>
            <LanguageSwitcher variant="auth" />
        </div>
    )
}
