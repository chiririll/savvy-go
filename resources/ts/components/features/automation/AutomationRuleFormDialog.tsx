import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import type { AutomationRule } from '@/types/automation'
import type { AutomationRuleFormData } from '@/schemas'
import { AutomationRuleForm } from './AutomationRuleForm'

const FORM_ID = 'automation-rule-form'

interface AutomationRuleFormDialogProps {
    rule?: AutomationRule | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: AutomationRuleFormData) => void
    isSubmitting?: boolean
}

export function AutomationRuleFormDialog({
    rule,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
}: AutomationRuleFormDialogProps) {
    const { t } = useTranslation('pages')

    return (
        <EntityFormDialog<AutomationRule, AutomationRuleFormData, AutomationRuleFormData>
            entity={rule}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={rule ? t('automation.editTitle') : t('automation.createTitle')}
            description={t('automation.createDescription')}
            className="sm:max-w-xl"
            toFormValues={(item) => ({
                name: item.name,
                description: item.description,
                trigger_type: item.triggerType,
                priority: item.priority,
                conditions: item.conditions,
                actions: item.actions,
                is_active: item.isActive,
                stop_processing: item.stopProcessing,
            })}
        >
            {({ formKey, formProps }) => (
                <AutomationRuleForm
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
