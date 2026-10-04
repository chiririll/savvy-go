import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { BudgetFormData } from '@/schemas'
import { Budget } from '@/types'
import { BudgetForm } from './BudgetForm'

const FORM_ID = 'budget-form'

interface BudgetFormDialogProps {
    budget?: Budget | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: BudgetFormData) => void
    isSubmitting?: boolean
}

export function BudgetFormDialog({
    budget,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
}: BudgetFormDialogProps) {
    const { t } = useTranslation('pages')

    return (
        <EntityFormDialog<Budget, BudgetFormData, BudgetFormData>
            entity={budget}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={budget ? t('budgets.editTitle') : t('budgets.createTitle')}
            description={t('budgets.description')}
            toFormValues={(item) => ({
                name: item.name,
                amount: item.amount,
                currency_id: item.currency?.id,
                period: item.period,
                start_date: item.startDate,
                end_date: item.endDate,
                is_global: item.isGlobal,
                notify_at_percent: item.notifyAtPercent,
                is_active: item.isActive,
                category_ids: item.categories.map((category) => category.id),
                tag_ids: item.tags?.map((tag) => tag.id) ?? [],
            })}
        >
            {({ formKey, formProps }) => (
                <BudgetForm
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
