import { Form } from '@/components/ui/form'
import { TransactionFormValues } from '@/schemas/transactions'
import { useTransferTargets } from '@/hooks'
import { FormSubmitButton, FormWrapper } from '@/components/shared'
import type { SpaceTransferInput, Transaction, TransactionInput } from '@/types'
import { TransactionFormBody } from './TransactionFormBody'
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
    const transferTargets = useTransferTargets()

    const { form } = useTransactionForm({ defaultValues, onValuesChange, originalAffectsBalance, open })

    const model = useTransactionModel({ form, transaction, isEdit, originalAffectsBalance, open })

    const { handleSubmit, handleTypeChange, negativeBalanceDialog } = useTransactionHandlers({
        form,
        warnings: model.balances.warnings,
        dateRequired: model.dateRequired,
        accounts: model.accounts,
        onSubmit,
        onSubmitSpaceTransfer,
        onTypeChange,
    })

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} noValidate onSubmit={handleSubmit} className="space-y-4">
                    <TransactionFormBody
                        form={form}
                        model={model}
                        onTypeChange={handleTypeChange}
                        transferTargets={transferTargets}
                        withSpaceTransfer={!isEdit && transferTargets.length > 0}
                        originalAffectsBalance={originalAffectsBalance}
                    />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
            {negativeBalanceDialog}
        </FormWrapper>
    )
}
