import { useTranslation } from 'react-i18next'
import { useFormValuesChange } from '@/hooks'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
    FormDescription,
} from '@/components/ui/form'
import { COLOR_OPTIONS } from '@/constants'
import { categorySchema, CategoryFormData } from '@/schemas'
import { defaultNameHint, defaultNameKey, localizeDefaultName, toStoredDefaultName } from '@/lib/format/localized-name'
import { TypeSelector } from './TypeSelector'
import { IconPicker } from './IconPicker'
import { ColorPicker } from './ColorPicker'
import { CategorySelect } from '@/components/shared/pickers/CategorySelect'
import { CategoryPreview } from './CategoryPreview'
import { FormWrapper } from '@/components/shared/form/FormWrapper'

interface CategoryFormProps {
    defaultValues?: Partial<CategoryFormData>
    onSubmit: (data: CategoryFormData) => void
    onValuesChange?: (data: CategoryFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
    isDefault?: boolean
    /** Top-level categories the category may go under; empty hides the field. */
    parentOptions?: { id: number; name: string; type: string }[]
}

export function CategoryForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
    isDefault,
    parentOptions = [],
}: CategoryFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const form = useForm<CategoryFormData>({
        resolver: zodResolver(categorySchema),
        defaultValues: {
            type: 'expense',
            icon: '🏠',
            color: COLOR_OPTIONS[9], // blue-400
            ...defaultValues,
            name: defaultValues?.name ? localizeDefaultName(defaultValues.name) : '',
        },
    })

    const watchedValues = form.watch()
    const storedName = toStoredDefaultName(watchedValues.name ?? '', defaultValues?.name)
    const isDefaultName = Boolean(defaultNameKey(storedName))

    useFormValuesChange(form, onValuesChange)

    return (
        <FormWrapper>
        <Form {...form}>
            <form
                id={formId}
                onSubmit={form.handleSubmit((data) => onSubmit({
                    ...data,
                    name: toStoredDefaultName(data.name, defaultValues?.name),
                }))}
                className="space-y-4"
            >
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.name')}</FormLabel>
                            <FormControl>
                                <Input placeholder={t('forms:categories.namePlaceholder')} {...field} />
                            </FormControl>
                            {defaultNameHint(storedName) && (
                                <FormDescription>{defaultNameHint(storedName)}</FormDescription>
                            )}
                            {isDefaultName && (
                                <FormDescription>
                                    {t('forms:categories.defaultNameHelp')}
                                </FormDescription>
                            )}
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="type"
                    render={({ field }) => (
                        <FormItem>
                            <TypeSelector
                                value={field.value}
                                onChange={field.onChange}
                                error={form.formState.errors.type?.message}
                                disabled={isDefault}
                                disabledHelp={t('forms:categories.typeDefaultHelp')}
                            />
                        </FormItem>
                    )}
                />

                {parentOptions.some((p) => p.type === watchedValues.type) && (
                    <FormField
                        control={form.control}
                        name="parentId"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('forms:categories.parent')}</FormLabel>
                                <CategorySelect
                                    plain={false}
                                    type={watchedValues.type}
                                    onlyIds={parentOptions.map((p) => p.id)}
                                    value={field.value}
                                    onChange={field.onChange}
                                    noneLabel={t('forms:categories.noParent')}
                                    onClear={() => field.onChange(null)}
                                    sortByPopularity={false}
                                />
                                <FormDescription>{t('forms:categories.parentHelp')}</FormDescription>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                )}

                <FormField
                    control={form.control}
                    name="icon"
                    render={({ field }) => (
                        <FormItem>
                            <IconPicker
                                value={field.value}
                                onChange={field.onChange}
                                error={form.formState.errors.icon?.message}
                            />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="color"
                    render={({ field }) => (
                        <FormItem>
                            <ColorPicker
                                value={field.value}
                                onChange={field.onChange}
                                error={form.formState.errors.color?.message}
                            />
                        </FormItem>
                    )}
                />

                <CategoryPreview
                    name={watchedValues.name}
                    icon={watchedValues.icon}
                    color={watchedValues.color}
                />

                {!hideSubmit && (
                    <Button type="submit" disabled={isSubmitting} className="w-full">
                        {isSubmitting ? t('actions.saving') : (submitLabel ?? t('actions.save'))}
                    </Button>
                )}
            </form>
        </Form>
        </FormWrapper>
    )
}
