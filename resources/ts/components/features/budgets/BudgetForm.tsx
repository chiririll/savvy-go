import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
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
import { budgetSchema, BudgetFormData } from '@/schemas'
import { useCategories, useFormValuesChange } from '@/hooks'
import { Category } from '@/types'
import { localizeDefaultName } from '@/lib/localized-name'
import { CurrencyIdField, FieldHelp, FormActiveField, FormWrapper, TagSelect } from '@/components/shared'

interface BudgetFormProps {
    defaultValues?: Partial<BudgetFormData>
    onSubmit: (data: BudgetFormData) => void
    onValuesChange?: (data: BudgetFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
}

const periodOptions = ['weekly', 'monthly', 'yearly', 'one_time'] as const

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
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.name')}</FormLabel>
                            <FormControl>
                                <Input placeholder={t('forms:budgets.namePlaceholder')} {...field} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div className="grid grid-cols-2 gap-4">
                    <FormField
                        control={form.control}
                        name="amount"
                        render={({ field }) => (
                            <FormItem className="min-w-0">
                                <FormLabel>{t('forms:budgets.limit')}</FormLabel>
                                <FormControl>
                                    <Input
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        placeholder="30000"
                                        {...field}
                                    />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
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

                <div className="grid grid-cols-2 gap-4">
                    <FormField
                        control={form.control}
                        name="period"
                        render={({ field }) => (
                            <FormItem className="min-w-0">
                                <FormLabel className="flex min-h-4 items-center">{t('forms:budgets.period')}</FormLabel>
                                <Select onValueChange={field.onChange} value={field.value}>
                                    <FormControl>
                                        <SelectTrigger className="w-full">
                                            <SelectValue placeholder={t('forms:selectPeriod')} />
                                        </SelectTrigger>
                                    </FormControl>
                                    <SelectContent>
                                        {periodOptions.map((option) => (
                                            <SelectItem key={option} value={option}>
                                                {t(`forms:budgets.periods.${option}`)}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <FormField
                        control={form.control}
                        name="notify_at_percent"
                        render={({ field }) => (
                            <FormItem className="min-w-0">
                                <FormLabel className="flex min-h-4 items-center gap-1.5">
                                    {t('forms:budgets.notifyAt')}
                                    <FieldHelp>{t('forms:budgets.notifyHelp')}</FieldHelp>
                                </FormLabel>
                                <FormControl>
                                    <Input
                                        type="number"
                                        min="1"
                                        max="100"
                                        placeholder={t('forms:budgets.notifyPlaceholder')}
                                        {...field}
                                        value={field.value ?? ''}
                                        onChange={(e) => {
                                            const val = e.target.value
                                            field.onChange(val === '' ? null : Number(val))
                                        }}
                                    />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                </div>

                {period === 'one_time' && (
                    <div className="grid grid-cols-2 gap-4">
                        <FormField
                            control={form.control}
                            name="start_date"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('forms:budgets.startDate')}</FormLabel>
                                    <FormControl>
                                        <Input
                                            type="date"
                                            {...field}
                                            value={field.value || ''}
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="end_date"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('forms:budgets.endDate')}</FormLabel>
                                    <FormControl>
                                        <Input
                                            type="date"
                                            {...field}
                                            value={field.value || ''}
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                )}

                <FormField
                    control={form.control}
                    name="is_global"
                    render={({ field }) => (
                        <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
                            <FormControl>
                                <Checkbox
                                    checked={field.value}
                                    onCheckedChange={field.onChange}
                                />
                            </FormControl>
                            <div className="space-y-1 leading-none">
                                <FormLabel>{t('forms:budgets.global')}</FormLabel>
                                <FormDescription>
                                    {t('forms:budgets.globalHelp')}
                                </FormDescription>
                            </div>
                        </FormItem>
                    )}
                />

                {!isGlobal && categories && categories.length > 0 && (
                    <FormField
                        control={form.control}
                        name="category_ids"
                        render={() => (
                            <FormItem>
                                <FormLabel>{t('forms:budgets.categories')}</FormLabel>
                                <div className="grid grid-cols-2 gap-2 max-h-48 overflow-y-auto rounded-md border p-3">
                                    {categories.map((category: Category) => (
                                        <FormField
                                            key={category.id}
                                            control={form.control}
                                            name="category_ids"
                                            render={({ field }) => (
                                                <FormItem
                                                    key={category.id}
                                                    className="flex flex-row items-center space-x-2 space-y-0"
                                                >
                                                    <FormControl>
                                                        <Checkbox
                                                            checked={field.value?.includes(category.id)}
                                                            onCheckedChange={(checked) => {
                                                                const current = field.value || []
                                                                if (checked) {
                                                                    field.onChange([...current, category.id])
                                                                } else {
                                                                    field.onChange(
                                                                        current.filter((id) => id !== category.id)
                                                                    )
                                                                }
                                                            }}
                                                        />
                                                    </FormControl>
                                                    <FormLabel className="flex items-center gap-2 font-normal cursor-pointer">
                                                        <span
                                                            className="w-5 h-5 rounded flex items-center justify-center text-xs text-white"
                                                            style={{ backgroundColor: category.color ?? undefined }}
                                                        >
                                                            {category.icon}
                                                        </span>
                                                        {localizeDefaultName(category.name)}
                                                    </FormLabel>
                                                </FormItem>
                                            )}
                                        />
                                    ))}
                                </div>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
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

                <FormActiveField
                    control={form.control}
                    help={t('forms:budgets.activeHelp')}
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
