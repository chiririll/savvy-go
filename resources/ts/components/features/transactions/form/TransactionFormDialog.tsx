import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { toFormValues } from '@/lib/transaction-form-values'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { SpaceTransferInput, Transaction, TransactionInput } from '@/types'
import { TransactionForm } from './TransactionForm'
import type { TransactionFormDefaults } from './use-transaction-form'

const FORM_ID = 'transaction-form'

interface TransactionFormDialogProps {
    transaction?: Transaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: TransactionInput) => void
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
    onSubmitSpaceTransfer,
    isSubmitting,
    defaultValues,
}: TransactionFormDialogProps) {
    const { t } = useTranslation('pages')
    const toValues = useCallback((tx: Transaction) => toFormValues(tx), [])

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
            {({ formKey, formProps, isEdit }) => (
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
            )}
        </EntityFormDialog>
    )
}
