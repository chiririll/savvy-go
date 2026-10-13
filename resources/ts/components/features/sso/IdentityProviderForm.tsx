import { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm, useFieldArray } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Plus, Trash2, Plug, UserCog, Waypoints, Copy } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
    FormDescription,
} from '@/components/ui/form'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { FormWrapper } from '@/components/shared/FormWrapper'
import { identityProviderSchema, IdentityProviderFormValues } from '@/schemas/sso'
import { useSettings } from '@/hooks/use-settings'
import { useSsoPresets } from '@/hooks/use-sso'
import { BrandIcon, brandVars } from './BrandIcon'
import { ssoFieldLabel, ssoFieldPlaceholder, ssoPresetLabel } from './presetMeta'
import type { LucideIcon } from 'lucide-react'
import type { PresetField } from '@/types/sso'

interface IdentityProviderFormProps {
    defaultValues?: Partial<IdentityProviderFormValues>
    onSubmit: (data: IdentityProviderFormValues) => void
    isSubmitting?: boolean
    submitLabel?: string
    isEdit?: boolean
    presetLocked?: boolean
    previewPreset?: string
}

const TOGGLE_NAMES: (keyof IdentityProviderFormValues)[] = ['enabled', 'allow_jit', 'link_by_email', 'sync_role_on_login']

function CopyableUrl({ label, value }: { label: string; value: string }) {
    const { t } = useTranslation('forms')

    return (
        <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground">{label}</p>
            <div className="flex items-center gap-2 rounded-md border bg-background px-3 py-2">
                <code className="flex-1 truncate font-mono text-xs">{value}</code>
                <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-7 shrink-0 text-muted-foreground"
                    onClick={() => {
                        navigator.clipboard?.writeText(value)
                        toast.success(t('sso.copied'))
                    }}
                >
                    <Copy className="size-3.5" />
                </Button>
            </div>
        </div>
    )
}

function RedirectUrls({ slug, protocol }: { slug: string; protocol: string }) {
    const { t } = useTranslation('forms')
    const { data: settings } = useSettings()
    // The server builds redirect URIs from the public URL, not from the address this page is opened at.
    const appUrl = settings?.app_url?.trim().replace(/\/+$/, '')
    const base = `${appUrl}/api/auth/sso/${slug}`

    if (!appUrl) {
        return (
            <div className="rounded-lg border border-dashed bg-muted/30 p-4">
                <p className="text-xs text-muted-foreground">{t('sso.needAppUrl')}</p>
            </div>
        )
    }

    return (
        <div className="space-y-3 rounded-lg border border-dashed bg-muted/30 p-4">
            <p className="text-xs text-muted-foreground">{t('sso.registerUrls')}</p>
            {protocol === 'saml' ? (
                <>
                    <CopyableUrl label={t('sso.acsUrl')} value={`${base}/acs`} />
                    <CopyableUrl label={t('sso.spEntityId')} value={`${base}/metadata`} />
                </>
            ) : (
                <CopyableUrl label={t('sso.redirectUrl')} value={`${base}/callback`} />
            )}
        </div>
    )
}

