import i18n from '@/lib/i18n'
import { cn, formatCurrency } from '@/lib/utils'

type Currency = Parameters<typeof formatCurrency>[1]

/**
 * FeedRow amounts of something filling up to a total: what remains (or how
 * much it went over), and below it "of $300". Without a total only the
 * remaining amount is shown.
 */
export function progressAmounts({ remaining, total, currency, done }: {
    remaining: number
    total: number
    currency: Currency
    /** Replaces the remaining amount, e.g. "Paid off". */
    done?: string
}) {
    const hasTotal = total > 0
    const over = hasTotal && remaining < 0

    return {
        amount: done ?? (
            <>
                {over && (
                    <span className="mr-1 font-sans text-xs font-normal text-muted-foreground">
                        {i18n.t('progress.over')}
                    </span>
                )}
                {formatCurrency(hasTotal ? Math.abs(remaining) : remaining, currency)}
            </>
        ),
        amountClassName: cn(done ? 'font-sans text-green-600' : over && 'text-red-600'),
        extraAmount: hasTotal && i18n.t('progress.of', { amount: formatCurrency(total, currency) }),
    }
}
