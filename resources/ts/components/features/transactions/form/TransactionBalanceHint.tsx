import { useTranslation } from 'react-i18next'
import { cn, formatCurrency } from '@/lib/utils'
import type { BalancePreview } from './balance-preview'

function BalancePair({ preview, label }: { preview: BalancePreview; label: string }) {
    return (
        <span className="inline-flex items-center gap-1.5">
            <span className="text-muted-foreground">{label}</span>
            <span className="font-mono">
                {formatCurrency(preview.currentBalance, preview.currency)}
            </span>
            <span className="text-muted-foreground">→</span>
            <span className={cn(
                'font-mono',
                preview.insufficientFunds
                    ? 'text-destructive'
                    : preview.newBalance > preview.currentBalance && 'text-green-600'
            )}>
                {formatCurrency(preview.newBalance, preview.currency)}
            </span>
        </span>
    )
}

export function TransactionBalanceHint({
    from,
    to,
    isPending,
    isUndated,
}: {
    from: BalancePreview | null
    to: BalancePreview | null
    isPending: boolean
    isUndated: boolean
}) {
    const { t } = useTranslation('forms')

    return (
        <div className="flex min-h-5 flex-wrap items-center gap-x-3 gap-y-1 text-xs">
            {isPending ? (
                <span className="text-muted-foreground">
                    {isUndated ? t('transactions.noDateHint') : t('transactions.pendingHint')}
                </span>
            ) : (
                <>
                    {from && <BalancePair preview={from} label={t('transactions.balance')} />}
                    {to && <BalancePair preview={to} label={t('transactions.toBalance')} />}
                    {from?.insufficientFunds && (
                        <span className="font-medium text-destructive">{t('transactions.willGoNegative')}</span>
                    )}
                </>
            )}
        </div>
    )
}
