import { useTranslation } from 'react-i18next'
import { cn, formatCurrency } from '@/lib/utils'
import type { BalancePreview } from './balance-preview'

/** "Balance after": what an account holds now and what it will hold once the transaction is saved. */
export function BalanceAfter({ preview }: { preview: BalancePreview }) {
    const { t } = useTranslation('forms')

    return (
        <div className="flex min-h-5 flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs">
            <span className="text-muted-foreground">{t('transactions.balanceAfter')}</span>
            <span className="font-mono">{formatCurrency(preview.currentBalance, preview.currency)}</span>
            <span className="text-muted-foreground">→</span>
            <span
                className={cn(
                    'font-mono',
                    preview.insufficientFunds
                        ? 'text-destructive'
                        : preview.newBalance > preview.currentBalance && 'text-success',
                )}
            >
                {formatCurrency(preview.newBalance, preview.currency)}
            </span>
            {preview.insufficientFunds && (
                <span className="font-medium text-destructive">{t('transactions.willGoNegative')}</span>
            )}
        </div>
    )
}

/** Stands in for the balances while the date is empty or in the future: nothing moves until it is confirmed. */
export function PendingHint({ isUndated }: { isUndated: boolean }) {
    const { t } = useTranslation('forms')

    return (
        <p className="text-xs text-muted-foreground">
            {isUndated ? t('transactions.noDateHint') : t('transactions.pendingHint')}
        </p>
    )
}
