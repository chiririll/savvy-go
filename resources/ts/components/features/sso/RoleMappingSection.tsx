import { Plus, Trash2, Waypoints } from 'lucide-react'
import { useFieldArray, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { IdentityProviderFormValues } from '@/schemas/sso'

import { Section } from './form-parts'
import { cn } from '@/lib/utils'

type Form = UseFormReturn<IdentityProviderFormValues>

const GRID = 'sm:grid-cols-[1fr_140px_1fr_160px_auto]'

function RuleRow({ form, index, onRemove }: { form: Form; index: number; onRemove: () => void }) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <div
            className={cn(
                'grid grid-cols-1 gap-2 rounded-lg border bg-muted/30 p-2',
                GRID,
                'sm:items-center sm:bg-transparent sm:border-0 sm:p-0',
            )}
        >
            <Input
                placeholder={t('forms:sso.claimPlaceholder')}
                className="font-mono text-sm"
                {...form.register(`role_mapping.${index}.claim`)}
            />
            <Select
                value={form.watch(`role_mapping.${index}.operator`)}
                onValueChange={(v) =>
                    form.setValue(`role_mapping.${index}.operator`, v as 'equals' | 'contains' | 'one_of')
                }
            >
                <SelectTrigger className="w-full">
                    <SelectValue />
                </SelectTrigger>
                <SelectContent>
                    <SelectItem value="equals">{t('forms:sso.equals')}</SelectItem>
                    <SelectItem value="contains">{t('forms:sso.contains')}</SelectItem>
                    <SelectItem value="one_of">{t('forms:sso.oneOf')}</SelectItem>
                </SelectContent>
            </Select>
            <Input
                placeholder={t('forms:sso.valuePlaceholder')}
                className="font-mono text-sm"
                {...form.register(`role_mapping.${index}.value`)}
            />
            <Select
                value={form.watch(`role_mapping.${index}.role`)}
                onValueChange={(v) => form.setValue(`role_mapping.${index}.role`, v as 'admin' | 'user' | 'guest')}
            >
                <SelectTrigger className="w-full">
                    <SelectValue />
                </SelectTrigger>
                <SelectContent>
                    <SelectItem value="admin">{t('roles.admin')}</SelectItem>
                    <SelectItem value="user">{t('roles.user')}</SelectItem>
                    <SelectItem value="guest">{t('roles.guest')}</SelectItem>
                </SelectContent>
            </Select>
            <Button
                type="button"
                variant="ghost"
                size="icon"
                className="justify-self-end text-muted-foreground hover:text-destructive"
                onClick={onRemove}
            >
                <Trash2 className="size-4" />
            </Button>
        </div>
    )
}

export function RoleMappingSection({ form }: { form: Form }) {
    const { t } = useTranslation(['common', 'forms'])
    const { fields, append, remove } = useFieldArray({ control: form.control, name: 'role_mapping' })

    return (
        <Section icon={Waypoints} title={t('forms:sso.roleMapping')} description={t('forms:sso.roleMappingHelp')}>
            {fields.length > 0 && (
                <div className={`hidden ${GRID} gap-2 px-1 text-xs font-medium text-muted-foreground sm:grid`}>
                    <span>{t('forms:sso.claim')}</span>
                    <span>{t('forms:sso.condition')}</span>
                    <span>{t('forms:sso.value')}</span>
                    <span>{t('fields.role')}</span>
                    <span />
                </div>
            )}

            {fields.map((rule, index) => (
                <RuleRow key={rule.id} form={form} index={index} onRemove={() => remove(index)} />
            ))}

            {fields.length === 0 && (
                <p className="rounded-lg border border-dashed px-4 py-6 text-center text-sm text-muted-foreground">
                    {t('forms:sso.noRules')}
                </p>
            )}

            <Button
                type="button"
                variant="outline"
                size="sm"
                className="w-full sm:w-auto"
                onClick={() => append({ claim: 'groups', operator: 'contains', value: '', role: 'user' })}
            >
                <Plus className="mr-2 size-4" />
                {t('forms:sso.addRule')}
            </Button>
        </Section>
    )
}
