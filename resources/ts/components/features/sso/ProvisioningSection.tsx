import { UserCog } from 'lucide-react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { IdentityProviderFormValues } from '@/schemas/sso'

import { Section } from './form-parts'

export function ProvisioningSection({ form }: { form: UseFormReturn<IdentityProviderFormValues> }) {
    const { t } = useTranslation(['common', 'forms'])
    const TOGGLES = [
        { name: 'enabled', label: t('forms:sso.enabled'), help: t('forms:sso.enabledHelp') },
        { name: 'allow_jit', label: t('forms:sso.allowJit'), help: t('forms:sso.allowJitHelp') },
        { name: 'link_by_email', label: t('forms:sso.linkByEmail'), help: t('forms:sso.linkByEmailHelp') },
        { name: 'sync_role_on_login', label: t('forms:sso.syncRole'), help: t('forms:sso.syncRoleHelp') },
    ] as const

    return (
        <Section icon={UserCog} title={t('forms:sso.provisioning')} description={t('forms:sso.provisioningHelp')}>
            <FormField
                control={form.control}
                name="default_role"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('forms:sso.defaultRole')}</FormLabel>
                        <Select value={field.value} onValueChange={field.onChange}>
                            <FormControl>
                                <SelectTrigger className="w-full">
                                    <SelectValue />
                                </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                                <SelectItem value="user">{t('roles.user')}</SelectItem>
                                <SelectItem value="guest">{t('roles.guest')}</SelectItem>
                            </SelectContent>
                        </Select>
                        <FormDescription>{t('forms:sso.defaultRoleHelp')}</FormDescription>
                        <FormMessage />
                    </FormItem>
                )}
            />

            <div className="divide-y rounded-lg border">
                {TOGGLES.map((toggle) => (
                    <FormField
                        key={toggle.name}
                        control={form.control}
                        name={toggle.name}
                        render={({ field }) => (
                            <FormItem className="flex items-center justify-between gap-4 px-4 py-3">
                                <div className="space-y-0.5">
                                    <FormLabel className="cursor-pointer">{toggle.label}</FormLabel>
                                    <FormDescription>{toggle.help}</FormDescription>
                                </div>
                                <FormControl>
                                    <Switch checked={Boolean(field.value)} onCheckedChange={field.onChange} />
                                </FormControl>
                            </FormItem>
                        )}
                    />
                ))}
            </div>
        </Section>
    )
}
