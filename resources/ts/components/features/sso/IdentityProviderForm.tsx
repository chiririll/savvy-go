import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Form } from '@/components/ui/form'
import { FormWrapper } from '@/components/shared/FormWrapper'
import { schemaResolver } from '@/lib/form-resolver'
import { identityProviderSchema, IdentityProviderFormValues } from '@/schemas/sso'

import { ConnectionSection } from './ConnectionSection'
import { LoginPreviewCard } from './LoginPreviewCard'
import { ProvisioningSection } from './ProvisioningSection'
import { RoleMappingSection } from './RoleMappingSection'

interface IdentityProviderFormProps {
    defaultValues?: Partial<IdentityProviderFormValues>
    onSubmit: (data: IdentityProviderFormValues) => void
    isSubmitting?: boolean
    submitLabel?: string
    isEdit?: boolean
    presetLocked?: boolean
    previewPreset?: string
}

export function IdentityProviderForm({
    defaultValues,
    onSubmit,
    isSubmitting,
    submitLabel,
    isEdit = false,
    presetLocked = false,
    previewPreset,
}: IdentityProviderFormProps) {
    const { t } = useTranslation('common')

    const form = useForm<IdentityProviderFormValues>({
        resolver: schemaResolver<IdentityProviderFormValues>(identityProviderSchema),
        defaultValues: {
            name: '',
            slug: '',
            preset: '',
            enabled: true,
            fields: {},
            role_mapping: [],
            default_role: 'user',
            allow_jit: true,
            sync_role_on_login: false,
            link_by_email: true,
            ...defaultValues,
        },
    })

    const formBody = (
        <FormWrapper>
            <Form {...form}>
                <form
                    onSubmit={form.handleSubmit(onSubmit)}
                    className={`space-y-6 ${previewPreset ? 'w-full' : 'max-w-2xl'}`}
                >
                    <ConnectionSection form={form} isEdit={isEdit} presetLocked={presetLocked} />
                    <ProvisioningSection form={form} />
                    <RoleMappingSection form={form} />

                    <Button type="submit" disabled={isSubmitting} className="w-full">
                        {isSubmitting ? t('actions.saving') : (submitLabel ?? t('actions.save'))}
                    </Button>
                </form>
            </Form>
        </FormWrapper>
    )

    if (!previewPreset) return formBody

    return (
        <div className="grid max-w-5xl gap-6 lg:grid-cols-[minmax(0,1fr)_300px] lg:items-start">
            {formBody}
            <LoginPreviewCard preset={previewPreset} name={form.watch('name')} enabled={form.watch('enabled')} />
        </div>
    )
}
