import { useTranslation } from 'react-i18next'
import { Copy } from 'lucide-react'
import { toast } from 'sonner'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { testId } from '@/lib/test-id'

interface RecoveryStepProps {
    codes: string[]
    onDone: () => void
}

export function RecoveryStep({ codes, onDone }: RecoveryStepProps) {
    const { t } = useTranslation('auth')
    const { t: tCommon } = useTranslation('common')

    const copyCodes = () => {
        navigator.clipboard.writeText(codes.join('\n'))
        toast.success(t('setup2fa.codesCopied'))
    }

    return (
        <>
            <CardHeader className="text-center">
                <CardTitle>{t('setup2fa.recoveryTitle')}</CardTitle>
                <CardDescription>{t('setup2fa.recoveryDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
                <div
                    className="bg-muted p-4 rounded-md font-mono text-sm grid grid-cols-2 gap-2"
                    {...testId('2fa-recovery-codes')}
                >
                    {codes.map((code, index) => (
                        <div key={index} className="text-center">{code}</div>
                    ))}
                </div>
                <Button variant="outline" className="w-full" onClick={copyCodes}>
                    <Copy className="h-4 w-4 mr-2" />
                    {t('setup2fa.copyCodes')}
                </Button>
                <Button onClick={onDone} className="w-full" {...testId('2fa-done')}>
                    {tCommon('actions.done')}
                </Button>
            </CardContent>
        </>
    )
}
