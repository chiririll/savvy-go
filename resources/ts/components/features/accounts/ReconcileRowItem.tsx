import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { AmountInput } from '@/components/features/transactions/form/AmountInput'
import { formatCurrency } from '@/lib/format/currency'
import { testId } from '@/lib/test-id'
import { cn } from '@/lib/utils'
import type { ReconcileRow } from '@/lib/transaction/reconciliation-rows'
import type { Account } from '@/types'

interface ReconcileRowItemProps {
    row: ReconcileRow
    account: Account
    /** Typed minus computed as exact text; null while it cannot be told. */
    difference: string | null
    disabled?: boolean
    onChange: (amount: string) => void
    onRemove: () => void
}

/** `[Account name + currency] [amount] (X)`, with the difference to the ledger's balance under the amount. */
export function ReconcileRowItem({ row, account, difference, disabled, onChange, onRemove }: ReconcileRowItemProps) {
    const { t } = useTranslation('pages')

    return (
        <li className="flex items-start gap-2" {...testId('reconcile-row')}>
            <div className="min-w-0 flex-1 pt-2">
                <div className="truncate text-sm font-medium">{account.name}</div>
                <div className="text-xs text-muted-foreground">{account.currency?.code}</div>
            </div>
            <div className="w-40 shrink-0 sm:w-48">
                <AmountInput
                    value={row.amount}
                    currency={account.currency}
                    hideCode
                    disabled={disabled}
                    aria-label={t('accounts.reconcile.amount', { name: account.name })}
                    onChange={onChange}
                />
                {difference !== null && (
                    <div
                        className={cn(
                            'mt-1 text-right font-mono text-xs text-muted-foreground',
                            Number(difference) > 0 && 'text-success',
                            Number(difference) < 0 && 'text-destructive',
                        )}
                        {...testId('reconcile-difference')}
                    >
                        {t('accounts.reconcile.difference', {
                            amount: `${Number(difference) > 0 ? '+' : ''}${formatCurrency(Number(difference), account.currency)}`,
                        })}
                    </div>
                )}
            </div>
            <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                className="mt-1 shrink-0 text-muted-foreground"
                aria-label={t('accounts.reconcile.remove', { name: account.name })}
                onClick={onRemove}
            >
                <X className="size-4" />
            </Button>
        </li>
    )
}
