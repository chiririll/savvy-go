import { useTranslation } from 'react-i18next'
import { TriangleAlert } from 'lucide-react'
import { cn, formatBalanceShort, formatCurrency } from '@/lib/utils'
import type { Account } from '@/types'
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

/** What a row shows under its account: nothing (a template), the balance now (a pending date), or now and after. */
export type BalanceMode = 'none' | 'current' | 'after'

/**
 * The balance of an account in a few characters. It is always there once an account is chosen, so the form does not
 * move when an amount is typed: the balance now, then `1.2K → 800` once the payment changes it. The currency is the
 * one of the amount above, so it is not repeated. A balance that goes negative is red with a triangle; screen
 * readers get the full numbers and the words.
 */
export function BalanceShort({
    account,
    preview,
    mode,
}: {
    account?: Account
    preview?: BalancePreview
    mode: BalanceMode
}) {
    const { t } = useTranslation('forms')
    const currency = account?.currency
    const after = mode === 'after' ? preview : undefined
    const current = account?.currentBalance

    return (
        <p className="flex min-h-5 min-w-0 items-center gap-1 font-mono text-xs text-muted-foreground" data-testid="balance-after">
            {account && current !== undefined && (
                <>
                    <span className="sr-only">
                        {t('transactions.balanceAfter')} {formatCurrency(current, currency)}
                        {after && ` → ${formatCurrency(after.newBalance, currency)}`}
                        {after?.insufficientFunds && `. ${t('transactions.willGoNegative')}`}
                    </span>
                    <span aria-hidden className="truncate">{formatBalanceShort(current, currency)}</span>
                    {after && (
                        <>
                            <span aria-hidden>→</span>
                            <span
                                aria-hidden
                                className={cn(
                                    'truncate',
                                    after.insufficientFunds ? 'text-destructive' : after.newBalance > current && 'text-success',
                                )}
                            >
                                {formatBalanceShort(after.newBalance, currency)}
                            </span>
                            {after.insufficientFunds && (
                                <TriangleAlert aria-hidden className="size-3.5 shrink-0 text-destructive" />
                            )}
                        </>
                    )}
                </>
            )}
        </p>
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
