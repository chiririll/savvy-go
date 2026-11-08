import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { formModeOf, type TransactionPatch } from '@/lib/transaction-details'
import { toFormValues } from '@/lib/transaction-form-values'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { SpaceTransferInput, Transaction, TransactionInput } from '@/types'
import { TransactionDetailsForm } from './TransactionDetailsForm'
import { TransactionForm } from './TransactionForm'
import type { TransactionFormDefaults } from './use-transaction-form'

const FORM_ID = 'transaction-form'

interface TransactionFormDialogProps {
    transaction?: Transaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: TransactionInput) => void
    /** A transaction that allows only its details sends just those. */
    onSubmitDetails?: (data: TransactionPatch) => void
    /** A transfer to a linked space goes through the transfers API. */
    onSubmitSpaceTransfer?: (data: SpaceTransferInput) => void
    isSubmitting?: boolean
    defaultValues?: TransactionFormDefaults
}

export function TransactionFormDialog({
    transaction,
    open,
    onOpenChange,
    onSubmit,
    onSubmitDetails,
    onSubmitSpaceTransfer,
    isSubmitting,
    defaultValues,
}: TransactionFormDialogProps) {
    const { t } = useTranslation('pages')
    // A transaction that allows only its details is edited in the restricted form.
    const detailsOnly = transaction ? formModeOf(transaction) === 'details' : false
    const toValues = useCallback(
        (tx: Transaction) => (formModeOf(tx) === 'details' ? {} : toFormValues(tx)),
        [],
    )

    return (
        <EntityFormDialog<Transaction, TransactionInput, TransactionFormValues>
            entity={transaction}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={transaction ? t('transactions.editTitle') : t('transactions.createTitle')}
            className="sm:max-w-xl"
            fallbackValues={defaultValues as Partial<TransactionFormValues> | undefined}
            toFormValues={toValues}
        >
            {({ formKey, formProps, isEdit }) =>
                detailsOnly && transaction ? (
                    <TransactionDetailsForm
                        key={`${formKey}-${open}`}
                        transaction={transaction}
                        onSubmit={onSubmitDetails ?? (() => {})}
                        formId={formProps.formId}
                        isSubmitting={formProps.isSubmitting}
                        hideSubmit
                        open={open}
                    />
                ) : (
                    <TransactionForm
                        key={`${formKey}-${open}`}
                        {...formProps}
                        onSubmitSpaceTransfer={onSubmitSpaceTransfer}
                        defaultValues={formProps.defaultValues}
                        open={open}
                        isEdit={isEdit}
                        transaction={transaction}
                        originalAffectsBalance={transaction?.status === 'confirmed'}
                    />
                )
            }
        </EntityFormDialog>
    )
}
