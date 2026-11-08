import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { AccountFormValues, AccountFormData, encodeAccountCurrency } from '@/schemas'
import { Account, FORM_ACCOUNT_TYPES } from '@/types'
import { AccountForm } from './AccountForm'

const FORM_ID = 'account-form'

interface AccountFormDialogProps {
    account?: Account | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: AccountFormData) => void
    isSubmitting?: boolean
}

export function AccountFormDialog({
    account,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
}: AccountFormDialogProps) {
    const { t } = useTranslation('pages')
    // A saved account that is not a card yet and is saved as one is converted.
    const submit = (data: AccountFormData) =>
        onSubmit(account && account.type !== 'credit' && data.type === 'credit' ? { ...data, convert: true } : data)

    return (
        <EntityFormDialog<Account, AccountFormData, AccountFormValues>
            entity={account}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={submit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={account ? t('accounts.editTitle') : t('accounts.createTitle')}
            description={t('accounts.description')}
            toFormValues={(item) => ({
                name: item.name,
                type: FORM_ACCOUNT_TYPES.find((type) => type === item.type) ?? 'bank',
                currency: encodeAccountCurrency({ id: item.currency?.id }),
                initial_balance: item.initialBalance,
                is_active: item.isActive,
                ...(item.creditCard && {
                    limit: item.creditCard.limit,
                    statement_day: item.creditCard.statementDay ?? undefined,
                    grace_days: item.creditCard.graceDays,
                }),
            })}
        >
            {({ formKey, formProps, isEdit }) => (
                <AccountForm
                    key={formKey}
                    isEdit={isEdit}
                    defaultValues={formProps.defaultValues}
                    onSubmit={submit}
                    onValuesChange={formProps.onValuesChange}
                    isSubmitting={formProps.isSubmitting}
                    formId={formProps.formId}
                    hideSubmit={formProps.hideSubmit}
                />
            )}
        </EntityFormDialog>
    )
}
