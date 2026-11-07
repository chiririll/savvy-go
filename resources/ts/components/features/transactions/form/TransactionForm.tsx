import { useFieldArray } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Form, FormControl, FormField, FormItem } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { TransactionFormValues } from '@/schemas/transactions'
import { useTransferTargets } from '@/hooks'
import { FormSubmitButton, FormWrapper, TransactionTypeTabs } from '@/components/shared'
import { linesToAdd } from '@/lib/transaction-lines'
import { formatCurrency } from '@/lib/utils'
import type { Transaction, TransactionInput } from '@/types'
import { PendingHint } from './TransactionBalanceHint'
import { ChangeSection } from './ChangeSection'
import { PaymentTotals, PaymentsSection } from './PaymentsSection'
import { TransactionDetailsRow } from './TransactionDetailsRow'
import { TransactionLines } from './TransactionLines'
import { TransactionTagsField } from './TransactionTagsField'
import { useTransactionForm, type TransactionFormDefaults } from './use-transaction-form'
import { useTransactionHandlers } from './use-transaction-handlers'
import { useTransactionModel } from './use-transaction-model'

interface TransactionFormProps {
    defaultValues?: TransactionFormDefaults
    onSubmit: (data: TransactionInput) => void
    onTypeChange?: (type: TransactionFormValues['type']) => void
    onValuesChange?: (data: TransactionFormValues) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
    isEdit?: boolean
    /** The transaction being edited. */
    transaction?: Transaction | null
    originalAffectsBalance?: boolean
    open?: boolean
}

// Transfers come back with their own form (F4d).
const TRANSFER_KINDS = ['transfer', 'transfer_out'] as const

export function TransactionForm({
    defaultValues,
    onSubmit,
    onTypeChange,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
    isEdit,
    transaction,
    originalAffectsBalance,
    open = true,
}: TransactionFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const transferTargets = useTransferTargets()

    const { form } = useTransactionForm({ defaultValues, onValuesChange, originalAffectsBalance, open })

    const {
        accounts,
        coverage,
        lineCurrencies,
        hasLines,
        transactionType,
        dateRequired,
        isPendingDate,
        isUndated,
        balances,
        usedAccountIds,
    } = useTransactionModel({ form, transaction, isEdit, originalAffectsBalance })

    const lines = useFieldArray({ control: form.control, name: 'lines' })
    const addLine = () => lines.append(linesToAdd(form.getValues(), accounts ?? []))
    const defaultCategoryId = form.watch('default_category_id')
    // What the lines leave uncovered goes to the default category: said where that category is chosen.
    const uncovered =
        hasLines && coverage.remainder > 0 && coverage.currency
            ? defaultCategoryId
                ? t('forms:transactions.uncovered', { amount: formatCurrency(coverage.remainder, coverage.currency) })
                : t('forms:transactions.uncoveredNoCategory', {
                      amount: formatCurrency(coverage.remainder, coverage.currency),
                  })
            : null

    const { handleSubmit, handleTypeChange, negativeBalanceDialog } = useTransactionHandlers({
        form,
        warnings: balances.warnings,
        dateRequired,
        accounts,
        onSubmit,
        onTypeChange,
    })

    // Change is for what is paid; an income shows it only when a saved one has some.
    const showChange = transactionType === 'expense' || (form.watch('change')?.length ?? 0) > 0

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} noValidate onSubmit={handleSubmit} className="space-y-4">
                    <TransactionTypeTabs
                        value={transactionType}
                        withSpaceTransfer={!isEdit && transferTargets.length > 0}
                        disabledKinds={TRANSFER_KINDS}
                        onChange={handleTypeChange}
                    />

                    <FormField
                        control={form.control}
                        name="description"
                        render={({ field }) => (
                            <FormItem>
                                <FormControl>
                                    <Input
                                        {...field}
                                        value={field.value ?? ''}
                                        maxLength={500}
                                        aria-label={t('fields.description')}
                                        placeholder={t('forms:transactions.titlePlaceholder')}
                                    />
                                </FormControl>
                            </FormItem>
                        )}
                    />

                    <PaymentsSection
                        accounts={accounts}
                        usedAccountIds={usedAccountIds}
                        balances={balances.rows}
                        showBalances={!isPendingDate}
                    />
                    <PaymentTotals accounts={accounts} />
                    {isPendingDate && <PendingHint isUndated={isUndated} />}

                    <TransactionDetailsRow
                        control={form.control}
                        type={transactionType}
                        dateRequired={dateRequired}
                        pastOnly={originalAffectsBalance}
                        category={{
                            hasLines,
                            locked: coverage.remainder === 0,
                            uncovered,
                            onAddLine: addLine,
                        }}
                    />

                    <TransactionLines
                        control={form.control}
                        fields={lines.fields}
                        remove={lines.remove}
                        type={transactionType}
                        currencies={lineCurrencies}
                        coverage={coverage}
                    />

                    <TransactionTagsField control={form.control} />

                    {showChange && (
                        <ChangeSection
                            accounts={accounts}
                            usedAccountIds={usedAccountIds}
                            balances={balances.rows}
                            showBalances={!isPendingDate}
                        />
                    )}

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
            {negativeBalanceDialog}
        </FormWrapper>
    )
}
