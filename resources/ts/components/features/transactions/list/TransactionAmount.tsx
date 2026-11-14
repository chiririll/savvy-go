import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatCurrency } from '@/lib/format/currency'
import type { RowAmount } from '@/types'

/** Shown when a row has no amount to give: a pending template waiting for its figure. */
export const NO_AMOUNT = '—'

/**
 * One amount of a row. Several currencies show as one value in the first of them, marked with an
 * asterisk; the exact totals are in the tooltip (and in the detail dialog).
 */
export function TransactionAmount({
    amount,
    sign,
    approximate = false,
}: {
    amount: RowAmount | null
    sign: '+' | '-' | ''
    /** The amount is a guess: a rule's estimate, or what the rates of the day will make of it. */
    approximate?: boolean
}) {
    if (!amount) {
        return <>{NO_AMOUNT}</>
    }
    const text = `${approximate ? '≈ ' : ''}${sign}${formatCurrency(amount.main.amount, amount.main.currency)}`
    if (!amount.converted) {
        return <>{text}</>
    }

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <span tabIndex={0} className="cursor-help" data-testid="converted-amount">
                    {text}*
                </span>
            </TooltipTrigger>
            <TooltipContent side="left">
                <ul className="font-mono">
                    {amount.totals.map(({ currency, amount: value }) => (
                        <li key={currency.code}>
                            {sign}
                            {formatCurrency(value, currency)}
                        </li>
                    ))}
                </ul>
            </TooltipContent>
        </Tooltip>
    )
}
