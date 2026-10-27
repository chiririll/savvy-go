import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Form } from '@/components/ui/form'
import { TransactionFormValues } from '@/schemas/transactions'
import { useTransferTargets } from '@/hooks'
import {
    FormEstimatedField,
    FormInputField,
    FormSubmitButton,
    FormWrapper,
    MoneyAccountFields,
    TransactionTypeTabs,
} from '@/components/shared'
import { TransactionFormItems } from './TransactionFormItems'
import { useTransactionForm } from './use-transaction-form'
import { useTransactionHandlers } from './use-transaction-handlers'
import { useTransactionModel } from './use-transaction-model'
import { TransactionDetailsRow } from './TransactionDetailsRow'
import { TransactionTagsField } from './TransactionTagsField'
import { SpaceTransferTargetField } from './SpaceTransferTargetField'
import { PaymentSection } from './PaymentSection'
import type { PaymentLeg } from '@/lib/split-payment'

interface TransactionFormProps {
    defaultValues?: Partial<TransactionFormValues>
    onSubmit: (data: TransactionFormValues) => void
    /** Called instead of onSubmit for an expense paid in several parts. */
    onSubmitPayment?: (legs: PaymentLeg[]) => void
    /** A saved split payment is being changed: it stays an expense paid in parts. */
    paymentEdit?: boolean
    onTypeChange?: (type: TransactionFormValues['type']) => void
    onValuesChange?: (data: TransactionFormValues) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
    isEdit?: boolean
    originalAffectsBalance?: boolean
    onPreviewChange?: (preview: ReactNode) => void
    open?: boolean
}

export function TransactionForm({
    defaultValues,
    onSubmit,
    onSubmitPayment,
    paymentEdit,
    onTypeChange,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
    isEdit,
    originalAffectsBalance,
    onPreviewChange,
    open = true,
}: TransactionFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const transferTargets = useTransferTargets()

    const { form, schemaOptionsRef } = useTransactionForm({
        defaultValues,
        onValuesChange,
        originalAffectsBalance,
        open,
    })

    const {
        accounts,
        transactionType,
        accountId,
        isTransfer,
        isSpaceTransfer,
        isSplit,
        paymentMode,
        hasItems,
        dateRequired,
        isPendingDate,
        destinationSpaceId,
        destinationSpaceAccounts,
        selectedAccount,
        selectedToAccount,
        rate,
        balances,
        balanceHint,
    } = useTransactionModel({
        form,
        schemaOptionsRef,
        defaultValues,
        isEdit,
        originalAffectsBalance,
        onPreviewChange,
        open,
    })

    const { handleSubmit, handleTypeChange, negativeBalanceDialog } = useTransactionHandlers({
        form,
        rate,
        warnings: balances.warnings,
        dateRequired,
        isSpaceTransfer,
        isSplit,
        accounts,
        onSubmit,
        onSubmitPayment,
        onTypeChange,
    })

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} noValidate onSubmit={handleSubmit} className="space-y-6">
                    {!paymentEdit && (
                        <TransactionTypeTabs
                            value={transactionType}
                            withSpaceTransfer={!isEdit && transferTargets.length > 0}
                            onChange={handleTypeChange}
                        />
                    )}

                    {!onPreviewChange && !isSplit && balanceHint}

                    {isSpaceTransfer && (
                        <SpaceTransferTargetField
                            links={transferTargets}
                            noAccounts={destinationSpaceAccounts?.length === 0}
                        />
                    )}

                    <MoneyAccountFields
                        control={form.control}
                        isTransfer={isTransfer}
                        toSpaceId={destinationSpaceId}
                        accountId={accountId}
                        fromCurrencySymbol={selectedAccount?.currency?.symbol}
                        toCurrencySymbol={selectedToAccount?.currency?.symbol}
                        amountDisabled={hasItems}
                        amountLabel={
                            isSplit
                                ? paymentMode === 'change'
                                    ? t('forms:transactions.payment.amount.change')
                                    : t('forms:transactions.payment.amount.several')
                                : undefined
                        }
                        toAmountReadOnly={rate.sameCurrency}
                        onSwapAccounts={rate.keepNextPair}
                    />

                    {transactionType === 'expense' && onSubmitPayment && (
                        <PaymentSection form={form} paymentEdit={paymentEdit} />
                    )}

                    {(isPendingDate || (isEdit && !originalAffectsBalance)) &&
                        !hasItems &&
                        !isSpaceTransfer &&
                        !isSplit && (
                        <FormEstimatedField control={form.control} help={t('forms:transactions.estimatedHelp')} />
                    )}

                    <TransactionDetailsRow
                        control={form.control}
                        type={transactionType}
                        rate={rate}
                        fromCode={selectedAccount?.currency?.code}
                        toCode={selectedToAccount?.currency?.code}
                        dateRequired={dateRequired}
                        pastOnly={originalAffectsBalance}
                    />

                    <FormInputField
                        control={form.control}
                        name="description"
                        label={t('fields.description')}
                        multiline
                        textareaProps={{
                            placeholder: t('forms:transactions.notesPlaceholder'),
                            className: 'resize-none h-20',
                        }}
                    />

                    {/* The transfers API keeps no tags. */}
                    {!isSpaceTransfer && <TransactionTagsField control={form.control} />}

                    {!isTransfer && !isSplit && (
                        <TransactionFormItems form={form} currency={selectedAccount?.currency} />
                    )}

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
            {negativeBalanceDialog}
        </FormWrapper>
    )
}
