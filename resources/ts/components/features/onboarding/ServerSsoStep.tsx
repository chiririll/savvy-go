import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { IdentityProviderForm, PresetChooser, SelectedPresetBanner } from '@/components/features/sso'
import { useCreateIdentityProvider } from '@/hooks'
import { serverSetupPaths } from './server-setup-paths'

/** Server setup: add an identity provider, then back to the list. */
export function ServerSsoStep() {
    const { t } = useTranslation('auth')
    const { t: tCommon } = useTranslation('common')
    const [preset, setPreset] = useState<string | null>(null)
    const create = useCreateIdentityProvider(serverSetupPaths.options)

    return (
        <div className="space-y-4">
            {preset ? (
                <>
                    <SelectedPresetBanner preset={preset} onChange={() => setPreset(null)} />
                    <IdentityProviderForm
                        presetLocked
                        previewPreset={preset}
                        defaultValues={{ preset }}
                        onSubmit={(data) => create.mutate(data)}
                        isSubmitting={create.isPending}
                        submitLabel={tCommon('actions.create')}
                    />
                </>
            ) : (
                <PresetChooser onSelect={setPreset} />
            )}
            <Button asChild variant="ghost" className="w-full">
                <Link to={serverSetupPaths.options}>{t('serverSetup.backToList')}</Link>
            </Button>
        </div>
    )
}
