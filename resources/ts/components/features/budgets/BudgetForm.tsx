import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Checkbox } from '@/components/ui/checkbox'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { budgetSchema, BudgetFormData } from '@/schemas'
import { useCategories, useFormValuesChange } from '@/hooks'
import {
    CurrencyIdField,
    FormActiveField,
    FormInputField,
    FormSubmitButton,
    FormWrapper,
    TagSelect,
} from '@/components/shared'
import { BudgetCategoriesField } from './form/BudgetCategoriesField'
import { BudgetPeriodFields } from './form/BudgetPeriodFields'

interface BudgetFormProps {
    defaultValues?: Partial<BudgetFormData>
    onSubmit: (data: BudgetFormData) => void
    onValuesChange?: (data: BudgetFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
}

export function BudgetForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
}: BudgetFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { data: categories } = useCategories('expense')

    const form = useForm<BudgetFormData>({
        resolver: schemaResolver<BudgetFormData>(budgetSchema),
        defaultValues: {
            name: '',
            amount: 0,
            currency_id: null,
            period: 'monthly',
            start_date: null,
            end_date: null,
            is_global: false,
            notify_at_percent: null,
            is_active: true,
            category_ids: [],
            tag_ids: [],
            ...defaultValues,
        },
    })

    const isGlobal = form.watch('is_global')
    const period = form.watch('period')

    useFormValuesChange(form, onValuesChange)

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                    <FormInputField
                        control={form.control}
                        name="name"
                        label={t('fields.name')}
                        inputProps={{ placeholder: t('forms:budgets.namePlaceholder') }}
                    />

                    <div className="grid grid-cols-2 gap-4">
                        <FormInputField
                            control={form.control}
                            name="amount"
                            className="min-w-0"
                            label={t('forms:budgets.limit')}
                            inputProps={{ type: 'number', step: '0.01', min: '0', placeholder: '30000' }}
                        />
                        <FormField
                            control={form.control}
                            name="currency_id"
                            render={({ field }) => (
                                <FormItem className="min-w-0">
                                    <FormLabel>{t('fields.currency')}</FormLabel>
                                    <CurrencyIdField
                                        value={field.value}
                                        onChange={field.onChange}
                                        placeholder={t('forms:budgets.baseCurrency')}
                                    />
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>

                    <BudgetPeriodFields control={form.control} isOneTime={period === 'one_time'} />

                    <FormField
                        control={form.control}
                        name="is_global"
                        render={({ field }) => (
                            <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
                                <FormControl>
                                    <Checkbox checked={field.value} onCheckedChange={field.onChange} />
                                </FormControl>
                                <div className="space-y-1 leading-none">
                                    <FormLabel>{t('forms:budgets.global')}</FormLabel>
                                    <FormDescription>{t('forms:budgets.globalHelp')}</FormDescription>
                                </div>
                            </FormItem>
                        )}
                    />

                    {!isGlobal && categories && categories.length > 0 && (
                        <BudgetCategoriesField control={form.control} categories={categories} />
                    )}

                    <FormField
                        control={form.control}
                        name="tag_ids"
                        render={({ field }) => (
                            <TagSelect
                                value={field.value ?? []}
                                onChange={field.onChange}
                                asFormItem
                                bordered
                                description={t('forms:budgets.tagsHelp')}
                            />
                        )}
                    />

                    <FormActiveField control={form.control} help={t('forms:budgets.activeHelp')} />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
        </FormWrapper>
    )
}
