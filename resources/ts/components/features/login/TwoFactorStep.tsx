import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { ArrowLeft, Fingerprint, Key, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { InputOTP, InputOTPGroup, InputOTPSlot, InputOTPSeparator } from '@/components/ui/input-otp'
import { useAuthStore } from '@/stores/auth'
import { getApiErrorMessage } from '@/lib/api-error'
import { AuthCardShell } from './AuthCardShell'
import { usePasskeyLogin } from './usePasskeyLogin'

interface Props {
    token: string
    rememberMe: boolean
    afterLoginPath: string
    onBack: () => void
}

function OtpInput({
    value,
    onChange,
    onComplete,
}: {
    value: string
    onChange: (v: string) => void
    onComplete: () => void
}) {
    return (
        <div className="flex justify-center">
            <InputOTP maxLength={6} value={value} onChange={onChange} onComplete={onComplete}>
                <InputOTPGroup>
                    <InputOTPSlot index={0} />
                    <InputOTPSlot index={1} />
                    <InputOTPSlot index={2} />
                </InputOTPGroup>
                <InputOTPSeparator />
                <InputOTPGroup>
                    <InputOTPSlot index={3} />
                    <InputOTPSlot index={4} />
                    <InputOTPSlot index={5} />
                </InputOTPGroup>
            </InputOTP>
        </div>
    )
}

export function TwoFactorStep({ token, rememberMe, afterLoginPath, onBack }: Props) {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const loginWith2FA = useAuthStore((state) => state.loginWith2FA)
    const passkey = usePasskeyLogin(afterLoginPath)
    const [isLoading, setIsLoading] = useState(false)
    const [otpValue, setOtpValue] = useState('')
    const [useRecoveryCode, setUseRecoveryCode] = useState(false)
    const [recoveryCode, setRecoveryCode] = useState('')

    const code = useRecoveryCode ? recoveryCode : otpValue
    const complete = useRecoveryCode ? recoveryCode !== '' : otpValue.length === 6

    const onVerify = async () => {
        if (!complete) return

        setIsLoading(true)
        try {
            await loginWith2FA(token, code, rememberMe)
            toast.success(t('welcomeToast'))
            navigate(afterLoginPath)
        } catch (error: unknown) {
            toast.error(getApiErrorMessage(error, t('twoFactor.invalidCode')))
            if (useRecoveryCode) {
                setRecoveryCode('')
            } else {
                setOtpValue('')
            }
        } finally {
            setIsLoading(false)
        }
    }

    const toggleMethod = () => {
        setUseRecoveryCode(!useRecoveryCode)
        setOtpValue('')
        setRecoveryCode('')
    }

    return (
        <AuthCardShell
            title={t('twoFactor.title')}
            description={useRecoveryCode ? t('twoFactor.recoveryDescription') : t('twoFactor.otpDescription')}
            contentClassName="space-y-6"
        >
            {useRecoveryCode ? (
                <Input
                    type="text"
                    placeholder={t('twoFactor.recoveryPlaceholder')}
                    value={recoveryCode}
                    onChange={(e) => setRecoveryCode(e.target.value)}
                    className="text-center font-mono text-lg"
                    autoFocus
                />
            ) : (
                <OtpInput value={otpValue} onChange={setOtpValue} onComplete={onVerify} />
            )}

            <div className="flex flex-col gap-2">
                <Button onClick={onVerify} className="w-full" disabled={!complete || isLoading}>
                    {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                    {t('twoFactor.verify')}
                </Button>
                {passkey.supported && (
                    <Button
                        variant="outline"
                        onClick={() => passkey.signIn(token)}
                        className="w-full"
                        disabled={passkey.loading || isLoading}
                    >
                        {passkey.loading ? (
                            <Loader2 className="mr-2 size-4 animate-spin" />
                        ) : (
                            <Fingerprint className="mr-2 size-4" />
                        )}
                        {t('twoFactor.usePasskey')}
                    </Button>
                )}
                <Button variant="ghost" onClick={toggleMethod} className="w-full">
                    <Key className="mr-2 size-4" />
                    {useRecoveryCode ? t('twoFactor.useAuthenticator') : t('twoFactor.useRecovery')}
                </Button>
                <Button variant="ghost" onClick={onBack} className="w-full">
                    <ArrowLeft className="mr-2 size-4" />
                    {t('twoFactor.backToLogin')}
                </Button>
            </div>
        </AuthCardShell>
    )
}
