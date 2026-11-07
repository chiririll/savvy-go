import { useMemo } from 'react'
import { useForm, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { CircleAlert } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Form } from '@/components/ui/form'
import { FormActiveField, FormEstimatedField, FormSubmitButton, FormWrapper } from '@/components/shared'
import { schemaResolver } from '@/lib/form-resolver'
import { recurringSchema, type RecurringFormValues } from '@/schemas/recurring'
import type { TransactionFormValues } from '@/schemas/transactions'
import { useFormValuesChange } from '@/hooks'
import type { RecurringInput } from '@/types'
import { TransactionFormBody } from '@/components/features/transactions/form/TransactionFormBody'
import { useTransactionModel } from '@/components/features/transactions/form/use-transaction-model'
import { useTypeChange } from '@/components/features/transactions/form/use-type-change'
import { toRecurringDefaults, type RecurringFormDefaults } from './recurring-defaults'
import { ScheduleSection } from './ScheduleSection'
import { useRecurringSubmit } from './use-recurring-submit'

interface RecurringFormProps {
    defaultValues?: RecurringFormDefaults
    onSubmit: (data: RecurringInput) => void
    onValuesChange?: (data: RecurringFormValues) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
    /** Why the rule being edited stopped: saving it again clears it. */
    error?: string | null
    open?: boolean
}

/**
 * A rule: the transaction it makes (the body of the transaction form, without its date) and when it is due.
 * One amount may stay open for the occurrence to work out: the amount received of a transfer between currencies.
 */
export function RecurringForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
    error,
    open = true,
}: RecurringFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const formDefaults = useMemo(() => toRecurringDefaults(defaultValues), [defaultValues])

    const form = useForm<RecurringFormValues>({
        resolver: schemaResolver<RecurringFormValues>(recurringSchema),
        defaultValues: formDefaults,
    })
    // The values of the rule are the transaction form's own, plus the schedule: the body reads them as its own.
    const transactionForm = form as unknown as UseFormReturn<TransactionFormValues>
    const model = useTransactionModel({ form: transactionForm, open, template: true })
    const handleTypeChange = useTypeChange(transactionForm)
    useFormValuesChange(form, onValuesChange)

    const submit = useRecurringSubmit(form, model.accounts, onSubmit)

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} noValidate onSubmit={submit} className="space-y-4">
                    {error && (
                        <Alert variant="destructive">
                            <CircleAlert />
                            <AlertDescription>{t('forms:recurring.stopped', { reason: error })}</AlertDescription>
                        </Alert>
                    )}

                    <TransactionFormBody
                        form={transactionForm}
                        model={model}
                        onTypeChange={handleTypeChange}
                        transferTargets={[]}
                        withSpaceTransfer={false}
                        template
                    />

                    <ScheduleSection control={form.control} frequency={form.watch('frequency')} />

                    <FormActiveField control={form.control} help={t('forms:recurring.activeHelp')} />

                    <FormEstimatedField control={form.control} name="estimated" help={t('forms:recurring.estimatedHelp')} />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
        </FormWrapper>
    )
}
