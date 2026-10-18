import { useTranslation } from 'react-i18next'
import { Debt } from '@/types'
import { formatCurrency } from '@/lib/utils'

interface DebtPaymentSummaryProps {
    debt: Debt
    mode: 'payment' | 'collection'
}

export function DebtPaymentSummary({ debt, mode }: DebtPaymentSummaryProps) {
    const { t } = useTranslation(['forms', 'pages', 'common'])

    return (
        <div className="rounded-lg bg-muted p-4 space-y-2">
            <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">{t('pages:debts.columns.debt')}</span>
                <span className="font-medium">{debt.name}</span>
            </div>
            <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">{t('pages:debts.columns.remaining')}</span>
                <span className="font-mono">{formatCurrency(debt.currentBalance, debt.currency)}</span>
            </div>
            {debt.counterparty && (
                <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">
                        {mode === 'payment' ? t('debts.payment.payTo') : t('debts.payment.receiveFrom')}
                    </span>
                    <span>{debt.counterparty}</span>
                </div>
            )}
        </div>
    )
}
