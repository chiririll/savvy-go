import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { RecurringFormData } from '@/schemas'
import { RecurringTransaction } from '@/types'
import { RecurringForm } from './RecurringForm'

const FORM_ID = 'recurring-form'

interface RecurringFormDialogProps {
    recurring?: RecurringTransaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: RecurringFormData) => void
    isSubmitting?: boolean
}

export function RecurringFormDialog({
    recurring,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
}: RecurringFormDialogProps) {
    const { t } = useTranslation('pages')

    return (
        <EntityFormDialog<RecurringTransaction, RecurringFormData, RecurringFormData>
            entity={recurring}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={recurring ? t('recurring.editTitle') : t('recurring.createTitle')}
            description={t('recurring.description')}
            toFormValues={(item) => ({
                type: item.type,
                account_id: item.account.id,
                to_account_id: item.toAccount?.id,
                category_id: item.category?.id,
                amount: item.amount,
                to_amount: item.toAmount,
                description: item.description,
                frequency: item.frequency,
                interval: item.interval,
                day_of_week: item.dayOfWeek,
                day_of_month: item.dayOfMonth,
                start_date: item.startDate,
                end_date: item.endDate,
                is_active: item.isActive,
                tag_ids: item.tags.map((tag) => tag.id),
            })}
        >
            {({ formKey, formProps }) => (
                <RecurringForm
                    key={formKey}
                    defaultValues={formProps.defaultValues}
                    onSubmit={onSubmit}
                    onValuesChange={formProps.onValuesChange}
                    isSubmitting={formProps.isSubmitting}
                    formId={formProps.formId}
                    hideSubmit
                />
            )}
        </EntityFormDialog>
    )
}
