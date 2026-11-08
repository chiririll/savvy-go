import { CircleHelp, TriangleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatCurrencyCompact } from '@/lib/currency'
import { cn } from '@/lib/utils'
import { reconciledLabel, type ReconciledTone } from './reconciliation-days'
import type { ReconciliationDay } from '@/types'

const TONE: Record<ReconciledTone, string> = {
    calm: 'text-muted-foreground',
    positive: 'text-success',
    negative: 'text-destructive',
}

interface ReconciledLineProps {
    day: ReconciliationDay
    baseCurrency?: string
    onOpenDetails: () => void
}

/** The end of a day that has reconciliation points: a rule with "Reconciled +amount" in the middle. */
export function ReconciledLine({ day, baseCurrency, onOpenDetails }: ReconciledLineProps) {
    const { t } = useTranslation('pages')
    const { amount, tone } = reconciledLabel(day, (value) => formatCurrencyCompact(value, baseCurrency))

    return (
        <div className="flex items-center gap-2 px-1.5 py-2 text-xs" data-testid="reconciled-line">
            <span className="h-px flex-1 bg-border" />
            <span className={cn('flex items-center gap-1 font-medium tabular-nums', TONE[tone])}>
                {t('transactions.reconciled.label')}
                {amount && <span>{amount}</span>}
                {day.unconverted && (
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <TriangleAlert
                                className="size-3.5 text-warning"
                                aria-label={t('transactions.reconciled.partial')}
                            />
                        </TooltipTrigger>
                        <TooltipContent>{t('transactions.reconciled.partial')}</TooltipContent>
                    </Tooltip>
                )}
            </span>
            <span className="h-px flex-1 bg-border" />
            <Button
                variant="ghost"
                size="icon"
                className="size-6 text-muted-foreground"
                aria-label={t('transactions.reconciled.details')}
                onClick={onOpenDetails}
            >
                <CircleHelp className="size-4" />
            </Button>
        </div>
    )
}
