import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Form, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { recurringSchema, RecurringFormData } from '@/schemas'
import { useAccounts, useCategories, useFormValuesChange, useTransactionPartyDefaults } from '@/hooks'
import { formatDateLocal } from '@/lib/utils'
import {
    CategorySelect,
    FormActiveField,
    FormEstimatedField,
    FormInputField,
    FormSubmitButton,
    FormWrapper,
    MoneyAccountFields,
    TagSelect,
    TransactionTypeTabs,
} from '@/components/shared'
import { ScheduleSection } from './ScheduleSection'

interface RecurringFormProps {
    defaultValues?: Partial<RecurringFormData>
    onSubmit: (data: RecurringFormData) => void
    onValuesChange?: (data: RecurringFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
}

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
    const selectedAccount = accounts?.find((a) => a.id === accountId)
    const selectedToAccount = accounts?.find((a) => a.id === toAccountId)

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

                    <FormInputField
                        control={form.control}
                        name="description"
                        label={t('fields.description')}
                        multiline
                        textareaProps={{
                            placeholder: t('forms:recurring.descriptionPlaceholder'),
                            className: 'resize-none h-20',
                        }}
                    />

                    <ScheduleSection control={form.control} frequency={frequency} />

                    <FormField
                        control={form.control}
                        name="tag_ids"
                        render={({ field }) => (
                            <TagSelect value={field.value ?? []} onChange={field.onChange} asFormItem bordered />
                        )}
                    />

                    <FormActiveField control={form.control} help={t('forms:recurring.activeHelp')} />

                    <FormEstimatedField control={form.control} help={t('forms:recurring.estimatedHelp')} />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
        </FormWrapper>
    )
}
