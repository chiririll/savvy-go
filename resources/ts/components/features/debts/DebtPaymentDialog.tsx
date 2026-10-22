import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { schemaResolver } from '@/lib/form-resolver'
import { Button } from '@/components/ui/button'
import { ResponsiveDialog } from '@/components/shared/dialogs/ResponsiveDialog'
import { Form } from '@/components/ui/form'
import { debtPaymentSchema, DebtPaymentFormData } from '@/schemas'
import { useNegativeBalanceConfirm } from '@/components/shared'
import { useAccounts } from '@/hooks'
import { warningsForAccountOutflow } from '@/lib/negative-balance'
import { Debt } from '@/types'
import { formatDateLocal } from '@/lib/utils'
import { DebtPaymentFields } from './DebtPaymentFields'
import { DebtPaymentSummary } from './DebtPaymentSummary'

interface DebtPaymentDialogProps {
    debt: Debt | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (debtId: number, data: DebtPaymentFormData) => void
    isSubmitting?: boolean
    mode: 'payment' | 'collection'
}

const emptyPayment = (): DebtPaymentFormData => ({
    account_id: 0,
    amount: 0,
    date: formatDateLocal(),
    description: '',
})

export function DebtPaymentDialog({ debt, open, onOpenChange, onSubmit, isSubmitting, mode }: DebtPaymentDialogProps) {
    const { t } = useTranslation(['forms', 'pages', 'common'])
    const { data: accounts } = useAccounts({ active: true, exclude_debts: true })
    const { confirmIfNeeded, dialog: negativeBalanceDialog } = useNegativeBalanceConfirm<DebtPaymentFormData>()

    const form = useForm<DebtPaymentFormData>({
        resolver: schemaResolver<DebtPaymentFormData>(debtPaymentSchema),
        defaultValues: emptyPayment(),
    })

    useEffect(() => {
        if (!open) {
            return
        }

        form.reset(emptyPayment())
    }, [open, debt?.id, mode, form])

    const handleSubmit = (data: DebtPaymentFormData) => {
        if (!debt) {
            return
        }

        const selectedAccount = accounts?.find((account) => account.id === Number(data.account_id))
        const warnings = mode === 'payment' ? warningsForAccountOutflow(selectedAccount, Number(data.amount) || 0) : []

        confirmIfNeeded(data, warnings, (payload) => onSubmit(debt.id, payload))
    }

    const title = mode === 'payment' ? t('debts.payment.makeTitle') : t('debts.payment.collectTitle')
    const description = mode === 'payment' ? t('debts.payment.makeDescription') : t('debts.payment.collectDescription')

    return (
        <>
            <ResponsiveDialog
                open={open}
                onOpenChange={onOpenChange}
                title={title}
                description={description}
                footer={
                    <>
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            {t('common:actions.cancel')}
                        </Button>
                        <Button type="submit" form="debt-payment-form" disabled={isSubmitting}>
                            {isSubmitting ? t('common:actions.processing') : title}
                        </Button>
                    </>
                }
            >
                <div className="space-y-4">
                    {debt && <DebtPaymentSummary debt={debt} mode={mode} />}

                    <Form {...form}>
                        <form id="debt-payment-form" onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
                            <DebtPaymentFields form={form} debt={debt} mode={mode} />
                        </form>
                    </Form>
                </div>
            </ResponsiveDialog>
            {negativeBalanceDialog}
        </>
    )
}
