import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { TransactionFormValues } from '@/schemas/transactions'
import { Transaction } from '@/types'
import { TransactionForm } from './TransactionForm'

const FORM_ID = 'transaction-form'

interface TransactionFormDialogProps {
    transaction?: Transaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: TransactionFormValues) => void
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
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
    defaultValues,
}: TransactionFormDialogProps) {
    const { t } = useTranslation('pages')
    const [preview, setPreview] = useState<ReactNode>(null)

    return (
        <EntityFormDialog
            entity={transaction}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={transaction ? t('transactions.editTitle') : t('transactions.createTitle')}
            className="sm:max-w-xl"
            headerExtra={preview}
            fallbackValues={defaultValues}
            toFormValues={toTransactionFormValues}
        >
            {({ formKey, formProps, isEdit }) => (
                <TransactionForm
                    key={`${formKey}-${open}`}
                    {...formProps}
                    open={open}
                    isEdit={isEdit}
                    originalAffectsBalance={transaction?.status === 'confirmed'}
                    onPreviewChange={setPreview}
                />
            )}
        </EntityFormDialog>
    )
}
