import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { AccountFormValues, AccountFormData, encodeAccountCurrency } from '@/schemas'
import { Account } from '@/types'
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

    return (
        <EntityFormDialog<Account, AccountFormData, AccountFormValues>
            entity={account}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={account ? t('accounts.editTitle') : t('accounts.createTitle')}
            description={t('accounts.description')}
            toFormValues={(item) => ({
                name: item.name,
                type: item.type === 'debt' ? 'bank' : item.type,
                currency: encodeAccountCurrency({ id: item.currencyId }),
                initial_balance: item.initialBalance,
                is_active: item.isActive,
            })}
        >
            {({ formKey, formProps, isEdit }) => (
                <AccountForm
                    key={formKey}
                    isEdit={isEdit}
                    defaultValues={formProps.defaultValues}
                    onSubmit={onSubmit}
                    onValuesChange={formProps.onValuesChange}
                    isSubmitting={formProps.isSubmitting}
                    formId={formProps.formId}
                    hideSubmit={formProps.hideSubmit}
                />
            )}
        </EntityFormDialog>
    )
}
