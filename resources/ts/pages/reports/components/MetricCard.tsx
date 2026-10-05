import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '@/components/ui/card'
import { Sparkline } from '@/components/ui/sparkline'
import { cn, formatCurrency } from '@/lib/utils'
import { TrendingUp, TrendingDown } from 'lucide-react'
import type { CompareType } from '../types'

interface MetricCardProps {
    title: string
    value: number
    previousValue?: number | null
    sparklineData: number[]
    type: 'income' | 'expense' | 'net' | 'percent'
    compareWith: CompareType
    suffix?: string
    currency?: string | null
}

export function MetricCard({ title, value, previousValue, sparklineData, type, compareWith, suffix, currency }: MetricCardProps) {
    const { t } = useTranslation('pages')
    const absoluteChange = previousValue != null ? value - previousValue : null
    const percentChange = previousValue && previousValue !== 0
        ? ((value - previousValue) / previousValue) * 100
        : null

    const isPositiveChange = percentChange !== null && percentChange > 0

    // For expenses, negative change (decrease) is good
    // For income/net/percent, positive change (increase) is good
    const isGoodChange = type === 'expense' ? !isPositiveChange : isPositiveChange

    const sparklineColor = type === 'income' ? 'success' : type === 'expense' ? 'danger' : 'default'

    const formatValue = (val: number) => {
        if (type === 'percent') {
            return `${val.toFixed(1)}%`
        }
        return formatCurrency(val, currency)
    }

    const formatChangeValue = (val: number) => {
        if (type === 'percent') {
            return t('reports.metrics.percentagePoints', { value: `${val > 0 ? '+' : ''}${val.toFixed(1)}` })
        }
        return `${val > 0 ? '+' : ''}${formatCurrency(val, currency)}`
    }

    return (
        <Card>
            <CardContent className="pt-5 pb-4">
                <div className="flex items-start justify-between gap-3">
                    <div className="space-y-1 min-w-0 flex-1">
                        <p className="text-sm text-muted-foreground truncate">{title}</p>
                        <p className={cn(
                            'text-xl sm:text-2xl font-bold tracking-tight break-words',
                            type === 'income' && 'text-green-600',
                            type === 'expense' && 'text-red-600',
                        )}>
                            {formatValue(value)}{suffix}
                        </p>
                        {compareWith !== 'none' && percentChange !== null && (
                            <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs">
                                <span className={cn(
                                    'flex items-center gap-0.5',
                                    isGoodChange ? 'text-green-600' : 'text-red-600'
                                )}>
                                    {isPositiveChange ? (
                                        <TrendingUp className="size-3" />
                                    ) : (
                                        <TrendingDown className="size-3" />
                                    )}
                                    {Math.abs(percentChange).toFixed(1)}%
                                </span>
                                {absoluteChange !== null && (
                                    <span className="text-muted-foreground">
                                        ({formatChangeValue(absoluteChange)})
                                    </span>
                                )}
                            </div>
                        )}
                    </div>
                    <Sparkline
                        data={sparklineData}
                        width={64}
                        height={28}
                        color={sparklineColor}
                        className="flex-shrink-0"
                    />
                </div>
            </CardContent>
        </Card>
    )
}
