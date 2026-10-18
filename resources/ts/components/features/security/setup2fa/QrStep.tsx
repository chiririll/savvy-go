import { useTranslation } from 'react-i18next'
import { ArrowLeft, Copy } from 'lucide-react'
import { toast } from 'sonner'
import { QRCode } from 'react-qrcode-logo'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/hooks'
import { testId } from '@/lib/test-id'

export interface TwoFactorSecret {
    secret: string
    qr_code_url: string
}

interface QrStepProps {
    data: TwoFactorSecret
    onBack: () => void
    onContinue: () => void
}

export function QrStep({ data, onBack, onContinue }: QrStepProps) {
    const { t } = useTranslation('auth')
    const { t: tCommon } = useTranslation('common')
    const { theme } = useTheme()

    const copySecret = () => {
        navigator.clipboard.writeText(data.secret)
        toast.success(t('setup2fa.secretCopied'))
    }

    return (
        <>
            <CardHeader className="text-center">
                <CardTitle>{t('setup2fa.scanTitle')}</CardTitle>
                <CardDescription>{t('setup2fa.scanDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
                <div className="flex justify-center" {...testId('2fa-qr')}>
                    <QRCode
                        value={data.qr_code_url}
                        size={240}
                        ecLevel="M"
                        bgColor="transparent"
                        fgColor={theme === 'dark' ? '#ffffff' : '#0f172a'}
                        qrStyle="dots"
                        eyeRadius={6}
                        quietZone={0}
                    />
                </div>
                <div>
                    <p className="text-sm text-muted-foreground mb-2 text-center">{t('setup2fa.manualCode')}</p>
                    <div className="flex items-center gap-2 bg-muted p-2 rounded-md">
                        <code className="flex-1 text-sm font-mono break-all" {...testId('2fa-secret')}>
                            {data.secret}
                        </code>
                        <Button variant="ghost" size="sm" onClick={copySecret}>
                            <Copy className="h-4 w-4" />
                        </Button>
                    </div>
                </div>
                <div className="flex gap-2">
                    <Button variant="outline" onClick={onBack} className="flex-1">
                        <ArrowLeft className="mr-2 h-4 w-4" />
                        {tCommon('actions.back')}
                    </Button>
                    <Button onClick={onContinue} className="flex-1" {...testId('2fa-continue')}>
                        {tCommon('actions.continue')}
                    </Button>
                </div>
            </CardContent>
        </>
    )
}
