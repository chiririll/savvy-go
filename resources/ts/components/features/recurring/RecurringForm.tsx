import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
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
import { recurringSchema, RecurringFormData } from '@/schemas'
import { useAccounts, useCategories, useFormValuesChange, useTransactionPartyDefaults } from '@/hooks'
import { formatDateLocal } from '@/lib/utils'
import {
    CategorySelect,
    FormActiveField,
    FormEstimatedField,
    FormWrapper,
    MoneyAccountFields,
    TagSelect,
    TransactionTypeTabs,
} from '@/components/shared'
import { frequencyLabel, weekdayLabel } from '@/lib/labels'

interface RecurringFormProps {
    defaultValues?: Partial<RecurringFormData>
    onSubmit: (data: RecurringFormData) => void
    onValuesChange?: (data: RecurringFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
}

const frequencyOptions = ['daily', 'weekly', 'monthly', 'yearly'] as const
const weekdayValues = [0, 1, 2, 3, 4, 5, 6] as const

export function RecurringForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
}: RecurringFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { data: accounts } = useAccounts({ active: true, exclude_debts: true })
    const { data: categories } = useCategories()

    const form = useForm<RecurringFormData>({
        resolver: schemaResolver<RecurringFormData>(recurringSchema),
        defaultValues: {
            type: 'expense',
            account_id: 0,
            to_account_id: null,
            category_id: null,
            amount: 0,
            to_amount: null,
            is_estimated: false,
            description: '',
            frequency: 'monthly',
            interval: 1,
            day_of_week: new Date().getDay(),
            day_of_month: new Date().getDate(),
            start_date: formatDateLocal(),
            end_date: null,
            is_active: true,
            tag_ids: [],
            ...defaultValues,
        },
    })

    const type = form.watch('type')
    const frequency = form.watch('frequency')
    const accountId = form.watch('account_id')
    const toAccountId = form.watch('to_account_id')

    const isTransfer = type === 'transfer'
    const selectedAccount = accounts?.find(a => a.id === accountId)
    const selectedToAccount = accounts?.find(a => a.id === toAccountId)

    useTransactionPartyDefaults(form, accounts, categories)
    useFormValuesChange(form, onValuesChange)

    return (
        <FormWrapper>
        <Form {...form}>
            <form id={formId} onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="type"
                    render={({ field }) => (
                        <FormItem>
                            <TransactionTypeTabs
                                value={field.value}
                                onChange={(value) => {
                                    field.onChange(value)
                                    if (value === 'transfer') {
                                        form.setValue('category_id', null)
                                    } else {
                                        form.setValue('to_account_id', null)
                                        form.setValue('to_amount', null)
                                    }
                                }}
                            />
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <MoneyAccountFields
                    control={form.control}
                    isTransfer={isTransfer}
                    accountId={accountId}
                    fromCurrencySymbol={selectedAccount?.currency?.symbol}
                    toCurrencySymbol={selectedToAccount?.currency?.symbol}
                    toAmountPlaceholder={t('forms:recurring.toAmountPlaceholder')}
                />

                {!isTransfer && (
                    <FormField
                        control={form.control}
                        name="category_id"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('fields.category')}</FormLabel>
                                <CategorySelect
                                    value={field.value}
                                    onChange={field.onChange}
                                    type={type as 'income' | 'expense'}
                                />
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                )}

                {/* Description */}
                <FormField
                    control={form.control}
                    name="description"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.description')}</FormLabel>
                            <FormControl>
                                <Textarea
                                    placeholder={t('forms:recurring.descriptionPlaceholder')}
                                    className="resize-none h-20"
                                    {...field}
                                    value={field.value ?? ''}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                {/* Schedule Section */}
                <div className="space-y-4 pt-4 border-t">
                    <h3 className="font-medium">{t('forms:recurring.schedule')}</h3>

                    <div className="grid grid-cols-2 gap-4">
                        <FormField
                            control={form.control}
                            name="frequency"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('forms:recurring.frequency')}</FormLabel>
                                    <Select onValueChange={field.onChange} value={field.value}>
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            {frequencyOptions.map((option) => (
                                                <SelectItem key={option} value={option}>
                                                    {frequencyLabel(t, option)}
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
                            name="interval"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('forms:recurring.every')}</FormLabel>
                                    <FormControl>
                                        <Input type="number" min="1" max="365" {...field} />
                                    </FormControl>
                                    <FormDescription>
                                        {frequency === 'daily' && t('forms:recurring.days')}
                                        {frequency === 'weekly' && t('forms:recurring.weeks')}
                                        {frequency === 'monthly' && t('forms:recurring.months')}
                                        {frequency === 'yearly' && t('forms:recurring.years')}
                                    </FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>

                    {frequency === 'weekly' && (
                        <FormField
                            control={form.control}
                            name="day_of_week"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('forms:recurring.dayOfWeek')}</FormLabel>
                                    <Select
                                        onValueChange={(val) => field.onChange(Number(val))}
                                        value={field.value?.toString() ?? ''}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder={t('forms:selectDay')} />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            {weekdayValues.map((value) => (
                                                <SelectItem key={value} value={value.toString()}>
                                                    {weekdayLabel(t, value)}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    )}

                    {frequency === 'monthly' && (
                        <FormField
                            control={form.control}
                            name="day_of_month"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('forms:recurring.dayOfMonth')}</FormLabel>
                                    <FormControl>
                                        <Input
                                            type="number"
                                            min="1"
                                            max="31"
                                            {...field}
                                            value={field.value ?? ''}
                                            onChange={(e) => field.onChange(e.target.value ? Number(e.target.value) : null)}
                                        />
                                    </FormControl>
                                    <FormDescription>{t('forms:recurring.dayOfMonthHelp')}</FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    )}

                    <div className="grid grid-cols-2 gap-4">
                        <FormField
                            control={form.control}
                            name="start_date"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('forms:recurring.startDate')}</FormLabel>
                                    <FormControl>
                                        <Input type="date" {...field} />
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
                                    <FormLabel>{t('forms:recurring.endDate')}</FormLabel>
                                    <FormControl>
                                        <Input
                                            type="date"
                                            {...field}
                                            value={field.value ?? ''}
                                            onChange={(e) => field.onChange(e.target.value || null)}
                                        />
                                    </FormControl>
                                    <FormDescription>{t('forms:recurring.endDateHelp')}</FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                </div>

                <FormField
                    control={form.control}
                    name="tag_ids"
                    render={({ field }) => (
                        <TagSelect
                            value={field.value ?? []}
                            onChange={field.onChange}
                            asFormItem
                            bordered
                        />
                    )}
                />

                <FormActiveField
                    control={form.control}
                    help={t('forms:recurring.activeHelp')}
                />

                <FormEstimatedField
                    control={form.control}
                    help={t('forms:recurring.estimatedHelp')}
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
