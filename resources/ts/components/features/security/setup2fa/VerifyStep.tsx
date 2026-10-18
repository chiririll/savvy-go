import { useTranslation } from 'react-i18next'
import { ArrowLeft, Loader2 } from 'lucide-react'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { OtpCodeInput } from '@/components/shared'
import { testId } from '@/lib/test-id'

interface VerifyStepProps {
    code: string
    verifying: boolean
    onCodeChange: (code: string) => void
    onBack: () => void
    onVerify: () => void
}

export function VerifyStep({ code, verifying, onCodeChange, onBack, onVerify }: VerifyStepProps) {
    const { t } = useTranslation('auth')
    const { t: tCommon } = useTranslation('common')

    return (
        <>
            <CardHeader className="text-center">
                <CardTitle>{t('setup2fa.verifyTitle')}</CardTitle>
                <CardDescription>{t('setup2fa.verifyDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
                <OtpCodeInput value={code} onChange={onCodeChange} />
                <div className="flex gap-2">
                    <Button variant="outline" onClick={onBack} className="flex-1">
                        <ArrowLeft className="mr-2 h-4 w-4" />
                        {tCommon('actions.back')}
                    </Button>
                    <Button
                        onClick={onVerify}
                        disabled={code.length !== 6 || verifying}
                        className="flex-1"
                        {...testId('2fa-verify')}
                    >
                        {verifying && <Loader2 className="mr-2 size-4 animate-spin" />}
                        {t('twoFactor.verify')}
                    </Button>
                </div>
            </CardContent>
        </>
    )
}
