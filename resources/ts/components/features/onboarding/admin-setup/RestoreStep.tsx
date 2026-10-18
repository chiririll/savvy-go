import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { isLaravelFile } from '@/components/features/backups/LaravelRestoreOptions'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { testId } from '@/lib/test-id'

interface RestoreStepProps {
    backupFile: File | null
    onFileChange: (file: File | null) => void
    appKey: string
    onAppKeyChange: (value: string) => void
    error: string | null
    isLoading: boolean
    onBack: () => void
}

export function RestoreStep({
    backupFile,
    onFileChange,
    appKey,
    onAppKeyChange,
    error,
    isLoading,
    onBack,
}: RestoreStepProps) {
    const { t } = useTranslation('auth')

    return (
        <>
            <div className="space-y-2">
                <Label htmlFor="backup-file">{t('setup.backupFile')}</Label>
                <Input
                    id="backup-file"
                    type="file"
                    accept=".zip,.sqlite"
                    onChange={(e) => onFileChange(e.target.files?.[0] ?? null)}
                    aria-invalid={error !== null}
                />
                <p className="text-xs text-muted-foreground">{t('setup.backupFileHint')}</p>
                {error && <p className="text-sm text-destructive">{error}</p>}
            </div>
            {isLaravelFile(backupFile) && (
                <div className="space-y-2">
                    <Label htmlFor="restore-app-key">{t('setup.appKey')}</Label>
                    <Input
                        id="restore-app-key"
                        value={appKey}
                        onChange={(e) => onAppKeyChange(e.target.value)}
                        placeholder="base64:..."
                        autoComplete="off"
                        spellCheck={false}
                    />
                    <p className="text-xs text-muted-foreground">{t('setup.appKeyHint')}</p>
                </div>
            )}
            <Button type="submit" className="w-full" disabled={isLoading}>
                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                {t('setup.restoreSubmit')}
            </Button>
            <Button
                type="button"
                variant="ghost"
                className="w-full"
                disabled={isLoading}
                onClick={onBack}
                {...testId('setup-back')}
            >
                {t('setup.back')}
            </Button>
        </>
    )
}
