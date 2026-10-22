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
    FormDescription,
} from '@/components/ui/form'
import { FormWrapper } from '@/components/shared/form/FormWrapper'
import { localizeDefaultName, toStoredDefaultName } from '@/lib/localized-name'
import { tagSchema, TagFormData } from '@/schemas'

interface TagFormProps {
    defaultValues?: Partial<TagFormData>
    onSubmit: (data: TagFormData) => void
    onValuesChange?: (data: TagFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
}

export function TagForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
}: TagFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const form = useForm<TagFormData>({
        resolver: zodResolver(tagSchema),
        defaultValues: {
            name: defaultValues?.name ? localizeDefaultName(defaultValues.name, 'tags') : '',
        },
    })

    useFormValuesChange(form, onValuesChange)

    const submit = (data: TagFormData) =>
        onSubmit({ ...data, name: toStoredDefaultName(data.name, defaultValues?.name, 'tags') })

    return (
        <FormWrapper>
        <Form {...form}>
            <form id={formId} onSubmit={form.handleSubmit(submit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.name')}</FormLabel>
                            <div className="flex items-center gap-2">
                                <span className="text-muted-foreground text-lg" aria-hidden>#</span>
                                <FormControl>
                                    <Input
                                        placeholder={t('forms:tags.namePlaceholder')}
                                        {...field}
                                    />
                                </FormControl>
                            </div>
                            <FormDescription>
                                {t('forms:tags.nameHelp')}
                            </FormDescription>
                            <FormMessage />
                        </FormItem>
                    )}
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
