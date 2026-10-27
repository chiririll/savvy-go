import { useCallback, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { TransactionFormValues } from '@/schemas/transactions'
import { paymentFormValues } from '@/lib/payment-input'
import type { PaymentLeg } from '@/lib/split-payment'
import { Transaction, TransactionGroup } from '@/types'
import { TransactionForm } from './TransactionForm'

const FORM_ID = 'transaction-form'

interface TransactionFormDialogProps {
    transaction?: Transaction | null
    /** A saved split payment being changed; the form shows its legs as one. */
    group?: TransactionGroup | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: TransactionFormValues) => void
    onSubmitPayment?: (legs: PaymentLeg[]) => void
    isSubmitting?: boolean
    defaultValues?: Partial<TransactionFormValues>
}

export function toTransactionFormValues(transaction: Transaction): Partial<TransactionFormValues> {
    return {
        type: transaction.type as TransactionFormValues['type'],
        account_id: transaction.account.id,
        to_account_id: transaction.toAccount?.id ?? null,
        category_id: transaction.category?.id ?? null,
        amount: transaction.amount,
        to_amount: transaction.toAmount ?? null,
        is_estimated: transaction.isEstimated,
        description: transaction.description ?? '',
        date: transaction.date ?? '',
        items: transaction.items?.map((item) => ({
            name: item.name,
            quantity: item.quantity,
            price_per_unit: item.pricePerUnit,
        })) ?? [],
        tag_ids: transaction.tags?.map((tag) => tag.id) ?? [],
    }
}

export function TransactionFormDialog({
    transaction,
    group,
    open,
    onOpenChange,
    onSubmit,
    onSubmitPayment,
    isSubmitting,
    defaultValues,
}: TransactionFormDialogProps) {
    const { t } = useTranslation('pages')
    const [preview, setPreview] = useState<ReactNode>(null)
    // A payment is edited as its first leg, which is how the dialog tells a change from a new one.
    const entity = transaction ?? group?.legs[0] ?? null
    const toFormValues = useCallback(
        (leg: Transaction) => (group ? (paymentFormValues(group) ?? {}) : toTransactionFormValues(leg)),
        [group],
    )

    return (
        <EntityFormDialog
            entity={entity}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={
                group
                    ? t('transactions.editPaymentTitle')
                    : transaction
                      ? t('transactions.editTitle')
                      : t('transactions.createTitle')
            }
            className="sm:max-w-xl"
            headerExtra={preview}
            fallbackValues={defaultValues}
            toFormValues={toFormValues}
        >
            {({ formKey, formProps, isEdit }) => (
                <TransactionForm
                    key={`${formKey}-${open}`}
                    {...formProps}
                    open={open}
                    isEdit={isEdit}
                    onSubmitPayment={onSubmitPayment}
                    paymentEdit={Boolean(group)}
                    originalAffectsBalance={entity?.status === 'confirmed'}
                    onPreviewChange={setPreview}
                />
            )}
        </EntityFormDialog>
    )
}
