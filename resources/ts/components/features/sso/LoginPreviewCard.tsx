import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { BrandIcon, brandVars } from './BrandIcon'
import { cn } from '@/lib/utils'

interface LoginPreviewCardProps {
    preset: string
    name: string
    enabled: boolean
}

export function LoginPreviewCard({ preset, name, enabled }: LoginPreviewCardProps) {
    const { t } = useTranslation('forms')

    return (
        <Card className="lg:sticky lg:top-6">
            <CardHeader>
                <CardTitle className="text-sm">{t('sso.loginPreview')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
                <div
                    style={brandVars(preset)}
                    className="rounded-lg border bg-[color-mix(in_oklab,var(--brand)_5%,var(--background))] p-4"
                >
                    <div
                        className={cn(
                            'inline-flex h-10 w-full items-center justify-center gap-2.5 rounded-md border',
                            'bg-background px-4 text-sm font-medium shadow-sm',
                        )}
                    >
                        <BrandIcon preset={preset} className="size-4" />
                        {t('sso.continueWith', { name: name || t('sso.providerFallback') })}
                    </div>
                </div>
                <div className="flex items-center gap-2 text-xs">
                    <span
                        className={`size-1.5 rounded-full ${enabled ? 'bg-emerald-500' : 'bg-muted-foreground/40'}`}
                    />
                    <span className="text-muted-foreground">
                        {enabled ? t('sso.visibleOnLogin') : t('sso.hiddenOnLogin')}
                    </span>
                </div>
            </CardContent>
        </Card>
    )
}
