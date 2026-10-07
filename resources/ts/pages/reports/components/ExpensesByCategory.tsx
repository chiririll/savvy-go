import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn, formatCurrency } from '@/lib/utils'
import { TrendingUp, TrendingDown, AlertTriangle, ChevronRight } from 'lucide-react'
import { useExpensesByCategory } from '@/hooks'
import type { ReportFilters } from '../types'

interface ExpensesByCategoryProps {
    filters: ReportFilters
}

export function ExpensesByCategory({ filters }: ExpensesByCategoryProps) {
    const { t } = useTranslation('pages')
    const navigate = useNavigate()
    const { data, isLoading, error } = useExpensesByCategory(filters)

    const maxValue = useMemo(() => {
        if (!data?.categories.length) return 0
        return Math.max(...data.categories.flatMap(c => [c.current, c.previous]))
    }, [data])

    const handleCategoryClick = (categoryId: number) => {
        navigate(`/transactions?category_ids=${categoryId}`)
    }

    if (error) {
        return (
            <Card>
                <CardContent className="py-8 text-center text-red-500">
                    {t('reports.errors.expensesByCategory')}
                </CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader className="pb-2">
                <CardTitle className="text-lg">{t('reports.expensesByCategory.title')}</CardTitle>
                <p className="text-sm text-muted-foreground">
                    {t('reports.expensesByCategory.subtitle')}
                </p>
            </CardHeader>
            <CardContent className="space-y-1 px-3 sm:px-6">
                {isLoading ? (
                    <div className="space-y-3">
                        {Array.from({ length: 5 }).map((_, i) => (
                            <Skeleton key={i} className="h-16" />
                        ))}
                    </div>
                ) : !data?.categories.length ? (
                    <div className="py-8 text-center text-muted-foreground">
                        {t('reports.expensesByCategory.noData')}
                    </div>
                ) : (
                    data.categories.map((category) => {
                        const change = category.previous > 0
                            ? ((category.current - category.previous) / category.previous) * 100
                            : 0
                        const isIncrease = change > 0
                        const isAnomalous = Math.abs(change) > 50
                        const currentWidth = maxValue > 0 ? (category.current / maxValue) * 100 : 0
                        const previousWidth = maxValue > 0 ? (category.previous / maxValue) * 100 : 0

                        return (
                            <div
                                key={category.id}
                                className="group cursor-pointer rounded-lg px-1 py-2 transition-colors hover:bg-muted/50"
                                onClick={() => handleCategoryClick(category.id)}
                            >
                                {/* Header row: icon, name, amounts */}
                                <div className="flex items-center gap-2.5 mb-1.5">
                                    <div
                                        className="flex items-center justify-center size-8 rounded-lg flex-shrink-0"
                                        style={{ backgroundColor: category.color }}
                                    >
                                        <span className="text-base">{category.icon}</span>
                                    </div>

                                    <div className="flex min-w-0 flex-1 items-center gap-2">
                                        <span className="font-medium text-sm truncate">{category.name}</span>
                                        {isAnomalous && filters.compareWith !== 'none' && (
                                            <AlertTriangle className="size-4 shrink-0 text-amber-500" />
                                        )}
                                    </div>

                                    <div className="flex shrink-0 flex-col items-end text-sm sm:flex-row sm:items-center sm:gap-3">
                                        <span className="font-semibold">
                                            {formatCurrency(category.current, data.currency)}
                                        </span>
                                        {filters.compareWith !== 'none' && category.previous > 0 && (
                                            <span className="flex items-center gap-2 text-xs sm:text-sm">
                                                <span className="text-muted-foreground">
                                                    {t('reports.vsAmount', { amount: formatCurrency(category.previous, data.currency) })}
                                                </span>
                                                <span className={cn(
                                                    'flex items-center gap-0.5 text-xs font-medium',
                                                    isIncrease ? 'text-red-600' : 'text-green-600'
                                                )}>
                                                    {isIncrease ? (
                                                        <TrendingUp className="size-3" />
                                                    ) : (
                                                        <TrendingDown className="size-3" />
                                                    )}
                                                    {Math.abs(change).toFixed(0)}%
                                                </span>
                                            </span>
                                        )}
                                    </div>
                                    <ChevronRight className="hidden size-4 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 sm:block" />
                                </div>

                                {/* Bars: full row width */}
                                <div className="space-y-1">
                                    <div className="h-2.5 bg-muted rounded-full overflow-hidden">
                                        <div
                                            className="h-full rounded-full transition-all duration-500"
                                            style={{
                                                width: `${currentWidth}%`,
                                                backgroundColor: category.color,
                                            }}
                                        />
                                    </div>
                                    {filters.compareWith !== 'none' && category.previous > 0 && (
                                        <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                                            <div
                                                className="h-full rounded-full transition-all duration-500 opacity-40"
                                                style={{
                                                    width: `${previousWidth}%`,
                                                    backgroundColor: category.color,
                                                }}
                                            />
                                        </div>
                                    )}
                                </div>
                            </div>
                        )
                    })
                )}
            </CardContent>
        </Card>
    )
}
