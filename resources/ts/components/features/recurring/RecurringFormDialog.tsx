import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { EntityFormDialog, FormDialogCancel } from '@/components/shared'
import { useAccounts } from '@/hooks'
import { canEditRecurringInForm, recurringToFormValues } from '@/lib/transaction/recurring-template'
import type { RecurringFormValues } from '@/schemas/recurring'
import { RecurringInput, RecurringTransaction } from '@/types'
import { RecurringForm } from './RecurringForm'

const FORM_ID = 'recurring-form'

interface RecurringFormDialogProps {
    recurring?: RecurringTransaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: RecurringInput) => void
    isSubmitting?: boolean
}

export function RecurringFormDialog({
    recurring,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
}: RecurringFormDialogProps) {
    const { t } = useTranslation(['pages', 'forms'])
    // Every account, the inactive ones too: a rule may be about one the form no longer offers for new ones.
    const { data: accounts } = useAccounts()
    const toFormValues = useCallback(
        (rule: RecurringTransaction) => recurringToFormValues(rule, accounts ?? []) ?? {},
        [accounts],
    )
    // The rule is read with its accounts; until they arrive there is nothing to show.
    const ready = !recurring || accounts !== undefined
    const editable = !recurring || (accounts !== undefined && canEditRecurringInForm(recurring, accounts))

    return (
        <EntityFormDialog<RecurringTransaction, RecurringInput, RecurringFormValues>
            entity={recurring}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={recurring ? t('pages:recurring.editTitle') : t('pages:recurring.createTitle')}
            description={t('pages:recurring.description')}
            className="sm:max-w-xl"
            toFormValues={toFormValues}
            footerEnd={ready && !editable ? <FormDialogCancel /> : undefined}
        >
            {({ formKey, formProps }) =>
                !ready ? null : !editable ? (
                    <p className="text-sm text-muted-foreground">{t('forms:recurring.cannotEdit')}</p>
                ) : (
                    <RecurringForm
                        key={`${formKey}-${open}`}
                        defaultValues={formProps.defaultValues}
                        onSubmit={onSubmit}
                        onValuesChange={formProps.onValuesChange}
                        isSubmitting={formProps.isSubmitting}
                        formId={formProps.formId}
                        error={recurring?.error}
                        open={open}
                        hideSubmit
                    />
                )
            }
        </EntityFormDialog>
    )
}
