import { useFieldArray, useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { formatCurrency } from '@/lib/utils'
import { nextFreeAccount, paymentTotals } from '@/lib/transaction-payments'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import type { BalancePreview } from './balance-preview'
import { BalanceAfter } from './TransactionBalanceHint'
import { PaymentRow } from './PaymentRow'
import { PaymentRowButtons } from './PaymentRowButtons'

const EMPTY_AMOUNT = '' as unknown as number

interface PaymentsSectionProps {
    accounts?: Account[]
    /** The accounts all the rows use: an account is in a transaction once. */
    usedAccountIds: number[]
    balances: BalancePreview[]
    /** Whether saving moves balances now; a pending transaction shows a hint instead of the balances. */
    showBalances: boolean
}

/** The accounts that paid (or received, for an income), one row each. */
export function PaymentsSection({ accounts, usedAccountIds, balances, showBalances }: PaymentsSectionProps) {
    const { t } = useTranslation('forms')
    const { control } = useFormContext<TransactionFormValues>()
    const { fields, append, remove } = useFieldArray({ control, name: 'payments' })
    const rows = useWatch({ control, name: 'payments' })
    const free = nextFreeAccount(accounts, usedAccountIds)

    return (
        <div className="space-y-3">
            {fields.map((field, index) => {
                const account = accounts?.find((a) => a.id === Number(rows?.[index]?.account_id))
                const balance = balances.find((row) => row.account.id === account?.id)
                const last = index === fields.length - 1
                return (
                    <PaymentRow
                        key={field.id}
                        control={control}
                        list="payments"
                        index={index}
                        currency={account?.currency}
                        excludeIds={usedAccountIds.filter((id) => id !== account?.id)}
                        action={
                            <PaymentRowButtons
                                onRemove={fields.length > 1 ? () => remove(index) : undefined}
                                removeLabel={t('transactions.removePayment')}
                                onAdd={last ? () => append({ account_id: free!.id, amount: EMPTY_AMOUNT }) : undefined}
                                addLabel={t('transactions.addPayment')}
                                addDisabled={!free}
                                addDisabledReason={t('transactions.noAccountLeft')}
                            />
                        }
                    >
                        {showBalances && balance && <BalanceAfter preview={balance} />}
                    </PaymentRow>
                )
            })}
        </div>
    )
}

/** One total per currency, once there is more than one payment or any change. */
export function PaymentTotals({ accounts }: { accounts?: Account[] }) {
    const { t } = useTranslation('forms')
    const { control } = useFormContext<TransactionFormValues>()
    const [payments, change] = useWatch({ control, name: ['payments', 'change'] })
    const totals = paymentTotals({ payments: payments ?? [], change: change ?? [] }, accounts ?? [])

    if ((payments?.length ?? 0) < 2 && !change?.length) {
        return null
    }
    if (totals.length === 0) {
        return null
    }

    return (
        <p className="flex flex-wrap items-baseline gap-x-1.5 text-xs text-muted-foreground" data-testid="payment-totals">
            <span>{t('transactions.totalPaid')}</span>
            {totals.map((total, i) => (
                <span key={total.currency.id} className="font-mono text-foreground">
                    {i > 0 && <span className="mr-1.5 text-muted-foreground">·</span>}
                    {formatCurrency(total.net, total.currency)}
                </span>
            ))}
        </p>
    )
}
