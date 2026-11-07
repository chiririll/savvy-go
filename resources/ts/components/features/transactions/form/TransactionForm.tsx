import { useFieldArray } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Form, FormControl, FormField, FormItem } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { TransactionFormValues } from '@/schemas/transactions'
import { useTransferTargets } from '@/hooks'
import { FormSubmitButton, FormWrapper, TransactionTypeTabs } from '@/components/shared'
import type { SpaceTransferInput, Transaction, TransactionInput } from '@/types'
import { PendingHint } from './TransactionBalanceHint'
import { ChangeSection } from './ChangeSection'
import { PaymentTotals, PaymentsSection } from './PaymentsSection'
import { TransactionDetailsRow } from './TransactionDetailsRow'
import { TransactionLines } from './TransactionLines'
import { TransactionTagsField } from './TransactionTagsField'
import { TransferAccounts } from './TransferAccounts'
import { TransferFeeSection } from './TransferFeeSection'
import { useTransactionForm, type TransactionFormDefaults } from './use-transaction-form'
import { useTransactionHandlers } from './use-transaction-handlers'
import { useTransactionModel } from './use-transaction-model'

interface TransactionFormProps {
    defaultValues?: TransactionFormDefaults
    onSubmit: (data: TransactionInput) => void
    /** A transfer to a linked space goes through the transfers API: called instead of onSubmit for it. */
    onSubmitSpaceTransfer?: (data: SpaceTransferInput) => void
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

export function TransactionForm({
    defaultValues,
    onSubmit,
    onSubmitSpaceTransfer,
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
        uncovered,
        nextLine,
        transactionType,
        isTransfer,
        isSpaceTransfer,
        destinationSpaceId,
        destinationSpaceAccounts,
        selectedAccount,
        selectedToAccount,
        rate,
        dateRequired,
        isPendingDate,
        isUndated,
        balances,
        usedAccountIds,
    } = useTransactionModel({ form, transaction, isEdit, originalAffectsBalance, open })

    const lines = useFieldArray({ control: form.control, name: 'lines' })
    const addLine = () => lines.append(nextLine())

    const { handleSubmit, handleTypeChange, negativeBalanceDialog } = useTransactionHandlers({
        form,
        warnings: balances.warnings,
        dateRequired,
        accounts,
        onSubmit,
        onSubmitSpaceTransfer,
        onTypeChange,
    })

    // Change is for what is paid; an income shows it only when a saved one has some.
    const showChange = !isTransfer && (transactionType === 'expense' || (form.watch('change')?.length ?? 0) > 0)

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} noValidate onSubmit={handleSubmit} className="space-y-4">
                    <TransactionTypeTabs
                        value={transactionType}
                        withSpaceTransfer={!isEdit && transferTargets.length > 0}
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

                    {isTransfer ? (
                        <TransferAccounts
                            isSpaceTransfer={isSpaceTransfer}
                            links={transferTargets}
                            destinationAccounts={destinationSpaceAccounts}
                            destinationSpaceId={destinationSpaceId}
                            from={selectedAccount}
                            to={selectedToAccount}
                            rate={rate}
                            balances={balances.rows}
                            showBalances={!isPendingDate}
                        />
                    ) : (
                        <>
                            <PaymentsSection
                                accounts={accounts}
                                usedAccountIds={usedAccountIds}
                                balances={balances.rows}
                                showBalances={!isPendingDate}
                            />
                            <PaymentTotals accounts={accounts} />
                        </>
                    )}
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

                    {!isTransfer && (
                        <TransactionLines
                            control={form.control}
                            fields={lines.fields}
                            remove={lines.remove}
                            type={transactionType === 'income' ? 'income' : 'expense'}
                            currencies={lineCurrencies}
                            coverage={coverage}
                        />
                    )}

                    {/* The transfers API keeps no tags. */}
                    {!isSpaceTransfer && <TransactionTagsField control={form.control} />}

                    {isTransfer && !isSpaceTransfer && <TransferFeeSection currency={selectedAccount?.currency} />}

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
