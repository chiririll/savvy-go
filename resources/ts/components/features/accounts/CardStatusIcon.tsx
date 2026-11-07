import { AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { useRelativeDay } from '@/hooks'
import { formatCurrency, cn } from '@/lib/utils'
import type { Account } from '@/types'

/**
 * The triangle of a card that needs attention: yellow for a warning, red for danger, nothing when all is well.
 * It opens a popover with the reason, so it works by touch as well as by mouse.
 */
export function CardStatusIcon({ account }: { account: Account }) {
    const { t } = useTranslation('pages')
    const relativeDay = useRelativeDay()
    const status = account.creditCard?.status
    if (!status || status.level === 'ok' || !status.reason) {
        return null
    }
    const values = { amount: formatCurrency(status.amount, account.currency), date: status.date ? relativeDay(status.date) : '' }
    const reason = {
        overdue: t('accounts.card.status.overdue', values),
        over_limit: t('accounts.card.status.over_limit', values),
        payment_due: t('accounts.card.status.payment_due', values),
    }[status.reason]

    return (
        <Popover>
            <PopoverTrigger asChild>
                <button
                    type="button"
                    aria-label={reason}
                    onClick={(event) => event.stopPropagation()}
                    className={cn(
                        'inline-flex size-5 shrink-0 items-center justify-center rounded-sm outline-none',
                        'focus-visible:ring-[3px] focus-visible:ring-ring/50',
                        status.level === 'danger' ? 'text-destructive' : 'text-warning',
                    )}
                >
                    <AlertTriangle className="size-4" />
                </button>
            </PopoverTrigger>
            <PopoverContent className="w-auto max-w-64 text-sm" onClick={(event) => event.stopPropagation()}>
                {reason}
            </PopoverContent>
        </Popover>
    )
}
