import { useTranslation } from 'react-i18next'
import { Form, FormControl, FormField, FormItem } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { TransactionFormValues } from '@/schemas/transactions'
import { useTransferTargets } from '@/hooks'
import { FormSubmitButton, FormWrapper, TransactionTypeTabs } from '@/components/shared'
import type { Transaction, TransactionInput } from '@/types'
import { BalanceAfter, PendingHint } from './TransactionBalanceHint'
import { PaymentRow } from './PaymentRow'
import { TransactionDetailsRow } from './TransactionDetailsRow'
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

    const { accounts, transactionType, dateRequired, isPendingDate, isUndated, balances, usedAccountIds } =
        useTransactionModel({ form, transaction, isEdit, originalAffectsBalance })

    const { handleSubmit, handleTypeChange, negativeBalanceDialog } = useTransactionHandlers({
        form,
        warnings: balances.warnings,
        dateRequired,
        accounts,
        onSubmit,
        onTypeChange,
    })

    // F4a has the one payment; the rows are a list so the next steps only add to it.
    const paymentIndexes = [0]

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

                    {paymentIndexes.map((index) => {
                        const account = accounts?.find((a) => a.id === Number(form.watch(`payments.${index}.account_id`)))
                        const balance = balances.rows.find((row) => row.account.id === account?.id)
                        return (
                            <PaymentRow
                                key={index}
                                control={form.control}
                                list="payments"
                                index={index}
                                currency={account?.currency}
                                excludeIds={usedAccountIds.filter((id) => id !== account?.id)}
                            >
                                {!isPendingDate && balance && <BalanceAfter preview={balance} />}
                            </PaymentRow>
                        )
                    })}
                    {isPendingDate && <PendingHint isUndated={isUndated} />}

                    <TransactionDetailsRow
                        control={form.control}
                        type={transactionType}
                        dateRequired={dateRequired}
                        pastOnly={originalAffectsBalance}
                    />

                    <TransactionTagsField control={form.control} />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
            {negativeBalanceDialog}
        </FormWrapper>
    )
}
