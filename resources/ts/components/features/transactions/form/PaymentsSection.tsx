import { useFieldArray, useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { formatCurrency } from '@/lib/utils'
import { nextFreeAccount, paymentTotals } from '@/lib/transaction-payments'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import type { BalancePreview } from './balance-preview'
import { PaymentGrid } from './PaymentGrid'
import { followAmount, useStartingAmount } from './use-payment-follows-line'
import type { BalanceMode } from './TransactionBalanceHint'

const EMPTY_AMOUNT = '' as unknown as number

interface PaymentsSectionProps {
    accounts?: Account[]
    /** The accounts all the rows use: an account is in a transaction once. */
    usedAccountIds: number[]
    balances: BalancePreview[]
    /** Whether saving moves balances now; a pending transaction shows a hint instead of the balances. */
    balanceMode: BalanceMode
}

/** The accounts that paid (or received, for an income), one row each. */
export function PaymentsSection({ accounts, usedAccountIds, balances, balanceMode }: PaymentsSectionProps) {
    const { t } = useTranslation('forms')
    const form = useFormContext<TransactionFormValues>()
    const { control } = form
    const { fields, append, remove } = useFieldArray({ control, name: 'payments' })
    const rows = useWatch({ control, name: 'payments' })
    useStartingAmount(accounts)
    const free = nextFreeAccount(accounts, usedAccountIds)

    return (
        <PaymentGrid
            control={control}
            list="payments"
            keys={fields.map((field) => field.id)}
            rows={rows}
            accounts={accounts}
            usedAccountIds={usedAccountIds}
            balances={balances}
            balanceMode={balanceMode}
            onAdd={() => append({ account_id: free!.id, amount: EMPTY_AMOUNT })}
            addLabel={t('transactions.addPayment')}
            addDisabled={!free}
            addDisabledReason={t('transactions.noAccountLeft')}
            onFirstAmountChange={(value) => followAmount(form, accounts, 'payment', value)}
            summary={<PaymentTotals accounts={accounts} />}
            onRemove={fields.length > 1 ? remove : undefined}
            removeLabel={t('transactions.removePayment')}
        />
    )
}

/** One total per currency, small, at the end of the Amount label: it takes no line of its own. */
export function PaymentTotals({ accounts }: { accounts?: Account[] }) {
    const { t } = useTranslation('forms')
    const { control } = useFormContext<TransactionFormValues>()
    const [payments, change] = useWatch({ control, name: ['payments', 'change'] })
    const totals = paymentTotals({ payments: payments ?? [], change: change ?? [] }, accounts ?? [])

    // Always there, so nothing appears when the second payment comes: zero in the currency of the first account.
    const first = accounts?.find((a) => a.id === Number(payments?.[0]?.account_id))?.currency
    const shown = totals.length > 0 ? totals.map((total) => ({ currency: total.currency, net: total.net })) : first ? [{ currency: first, net: 0 }] : []

    return (
        <p className="flex min-w-0 items-baseline justify-end gap-x-1.5 truncate text-xs font-normal text-muted-foreground" data-testid="payment-totals">
            <span>{t('transactions.totalPaid')}</span>
            {shown.map((total, i) => (
                <span key={total.currency.id} className="font-mono text-foreground">
                    {i > 0 && <span className="mr-1.5 text-muted-foreground">·</span>}
                    {formatCurrency(total.net, total.currency)}
                </span>
            ))}
        </p>
    )
}
