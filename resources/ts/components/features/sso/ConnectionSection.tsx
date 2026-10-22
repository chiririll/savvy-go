import { Plug } from 'lucide-react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useSsoPresets } from '@/hooks/use-sso'
import type { IdentityProviderFormValues } from '@/schemas/sso'
import type { PresetField } from '@/types/sso'

import { RedirectUrls } from './form-parts'
import { ssoFieldLabel, ssoFieldPlaceholder, ssoPresetLabel } from './presetMeta'
import { SettingsSection } from '@/components/shared'

type Form = UseFormReturn<IdentityProviderFormValues>

interface ConnectionSectionProps {
    form: Form
    isEdit: boolean
    presetLocked: boolean
}

function PresetFieldInput({
    form,
    field,
    preset,
    isEdit,
}: {
    form: Form
    field: PresetField
    preset: string
    isEdit: boolean
}) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <FormField
            control={form.control}
            name={`fields.${field.key}` as const}
            render={({ field: f }) => (
                <FormItem>
                    <FormLabel>{ssoFieldLabel(preset, field, t)}</FormLabel>
                    <FormControl>
                        {field.type === 'textarea' ? (
                            <Textarea
                                placeholder={ssoFieldPlaceholder(field, t)}
                                rows={4}
                                {...f}
                                value={f.value ?? ''}
                            />
                        ) : (
                            <Input
                                type={field.secret ? 'password' : field.type === 'url' ? 'url' : 'text'}
                                placeholder={
                                    isEdit && field.secret
                                        ? t('forms:sso.secretKeepPlaceholder')
                                        : ssoFieldPlaceholder(field, t)
                                }
                                {...f}
                                value={f.value ?? ''}
                            />
                        )}
                    </FormControl>
                    {field.help && <FormDescription>{field.help}</FormDescription>}
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}

export function ConnectionSection({ form, isEdit, presetLocked }: ConnectionSectionProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { data: presets } = useSsoPresets()
    const selectedKey = form.watch('preset')
    const selectedPreset = presets?.find((p) => p.key === selectedKey)
    const slug = form.watch('slug')

    return (
        <SettingsSection icon={Plug} title={t('forms:sso.connection')} description={t('forms:sso.connectionHelp')} contentClassName="space-y-5">
            <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('forms:sso.displayName')}</FormLabel>
                        <FormControl>
                            <Input placeholder={t('forms:sso.displayNamePlaceholder')} {...field} />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                )}
            />

            <FormField
                control={form.control}
                name="slug"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('forms:sso.slug')}</FormLabel>
                        <FormControl>
                            <Input placeholder={t('forms:sso.slugPlaceholder')} {...field} disabled={isEdit} />
                        </FormControl>
                        <FormDescription>{t('forms:sso.slugHelp')}</FormDescription>
                        <FormMessage />
                    </FormItem>
                )}
            />

            {!presetLocked && (
                <FormField
                    control={form.control}
                    name="preset"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('forms:sso.providerType')}</FormLabel>
                            <Select value={field.value} onValueChange={field.onChange} disabled={isEdit}>
                                <FormControl>
                                    <SelectTrigger className="w-full">
                                        <SelectValue placeholder={t('forms:sso.selectProviderType')} />
                                    </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                    {presets?.map((preset) => (
                                        <SelectItem key={preset.key} value={preset.key}>
                                            {ssoPresetLabel(preset, t)} ({preset.protocol.toUpperCase()})
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                            <FormMessage />
                        </FormItem>
                    )}
                />
            )}

            {selectedPreset?.fields.map((field) => (
                <PresetFieldInput key={field.key} form={form} field={field} preset={selectedKey} isEdit={isEdit} />
            ))}

            {selectedPreset && slug && <RedirectUrls slug={slug} protocol={selectedPreset.protocol} />}
        </SettingsSection>
    )
}
