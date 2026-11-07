import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { CategorySelect, FormSubmitButton, FormWrapper } from '@/components/shared'
import { Form, FormField, FormItem } from '@/components/ui/form'
import { schemaResolver } from '@/lib/form-resolver'
import { buildDetailsInput, toDetailsValues, type TransactionPatch } from '@/lib/transaction-details'
import { linesNeedNotes } from '@/lib/transaction-lines'
import { formatCurrency } from '@/lib/utils'
import { transactionDetailsSchema, type TransactionDetailsValues } from '@/schemas/transaction-details'
import type { Transaction } from '@/types'
import { LineNoteField } from './LineNoteField'
import { LockedPayments, LockedReasons } from './LockedTransactionSummary'
import { TransactionDescriptionField } from './TransactionDescriptionField'
import { TransactionTagsField } from './TransactionTagsField'

interface TransactionDetailsFormProps {
    transaction: Transaction
    onSubmit: (data: TransactionPatch) => void
    formId?: string
    isSubmitting?: boolean
    hideSubmit?: boolean
    open?: boolean
}

/**
 * The form of a transaction that allows only its details: the title, the tags and the category and note of each
 * line. The rest (date, payments, the amounts and currencies of the lines) is shown as text, with the reason.
 */
export function TransactionDetailsForm({
    transaction,
    onSubmit,
    formId,
    isSubmitting,
    hideSubmit,
    open = true,
}: TransactionDetailsFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const defaults = useMemo(() => toDetailsValues(transaction), [transaction])
    const form = useForm<TransactionDetailsValues>({
        resolver: schemaResolver<TransactionDetailsValues>(transactionDetailsSchema),
        defaultValues: defaults,
    })

    useEffect(() => {
        if (open) {
            form.reset(defaults)
        }
    }, [open, defaults, form])

    const lines = [...transaction.lines].sort((a, b) => a.seq - b.seq)
    const type = transaction.type === 'income' ? 'income' : 'expense'

    return (
        <FormWrapper>
            <Form {...form}>
                <form
                    id={formId}
                    noValidate
                    onSubmit={form.handleSubmit((values) => onSubmit(buildDetailsInput(transaction, values)))}
                    className="space-y-4"
                >
                    <LockedReasons transaction={transaction} />
                    <TransactionDescriptionField control={form.control} />
                    <LockedPayments transaction={transaction} />

                    {lines.length > 0 && (
                        <div className="space-y-3" role="group" aria-label={t('forms:transactions.lines')}>
                            {lines.map((l, index) => {
                                const currency = transaction.payments.find(
                                    (p) => p.account?.currency?.id === l.currencyId,
                                )?.account?.currency
                                const showNote = linesNeedNotes(lines.length) || Boolean(l.note)
                                return (
                                    <div key={l.seq} className="space-y-1.5">
                                        <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-2 sm:gap-4">
                                            <FormField
                                                control={form.control}
                                                name={`lines.${index}.category_id`}
                                                render={({ field }) => (
                                                    <FormItem className="min-w-0">
                                                        <CategorySelect
                                                            value={field.value}
                                                            onChange={field.onChange}
                                                            type={type}
                                                            ariaLabel={t('common:fields.category')}
                                                        />
                                                    </FormItem>
                                                )}
                                            />
                                            <span
                                                className="tabular-nums text-sm text-muted-foreground"
                                                aria-label={t('forms:transactions.lockedLineAmount')}
                                            >
                                                {formatCurrency(l.amount, currency)}
                                            </span>
                                        </div>
                                        {showNote && (
                                            <FormField
                                                control={form.control}
                                                name={`lines.${index}.note`}
                                                render={({ field }) => (
                                                    <FormItem>
                                                        <LineNoteField
                                                            value={field.value ?? ''}
                                                            onChange={field.onChange}
                                                            onBlur={field.onBlur}
                                                            label={t('forms:transactions.lineNote')}
                                                        />
                                                    </FormItem>
                                                )}
                                            />
                                        )}
                                    </div>
                                )
                            })}
                        </div>
                    )}

                    <TransactionTagsField control={form.control} />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} />
                </form>
            </Form>
        </FormWrapper>
    )
}
