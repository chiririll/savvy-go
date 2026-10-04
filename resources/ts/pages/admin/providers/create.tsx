import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FormPage } from '@/components/shared'
import { IdentityProviderForm, PresetChooser, SelectedPresetBanner } from '@/components/features/sso'
import { useCreateIdentityProvider } from '@/hooks/use-sso'

export default function ProviderCreatePage() {
    const { t } = useTranslation('settings')
    const [preset, setPreset] = useState<string | null>(null)
    const createProvider = useCreateIdentityProvider('/admin/sso')

    if (!preset) {
        return (
            <FormPage title={t('providers.create')} backLink="/admin/sso">
                <PresetChooser onSelect={setPreset} />
            </FormPage>
        )
    }

    return (
        <FormPage title={t('providers.create')} backLink="/admin/sso">
            <div className="space-y-6">
                <SelectedPresetBanner preset={preset} onChange={() => setPreset(null)} />

                <IdentityProviderForm
                    presetLocked
                    previewPreset={preset}
                    defaultValues={{ preset }}
                    onSubmit={(data) => createProvider.mutate(data)}
                    isSubmitting={createProvider.isPending}
                    submitLabel={t('actions.create', { ns: 'common' })}
                />
            </div>
        </FormPage>
    )
}