function Section({ icon: Icon, title, description, children }: { icon: LucideIcon; title: string; description?: string; children: ReactNode }) {
    return (
        <section className="rounded-xl border bg-card shadow-sm">
            <header className="flex items-start gap-3 border-b px-5 py-4">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border bg-muted/50 text-muted-foreground">
                    <Icon className="size-4" />
                </div>
                <div className="space-y-0.5">
                    <h3 className="text-sm font-semibold leading-none">{title}</h3>
                    {description && <p className="text-xs text-muted-foreground">{description}</p>}
                </div>
            </header>
            <div className="space-y-5 p-5">{children}</div>
        </section>
    )
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
    const { t } = useTranslation(['common', 'forms'])
    const toggleLabels: Record<string, { label: string; description: string }> = {
        enabled: { label: t('forms:sso.enabled'), description: t('forms:sso.enabledHelp') },
        allow_jit: { label: t('forms:sso.allowJit'), description: t('forms:sso.allowJitHelp') },
        link_by_email: { label: t('forms:sso.linkByEmail'), description: t('forms:sso.linkByEmailHelp') },
        sync_role_on_login: { label: t('forms:sso.syncRole'), description: t('forms:sso.syncRoleHelp') },
    }
    const { data: presets } = useSsoPresets()

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

    const { fields: ruleFields, append, remove } = useFieldArray({
        control: form.control,
        name: 'role_mapping',
    })

    const selectedKey = form.watch('preset')
    const selectedPreset = presets?.find((p) => p.key === selectedKey)
    const previewName = form.watch('name')
    const previewEnabled = form.watch('enabled')
    const slug = form.watch('slug')

    const renderField = (field: PresetField) => (
        <FormField
            key={field.key}
            control={form.control}
            name={`fields.${field.key}` as const}
            render={({ field: f }) => (
                <FormItem>
                    <FormLabel>{ssoFieldLabel(selectedKey, field, t)}</FormLabel>
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
                                placeholder={isEdit && field.secret ? t('forms:sso.secretKeepPlaceholder') : ssoFieldPlaceholder(field, t)}
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

    const formBody = (
        <FormWrapper>
            <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className={`space-y-6 ${previewPreset ? 'w-full' : 'max-w-2xl'}`}>
                    <Section icon={Plug} title={t('forms:sso.connection')} description={t('forms:sso.connectionHelp')}>
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

                        {selectedPreset && selectedPreset.fields.map(renderField)}

                        {selectedPreset && slug && (
                            <RedirectUrls slug={slug} protocol={selectedPreset.protocol} />
                        )}
                    </Section>

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
                            {TOGGLE_NAMES.map((name) => (
                                <FormField
                                    key={name}
                                    control={form.control}
                                    name={name}
                                    render={({ field }) => (
                                        <FormItem className="flex items-center justify-between gap-4 px-4 py-3">
                                            <div className="space-y-0.5">
                                                <FormLabel className="cursor-pointer">{toggleLabels[name].label}</FormLabel>
                                                <FormDescription>{toggleLabels[name].description}</FormDescription>
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

                    <Section icon={Waypoints} title={t('forms:sso.roleMapping')} description={t('forms:sso.roleMappingHelp')}>
                        {ruleFields.length > 0 && (
                            <div className="hidden grid-cols-[1fr_140px_1fr_160px_auto] gap-2 px-1 text-xs font-medium text-muted-foreground sm:grid">
                                <span>{t('forms:sso.claim')}</span>
                                <span>{t('forms:sso.condition')}</span>
                                <span>{t('forms:sso.value')}</span>
                                <span>{t('fields.role')}</span>
                                <span />
                            </div>
                        )}

                        {ruleFields.map((rule, index) => (
                            <div
                                key={rule.id}
                                className="grid grid-cols-1 gap-2 rounded-lg border bg-muted/30 p-2 sm:grid-cols-[1fr_140px_1fr_160px_auto] sm:items-center sm:bg-transparent sm:border-0 sm:p-0"
                            >
                                <Input
                                    placeholder={t('forms:sso.claimPlaceholder')}
                                    className="font-mono text-sm"
                                    {...form.register(`role_mapping.${index}.claim`)}
                                />
                                <Select
                                    value={form.watch(`role_mapping.${index}.operator`)}
                                    onValueChange={(v) => form.setValue(`role_mapping.${index}.operator`, v as 'equals' | 'contains' | 'one_of')}
                                >
                                    <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
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
                                    <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
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
                                    onClick={() => remove(index)}
                                >
                                    <Trash2 className="size-4" />
                                </Button>
                            </div>
                        ))}

                        {ruleFields.length === 0 && (
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

                    <Button type="submit" disabled={isSubmitting} className="w-full">
                        {isSubmitting ? t('actions.saving') : (submitLabel ?? t('actions.save'))}
                    </Button>
                </form>
            </Form>
        </FormWrapper>
    )

    if (!previewPreset) {
        return formBody
    }

    return (
        <div className="grid max-w-5xl gap-6 lg:grid-cols-[minmax(0,1fr)_300px] lg:items-start">
            {formBody}
            <Card className="lg:sticky lg:top-6">
                <CardHeader>
                    <CardTitle className="text-sm">{t('forms:sso.loginPreview')}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                    <div
                        style={brandVars(previewPreset)}
                        className="rounded-lg border bg-[color-mix(in_oklab,var(--brand)_5%,var(--background))] p-4"
                    >
                        <div className="inline-flex h-10 w-full items-center justify-center gap-2.5 rounded-md border bg-background px-4 text-sm font-medium shadow-sm">
                            <BrandIcon preset={previewPreset} className="size-4" />
                            {t('forms:sso.continueWith', { name: previewName || t('forms:sso.providerFallback') })}
                        </div>
                    </div>
                    <div className="flex items-center gap-2 text-xs">
                        <span className={`size-1.5 rounded-full ${previewEnabled ? 'bg-emerald-500' : 'bg-muted-foreground/40'}`} />
                        <span className="text-muted-foreground">
                            {previewEnabled ? t('forms:sso.visibleOnLogin') : t('forms:sso.hiddenOnLogin')}
                        </span>
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}
