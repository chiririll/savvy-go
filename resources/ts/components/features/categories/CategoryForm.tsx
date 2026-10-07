import { useTranslation } from 'react-i18next'
import { useFormValuesChange } from '@/hooks'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
} from '@/components/ui/form'
import { COLOR_OPTIONS } from '@/constants'
import { categorySchema, CategoryFormData } from '@/schemas'
import { TypeSelector } from './TypeSelector'
import { IconPicker } from './IconPicker'
import { ColorPicker } from './ColorPicker'
import { CategoryPreview } from './CategoryPreview'
import { FormWrapper } from '@/components/shared/FormWrapper'

interface CategoryFormProps {
    defaultValues?: Partial<CategoryFormData>
    onSubmit: (data: CategoryFormData) => void
    onValuesChange?: (data: CategoryFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
    isDefault?: boolean
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
}: CategoryFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const form = useForm<CategoryFormData>({
        resolver: zodResolver(categorySchema),
        defaultValues: {
            type: 'expense',
            icon: '🏠',
            color: COLOR_OPTIONS[9], // blue-400
            ...defaultValues,
        },
    })

    const watchedValues = form.watch()

    useFormValuesChange(form, onValuesChange)

    return (
        <FormWrapper>
        <Form {...form}>
            <form
                id={formId}
                onSubmit={form.handleSubmit(onSubmit)}
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
