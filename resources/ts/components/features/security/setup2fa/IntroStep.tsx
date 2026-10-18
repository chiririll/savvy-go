import { useTranslation } from 'react-i18next'
import { Loader2, ShieldCheck } from 'lucide-react'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { testId } from '@/lib/test-id'
import { cn } from '@/lib/utils'

interface IntroStepProps {
    enabling: boolean
    onEnable: () => void
    onSkip: () => void
}

export function IntroStep({ enabling, onEnable, onSkip }: IntroStepProps) {
    const { t } = useTranslation('auth')

    return (
        <>
            <CardHeader className="text-center">
                <div className="flex justify-center mb-4">
                    <div
                        className={cn(
                            'flex size-12 items-center justify-center rounded-xl bg-primary',
                            'text-primary-foreground',
                        )}
                    >
                        <ShieldCheck className="size-7" />
                    </div>
                </div>
                <CardTitle className="text-2xl">{t('setup2fa.title')}</CardTitle>
                <CardDescription>{t('setup2fa.description')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
                <div className="text-sm text-muted-foreground space-y-2">
                    <p>{t('setup2fa.withEnabled')}</p>
                    <ul className="list-disc list-inside space-y-1">
                        <li>{t('setup2fa.needPhone')}</li>
                        <li>{t('setup2fa.protected')}</li>
                        <li>{t('setup2fa.recoveryCodes')}</li>
                    </ul>
                </div>
                <div className="flex flex-col gap-2">
                    <Button onClick={onEnable} disabled={enabling} className="w-full" {...testId('2fa-enable')}>
                        {enabling && <Loader2 className="mr-2 size-4 animate-spin" />}
                        {t('setup2fa.enable')}
                    </Button>
                    <Button variant="ghost" onClick={onSkip} className="w-full" {...testId('2fa-skip')}>
                        {t('setup2fa.skip')}
                    </Button>
                </div>
            </CardContent>
        </>
    )
}
