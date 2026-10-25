import { cn, formatCurrency } from '@/lib/utils'
import { lastAmount, type LabeledSeries } from './balance-dynamics'

interface BalanceSeriesLegendProps {
    series: LabeledSeries[]
    hiddenKeys: ReadonlySet<string>
    onToggle: (key: string) => void
}

export function BalanceSeriesLegend({ series, hiddenKeys, onToggle }: BalanceSeriesLegendProps) {
    return (
        <div className="flex min-w-0 flex-wrap justify-center gap-x-3 gap-y-1.5">
            {series.map((item) => {
                const hidden = hiddenKeys.has(item.key)
                return (
                    <button
                        key={item.key}
                        type="button"
                        onClick={() => onToggle(item.key)}
                        aria-pressed={!hidden}
                        className={cn(
                            'inline-flex max-w-full min-w-0 items-center gap-1.5 overflow-hidden rounded-md',
                            'px-1.5 py-0.5 text-xs transition-opacity hover:bg-muted',
                            hidden && 'text-muted-foreground line-through',
                        )}
                    >
                        <span className={cn('size-2 shrink-0 rounded-sm', hidden && 'opacity-40')} style={{ backgroundColor: item.color }} />
                        <span className="min-w-0 truncate">{item.label}</span>
                        <span className="shrink-0 font-mono text-muted-foreground">
                            {formatCurrency(lastAmount(item), item.currency)}
                        </span>
                    </button>
                )
            })}
        </div>
    )
}
