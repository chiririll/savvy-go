import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { BootstrapInstructions } from '@/components/features/onboarding/BootstrapInstructions'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { testId } from '@/lib/test-id'
import type { BootstrapHint } from '@/types'

interface BootstrapStepProps {
    hint?: BootstrapHint | null
    password: string
    onPasswordChange: (value: string) => void
    error: string | null
    isLoading: boolean
    onNext: (next: 'admin' | 'restore') => void
}

export function BootstrapStep({ hint, password, onPasswordChange, error, isLoading, onNext }: BootstrapStepProps) {
    const { t } = useTranslation('auth')

    return (
        <>
            {hint && <BootstrapInstructions hint={hint} />}
            <div className="space-y-2">
                <Label htmlFor="bootstrap-password">{t('setup.bootstrapPassword')}</Label>
                <Input
                    id="bootstrap-password"
                    value={password}
                    onChange={(e) => onPasswordChange(e.target.value)}
                    placeholder="XXXXX-XXXXX-XXXXX-XXXXX-XXXXX-XXXXX"
                    autoComplete="off"
                    autoCapitalize="characters"
                    spellCheck={false}
                    aria-invalid={error !== null}
                />
                {error && <p className="text-sm text-destructive">{error}</p>}
            </div>
            <Button
                type="button"
                className="w-full"
                disabled={isLoading}
                onClick={() => onNext('admin')}
                {...testId('setup-create-admin')}
            >
                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                {t('setup.createAdmin')}
            </Button>
            <Button
                type="button"
                variant="outline"
                className="w-full"
                disabled={isLoading}
                onClick={() => onNext('restore')}
                {...testId('setup-restore')}
            >
                {t('setup.restore')}
            </Button>
        </>
    )
}
