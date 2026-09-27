import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { LucideIcon } from 'lucide-react'
import { Filter, ArrowUpDown, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import { TRANSACTION_TYPE_OPTIONS } from '@/constants/transactions'
import type { Category, Tag } from '@/types'
import type { TransactionListFilters } from '@/hooks/use-transaction-list-filters'

const TYPE_FILTERS: {
    value: 'income' | 'expense' | 'transfer' | null
    labelKey: string
    icon?: LucideIcon
}[] = [
    { value: null, labelKey: 'all' },
    ...TRANSACTION_TYPE_OPTIONS.map((option) => ({
        value: option.value,
        labelKey: option.value,
        icon: option.icon,
    })),
]

const SORT_OPTIONS = [
    { value: 'date:desc', labelKey: 'dateNewest' },
    { value: 'date:asc', labelKey: 'dateOldest' },
    { value: 'amount:desc', labelKey: 'amountHigh' },
    { value: 'amount:asc', labelKey: 'amountLow' },
] as const

interface TransactionFiltersPanelProps {
    list: TransactionListFilters
    categories?: Category[]
    tags?: Tag[]
}

export function TransactionFiltersPanel({ list, categories, tags }: TransactionFiltersPanelProps) {
    const { t } = useTranslation('pages')
    const [filtersOpen, setFiltersOpen] = useState(false)
    const { params, activeFiltersCount, setType, setStatus, setSort, setDateRange, toggleCategory, toggleTag, clearFilters } = list

    const filteredCategories = categories?.filter((category) =>
        !params.type || params.type === 'transfer' || category.type === params.type
    ) ?? []

    return (
        <>
            <Tabs
                value={params.status ?? 'all'}
                onValueChange={(value) => setStatus(value === 'pending' ? 'pending' : null)}
                className="mb-4"
            >
                <TabsList>
                    <TabsTrigger value="all">{t('transactions.tabs.confirmed')}</TabsTrigger>
                    <TabsTrigger value="pending">{t('transactions.tabs.pending')}</TabsTrigger>
                </TabsList>
            </Tabs>

            <div className="mb-4 flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="grid min-w-0 grid-cols-4 gap-1.5 sm:flex sm:gap-2">
                    {TYPE_FILTERS.map(({ value, labelKey, icon: Icon }) => {
                        const label = labelKey === 'all'
                            ? t('common:actions.all')
                            : t(`transactions.types.${labelKey}`)

                        return (
                            <Button
                                key={labelKey}
                                variant={params.type === value ? 'default' : 'outline'}
                                size="sm"
                                className="min-w-0 px-2 sm:shrink-0 sm:px-2.5"
                                onClick={() => setType(value)}
                                aria-pressed={params.type === value}
                            >
                                {Icon && <Icon className="hidden size-4 sm:block" />}
                                <span className="truncate">{label}</span>
                            </Button>
                        )
                    })}
                </div>
                <Select
                    value={`${params.sortBy}:${params.sortDir}`}
                    onValueChange={(val) => {
                        const [sortBy, sortDir] = val.split(':') as ['date' | 'amount', 'asc' | 'desc']
                        setSort(sortBy, sortDir)
                    }}
                >
                    <SelectTrigger className="h-9 w-full min-w-0 sm:w-[180px]">
                        <ArrowUpDown className="size-4" />
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {SORT_OPTIONS.map((opt) => (
                            <SelectItem key={opt.value} value={opt.value}>
                                {t(`transactions.sort.${opt.labelKey}`)}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>

            <Collapsible open={filtersOpen} onOpenChange={setFiltersOpen} className="mb-4">
                <div className="flex items-center gap-2">
                    <CollapsibleTrigger asChild>
                        <Button variant="outline" size="sm">
                            <Filter className="size-4 mr-2" />
                            {t('transactions.filters')}
                            {activeFiltersCount > 0 && (
                                <Badge variant="secondary" className="ml-2 px-1.5 py-0 text-xs">
                                    {activeFiltersCount}
                                </Badge>
                            )}
                        </Button>
                    </CollapsibleTrigger>
                    {activeFiltersCount > 0 && (
                        <Button variant="ghost" size="sm" onClick={clearFilters}>
                            <X className="size-4 mr-1" />
                            {t('transactions.clear')}
                        </Button>
                    )}
                </div>
                <CollapsibleContent className="mt-4 space-y-4">
                    <Card>
                        <CardContent className="pt-4 space-y-4">
                            <div>
                                <label className="text-sm font-medium mb-2 block">{t('transactions.dateRange')}</label>
                                <div className="flex items-center gap-2">
                                    <Input
                                        type="date"
                                        value={params.startDate ?? ''}
                                        onChange={(event) => setDateRange('startDate', event.target.value)}
                                        className="w-auto"
                                    />
                                    <span className="text-muted-foreground">{t('transactions.to')}</span>
                                    <Input
                                        type="date"
                                        value={params.endDate ?? ''}
                                        onChange={(event) => setDateRange('endDate', event.target.value)}
                                        className="w-auto"
                                    />
                                </div>
                            </div>

                            {filteredCategories.length > 0 && (
                                <div>
                                    <label className="text-sm font-medium mb-2 block">{t('transactions.categories')}</label>
                                    <div className="flex flex-wrap gap-2">
                                        {filteredCategories.map((category) => {
                                            const isSelected = params.categoryIds.includes(category.id)
                                            return (
                                                <Badge
                                                    key={category.id}
                                                    variant={isSelected ? 'default' : 'outline'}
                                                    className={cn(
                                                        'cursor-pointer transition-colors',
                                                        isSelected ? 'hover:bg-primary/80' : 'hover:bg-muted'
                                                    )}
                                                    onClick={() => toggleCategory(category.id)}
                                                >
                                                    {category.icon} {localizeDefaultName(category.name)}
                                                </Badge>
                                            )
                                        })}
                                    </div>
                                </div>
                            )}

                            {tags && tags.length > 0 && (
                                <div>
                                    <label className="text-sm font-medium mb-2 block">{t('transactions.tags')}</label>
                                    <div className="flex flex-wrap gap-2">
                                        {tags.map((tag) => {
                                            const isSelected = params.tagIds.includes(tag.id)
                                            return (
                                                <Badge
                                                    key={tag.id}
                                                    variant={isSelected ? 'default' : 'outline'}
                                                    className={cn(
                                                        'cursor-pointer transition-colors',
                                                        isSelected ? 'hover:bg-primary/80' : 'hover:bg-muted'
                                                    )}
                                                    onClick={() => toggleTag(tag.id)}
                                                >
                                                    #{tag.name}
                                                </Badge>
                                            )
                                        })}
                                    </div>
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </CollapsibleContent>
            </Collapsible>
        </>
    )
}
