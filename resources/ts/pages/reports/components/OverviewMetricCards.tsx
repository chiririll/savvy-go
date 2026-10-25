import { MetricCard } from './MetricCard'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import type { OverviewMetrics } from '@/api/reports'
import type { CompareType } from '../types'

export interface OverviewMetricSpec {
    key: 'income' | 'expenses' | 'netCashFlow' | 'savingsRate'
    title: string
    type: 'income' | 'expense' | 'net' | 'percent'
}

interface OverviewMetricCardsProps {
    metrics: OverviewMetricSpec[]
    data?: OverviewMetrics
    isLoading: boolean
    compareWith: CompareType
    className: string
}

/** A row of cards from the overview report; a skeleton per card while it loads. */
export function OverviewMetricCards({ metrics, data, isLoading, compareWith, className }: OverviewMetricCardsProps) {
    return (
        <div className={cn('grid grid-cols-1 gap-4', className)}>
            {isLoading
                ? metrics.map((m) => <Skeleton key={m.key} className="h-[140px]" />)
                : data &&
                  metrics.map(({ key, title, type }) => (
                      <MetricCard
                          key={key}
                          title={title}
                          value={data[key].value}
                          previousValue={data[key].previous}
                          sparklineData={data[key].sparkline}
                          type={type}
                          compareWith={compareWith}
                          currency={type === 'percent' ? undefined : data.currency}
                      />
                  ))}
        </div>
    )
}
