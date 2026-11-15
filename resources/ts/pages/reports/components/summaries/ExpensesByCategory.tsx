import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn, formatCurrency } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/format/localized-name'
import { colorShades } from '@/lib/ui/color-shades'
import { TrendingUp, TrendingDown, AlertTriangle, ChevronRight, ChevronDown } from 'lucide-react'
import { useExpensesByCategory } from '@/hooks'
import type { CategoryExpense } from '@/api/reports'
import type { ReportFilters } from '../../types'

interface ExpensesByCategoryProps {
    filters: ReportFilters
}

/** Top-level categories, each parent opening into its subcategories. */
export function ExpensesByCategory({ filters }: ExpensesByCategoryProps) {
    const { t } = useTranslation('pages')
    const navigate = useNavigate()
    const { data, isLoading, error } = useExpensesByCategory(filters)
    const [expanded, setExpanded] = useState<Set<number>>(new Set())

    const maxValue = useMemo(() => {
        if (!data?.categories.length) return 0
        return Math.max(...data.categories.flatMap((c) => [c.current, c.previous]))
    }, [data])
    const anyGroup = data?.categories.some((c) => c.children?.length) ?? false

    // A parent opens with its subcategories' transactions too.
    const open = (category: CategoryExpense) => {
        const ids = [...new Set([category.id, ...(category.children ?? []).map((c) => c.id)])]
        navigate(`/transactions?category_ids=${ids.join(',')}`)
    }
    const toggle = (id: number) =>
        setExpanded((prev) => {
            const next = new Set(prev)
            if (!next.delete(id)) next.add(id)
            return next
        })

    if (error) {
        return (
            <Card>
                <CardContent className="py-8 text-center text-destructive">
                    {t('reports.errors.expensesByCategory')}
                </CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader className="pb-2">
                <CardTitle className="text-lg">{t('reports.expensesByCategory.title')}</CardTitle>
                <p className="text-sm text-muted-foreground">{t('reports.expensesByCategory.subtitle')}</p>
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
                        const children = category.children ?? []
                        const isOpen = expanded.has(category.id)
                        const shades = colorShades(category.color, children.length)
                        const parentName = localizeDefaultName(category.name)
                        return (
                            <div key={category.id}>
                                <div className="flex items-center gap-1">
                                    {children.length > 0 ? (
                                        <button
                                            type="button"
                                            aria-expanded={isOpen}
                                            aria-label={
                                                isOpen
                                                    ? t('reports.structure.collapse', { name: parentName })
                                                    : t('reports.structure.expand', { name: parentName })
                                            }
                                            className={cn(
                                                'flex size-6 shrink-0 items-center justify-center rounded-md',
                                                'text-muted-foreground hover:bg-muted/50 focus-visible:outline-none',
                                                'focus-visible:ring-2 focus-visible:ring-ring',
                                            )}
                                            onClick={() => toggle(category.id)}
                                        >
                                            {isOpen ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
                                        </button>
                                    ) : (
                                        anyGroup && <span className="size-6 shrink-0" />
                                    )}
                                    <CategoryRow
                                        category={category}
                                        name={parentName}
                                        color={category.color}
                                        maxValue={maxValue}
                                        compare={filters.compareWith !== 'none'}
                                        currency={data.currency}
                                        onOpen={() => open(category)}
                                    />
                                </div>
                                {isOpen && (
                                    <div className={cn('space-y-0.5', anyGroup && 'pl-7')}>
                                        {children.map((child, i) => (
                                            <CategoryRow
                                                key={child.id}
                                                small
                                                category={child}
                                                name={
                                                    child.id === category.id
                                                        ? t('reports.structure.ownLines', { name: parentName })
                                                        : localizeDefaultName(child.name)
                                                }
                                                color={shades[i]}
                                                maxValue={maxValue}
                                                compare={filters.compareWith !== 'none'}
                                                currency={data.currency}
                                                onOpen={() => open(child)}
                                            />
                                        ))}
                                    </div>
                                )}
                            </div>
                        )
                    })
                )}
            </CardContent>
        </Card>
    )
}

