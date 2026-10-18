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
import { useTransactionForm } from './form/use-transaction-form'
import { useTransactionHandlers } from './form/use-transaction-handlers'
import { useTransactionModel } from './form/use-transaction-model'
import { TransactionDetailsRow } from './form/TransactionDetailsRow'
import { TransactionTagsField } from './form/TransactionTagsField'
import { SpaceTransferTargetField } from './form/SpaceTransferTargetField'

interface TransactionFormProps {
    defaultValues?: Partial<TransactionFormValues>
    onSubmit: (data: TransactionFormValues) => void
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
        transactionType,
        accountId,
        isTransfer,
        isSpaceTransfer,
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
        onSubmit,
        onTypeChange,
    })

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} noValidate onSubmit={handleSubmit} className="space-y-6">
                    <TransactionTypeTabs
                        value={transactionType}
                        withSpaceTransfer={!isEdit && transferTargets.length > 0}
                        onChange={handleTypeChange}
                    />

                    {!onPreviewChange && balanceHint}

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
                        toAmountReadOnly={rate.sameCurrency}
                        onSwapAccounts={rate.keepNextPair}
                    />

                    {(isPendingDate || (isEdit && !originalAffectsBalance)) && !hasItems && !isSpaceTransfer && (
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

                    {!isTransfer && <TransactionFormItems form={form} currency={selectedAccount?.currency} />}

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
            {negativeBalanceDialog}
        </FormWrapper>
    )
}
