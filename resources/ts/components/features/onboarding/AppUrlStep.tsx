import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { UseBrowserUrlButton } from '@/components/features/settings/UseBrowserUrlButton'
import { testId } from '@/lib/test-id'

interface AppUrlStepProps {
    value: string
    /** False when config.toml is read-only: the URL can only be set in the file. */
    writable: boolean
    saving: boolean
    /** Called with the URL to save, or null when there is nothing to save. */
    onContinue: (url: string | null) => void
}

/** Server setup, the required step: the public URL. The address the admin opened the app at is the usual answer. */
export function AppUrlStep({ value, writable, saving, onContinue }: AppUrlStepProps) {
    const { t } = useTranslation('auth')
    const browserUrl = window.location.origin
    const [draft, setDraft] = useState(value)
    const url = draft.trim()

    return (
        <div className="space-y-4">
            <p className="text-sm text-muted-foreground">{t('serverSetup.url.description')}</p>
            {writable ? (
                <div className="space-y-2">
                    <Label htmlFor="app-url">{t('serverSetup.url.label')}</Label>
                    <div className="relative">
                        <Input
                            id="app-url"
                            type="url"
                            inputMode="url"
                            autoCapitalize="none"
                            spellCheck={false}
                            className="pr-10"
                            value={draft}
                            placeholder={browserUrl}
                            onChange={(e) => setDraft(e.target.value)}
                        />
                        <UseBrowserUrlButton url={browserUrl} current={draft} onUse={setDraft} />
                    </div>
                </div>
            ) : (
                <p className="rounded-md bg-muted p-3 text-sm">{t('serverSetup.url.readOnly')}</p>
            )}
            <Button
                type="button"
                className="w-full"
                disabled={saving || (writable && url === '')}
                onClick={() => onContinue(writable && url !== value ? url : null)}
                {...testId('server-setup-continue')}
            >
                {saving && <Loader2 className="mr-2 size-4 animate-spin" />}
                {t('serverSetup.continue')}
            </Button>
        </div>
    )
}