interface CategoryRowProps {
    category: CategoryExpense
    name: string
    color: string
    maxValue: number
    compare: boolean
    currency: string | null
    onOpen: () => void
    /** A subcategory: a smaller icon and bar under its parent. */
    small?: boolean
}

function CategoryRow({ category, name, color, maxValue, compare, currency, onOpen, small = false }: CategoryRowProps) {
    const { t } = useTranslation('pages')
    const change = category.previous > 0 ? ((category.current - category.previous) / category.previous) * 100 : 0
    const isIncrease = change > 0
    const isAnomalous = Math.abs(change) > 50
    const currentWidth = maxValue > 0 ? (category.current / maxValue) * 100 : 0
    const previousWidth = maxValue > 0 ? (category.previous / maxValue) * 100 : 0
    const showPrevious = compare && category.previous > 0

    return (
        <button
            type="button"
            className={cn(
                'group block w-full min-w-0 cursor-pointer rounded-lg px-1 text-left transition-colors',
                small ? 'py-1.5' : 'py-2',
                'hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2',
                'focus-visible:ring-ring',
            )}
            onClick={onOpen}
        >
            {/* Header row: icon, name, amounts */}
            <div className="flex items-center gap-2.5 mb-1.5">
                <div
                    className={cn(
                        'flex items-center justify-center rounded-lg flex-shrink-0',
                        small ? 'size-6' : 'size-8',
                    )}
                    style={{ backgroundColor: color }}
                >
                    <span className={small ? 'text-sm' : 'text-base'}>{category.icon}</span>
                </div>

                <div className="flex min-w-0 flex-1 items-center gap-2">
                    <span className={cn('truncate', small ? 'text-sm' : 'font-medium text-sm')}>{name}</span>
                    {isAnomalous && compare && <AlertTriangle className="size-4 shrink-0 text-warning" />}
                </div>

                <div className={cn('flex shrink-0 flex-col items-end text-sm sm:flex-row', 'sm:items-center sm:gap-3')}>
                    <span className={small ? 'font-medium' : 'font-semibold'}>{formatCurrency(category.current, currency)}</span>
                    {showPrevious && (
                        <span className="flex items-center gap-2 text-xs sm:text-sm">
                            <span className="text-muted-foreground">
                                {t('reports.vsAmount', { amount: formatCurrency(category.previous, currency) })}
                            </span>
                            <span
                                className={cn(
                                    'flex items-center gap-0.5 text-xs font-medium',
                                    isIncrease ? 'text-destructive' : 'text-success',
                                )}
                            >
                                {isIncrease ? <TrendingUp className="size-3" /> : <TrendingDown className="size-3" />}
                                {Math.abs(change).toFixed(0)}%
                            </span>
                        </span>
                    )}
                </div>
                <ChevronRight
                    className={cn(
                        'hidden size-4 shrink-0 text-muted-foreground opacity-0 transition-opacity',
                        'group-hover:opacity-100 sm:block',
                    )}
                />
            </div>

            {/* Bars: full row width, one scale for parents and children */}
            <div className="space-y-1">
                <div className={cn('bg-muted rounded-full overflow-hidden', small ? 'h-1.5' : 'h-2.5')}>
                    <div
                        className="h-full rounded-full transition-all duration-500"
                        style={{ width: `${currentWidth}%`, backgroundColor: color }}
                    />
                </div>
                {showPrevious && (
                    <div className={cn('bg-muted rounded-full overflow-hidden', small ? 'h-1' : 'h-1.5')}>
                        <div
                            className="h-full rounded-full transition-all duration-500 opacity-40"
                            style={{ width: `${previousWidth}%`, backgroundColor: color }}
                        />
                    </div>
                )}
            </div>
        </button>
    )
}
