import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from '@/components/ui/popover'
import {
    Sheet,
    SheetContent,
    SheetHeader,
    SheetTitle,
    SheetTrigger,
} from '@/components/ui/sheet'
import { Checkbox } from '@/components/ui/checkbox'
import { useCategories, useAccounts, useTags } from '@/hooks'
import { localizeDefaultName } from '@/lib/localized-name'
import { useIsMobile } from '@/hooks/use-mobile'
import { RotateCcw, ChevronDown, Calendar, Filter, SlidersHorizontal } from 'lucide-react'
import type { ReportFilters, PeriodType, CompareType } from '../types'
import { getMonthOptions, getQuarterOptions, getYearOptions } from '../utils'
import { periodTypeLabel } from '@/lib/labels'

interface EntityFilterItem {
    id: number
    name: string
}

function EntityFilter({
    label,
    items,
    selectedIds,
    onToggle,
    inSheet = false,
    formatName,
    popoverClassName,
    hideWhenEmpty = false,
}: {
    label: string
    items?: EntityFilterItem[]
    selectedIds: number[]
    onToggle: (id: number) => void
    inSheet?: boolean
    formatName?: (item: EntityFilterItem) => string
    popoverClassName?: string
    hideWhenEmpty?: boolean
}) {
    if (hideWhenEmpty && (!items || items.length === 0)) {
        return null
    }

    const list = (
        <div className={inSheet ? 'space-y-1 max-h-40 overflow-y-auto' : 'space-y-1 max-h-64 overflow-y-auto'}>
            {items?.map((item) => (
                <div
                    key={item.id}
                    className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-muted cursor-pointer"
                    onClick={() => onToggle(item.id)}
                >
                    <Checkbox
                        checked={selectedIds.includes(item.id)}
                        className="pointer-events-none"
                    />
                    <span className="text-sm">{formatName?.(item) ?? item.name}</span>
                </div>
            ))}
        </div>
    )

    if (inSheet) {
        return (
            <div className="space-y-2">
                <Label className="text-sm font-medium">{label}</Label>
                <div className="border rounded-md p-2">{list}</div>
            </div>
        )
    }

    return (
        <Popover>
            <PopoverTrigger asChild>
                <Button variant="outline" size="sm" className="h-8">
                    {label}
                    {selectedIds.length > 0 && (
                        <Badge variant="secondary" className="ml-1 px-1.5">
                            {selectedIds.length}
                        </Badge>
                    )}
                    <ChevronDown className="ml-1 size-3" />
                </Button>
            </PopoverTrigger>
            <PopoverContent className={popoverClassName ?? 'w-56 p-2'} align="start">
                {list}
            </PopoverContent>
        </Popover>
    )
}

interface FiltersBarProps {
    filters: ReportFilters
    onFilterChange: <K extends keyof ReportFilters>(key: K, value: ReportFilters[K]) => void
    onToggleArrayFilter: (key: 'accountIds' | 'categoryIds' | 'tagIds', id: number) => void
    onReset: () => void
}

export function FiltersBar({ filters, onFilterChange, onToggleArrayFilter, onReset }: FiltersBarProps) {
    const { t, i18n } = useTranslation('pages')
    const { data: categories } = useCategories()
    const { data: accounts } = useAccounts()
    const { data: tags } = useTags()
    const isMobile = useIsMobile()
    const [sheetOpen, setSheetOpen] = useState(false)

    const monthOptions = useMemo(() => getMonthOptions(), [i18n.language])
    const quarterOptions = useMemo(() => getQuarterOptions(), [i18n.language])
    const yearOptions = useMemo(() => getYearOptions(), [])

    const hasActiveFilters =
        filters.accountIds.length > 0 ||
        filters.categoryIds.length > 0 ||
        filters.tagIds.length > 0

    const activeFiltersCount =
        filters.accountIds.length +
        filters.categoryIds.length +
        filters.tagIds.length

    // Period type badges component
    const PeriodTypeBadges = () => (
        <div className="flex flex-wrap gap-1">
            {(['last_30_days', 'month', 'quarter', 'year', 'ytd'] as PeriodType[]).map(type => (
                <Badge
                    key={type}
                    variant={filters.periodType === type ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => onFilterChange('periodType', type)}
                >
                    {periodTypeLabel(t, type)}
                </Badge>
            ))}
            <Badge
                variant={filters.periodType === 'custom' ? 'default' : 'outline'}
                className="cursor-pointer"
                onClick={() => onFilterChange('periodType', 'custom')}
            >
                {t('reports.filters.custom')}
            </Badge>
        </div>
    )

    // Period selector based on type
    const PeriodSelector = () => (
        <>
            {filters.periodType === 'month' && (
                <Select
                    value={filters.selectedMonth}
                    onValueChange={(val) => onFilterChange('selectedMonth', val)}
                >
                    <SelectTrigger className="w-full md:w-[180px]">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {monthOptions.map(opt => (
                            <SelectItem key={opt.value} value={opt.value}>
                                {opt.label}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            )}

            {filters.periodType === 'quarter' && (
                <Select
                    value={filters.selectedQuarter}
                    onValueChange={(val) => onFilterChange('selectedQuarter', val)}
                >
                    <SelectTrigger className="w-full md:w-[140px]">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {quarterOptions.map(opt => (
                            <SelectItem key={opt.value} value={opt.value}>
                                {opt.label}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            )}

            {filters.periodType === 'year' && (
                <Select
                    value={filters.selectedYear}
                    onValueChange={(val) => onFilterChange('selectedYear', val)}
                >
                    <SelectTrigger className="w-full md:w-[100px]">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {yearOptions.map(opt => (
                            <SelectItem key={opt.value} value={opt.value}>
                                {opt.label}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            )}

            {filters.periodType === 'custom' && (
                <div className="flex flex-col md:flex-row items-stretch md:items-center gap-2">
                    <Input
                        type="date"
                        value={filters.customStartDate}
                        onChange={(e) => onFilterChange('customStartDate', e.target.value)}
                        className="w-full md:w-[140px]"
                    />
                    <span className="text-muted-foreground text-center hidden md:block">—</span>
                    <Input
                        type="date"
                        value={filters.customEndDate}
                        onChange={(e) => onFilterChange('customEndDate', e.target.value)}
                        className="w-full md:w-[140px]"
                    />
                </div>
            )}
        </>
    )

    // Comparison selector
    const ComparisonSelector = () => (
        <Select
            value={filters.compareWith}
            onValueChange={(val) => onFilterChange('compareWith', val as CompareType)}
        >
            <SelectTrigger className="w-full md:w-[200px]">
                <SelectValue placeholder={t('reports.filters.comparePlaceholder')} />
            </SelectTrigger>
            <SelectContent>
                <SelectItem value="none">{t('reports.filters.noComparison')}</SelectItem>
                <SelectItem value="previous_period">{t('reports.filters.previousPeriod')}</SelectItem>
                <SelectItem value="same_period_last_year">{t('reports.filters.samePeriodLastYear')}</SelectItem>
            </SelectContent>
        </Select>
    )

    const entityFilters = (inSheet = false) => (
        <>
            <EntityFilter
                label={t('reports.filters.accounts')}
                items={accounts}
                selectedIds={filters.accountIds}
                onToggle={(id) => onToggleArrayFilter('accountIds', id)}
                inSheet={inSheet}
            />
            <EntityFilter
                label={t('reports.filters.categories')}
                items={categories}
                selectedIds={filters.categoryIds}
                onToggle={(id) => onToggleArrayFilter('categoryIds', id)}
                inSheet={inSheet}
                formatName={(item) => localizeDefaultName(item.name)}
                popoverClassName="w-64 p-2"
            />
            <EntityFilter
                label={t('reports.filters.tags')}
                items={tags}
                selectedIds={filters.tagIds}
                onToggle={(id) => onToggleArrayFilter('tagIds', id)}
                inSheet={inSheet}
                formatName={(item) => `#${item.name}`}
                popoverClassName="w-48 p-2"
                hideWhenEmpty
            />
        </>
    )

    // Mobile version
    if (isMobile) {
        return (
            <Card className="mb-6">
                <CardContent className="py-3">
                    <div className="flex items-center justify-between gap-2">
                        {/* Quick period selection */}
                        <div className="flex-1 overflow-x-auto">
                            <PeriodTypeBadges />
                        </div>

                        {/* Filters button */}
                        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
                            <SheetTrigger asChild>
                                <Button variant="outline" size="sm" className="shrink-0">
                                    <SlidersHorizontal className="size-4" />
                                    {activeFiltersCount > 0 && (
                                        <Badge variant="secondary" className="ml-1 px-1.5">
                                            {activeFiltersCount}
                                        </Badge>
                                    )}
                                </Button>
                            </SheetTrigger>
                            <SheetContent side="bottom" className="h-[85vh] rounded-t-xl">
                                <SheetHeader>
                                    <SheetTitle>{t('reports.filters.title')}</SheetTitle>
                                </SheetHeader>
                                <div className="flex-1 overflow-y-auto px-4 pb-8 space-y-6">
                                    {/* Period */}
                                    <div className="space-y-3">
                                        <Label className="text-sm font-medium flex items-center gap-2">
                                            <Calendar className="size-4" />
                                            {t('reports.filters.period')}
                                        </Label>
                                        <PeriodTypeBadges />
                                        <PeriodSelector />
                                    </div>

                                    {/* Comparison */}
                                    <div className="space-y-2">
                                        <Label className="text-sm font-medium">{t('reports.filters.compareWith')}</Label>
                                        <ComparisonSelector />
                                    </div>

                                    {/* Entity filters */}
                                    <div className="space-y-4">
                                        <Label className="text-sm font-medium flex items-center gap-2">
                                            <Filter className="size-4" />
                                            {t('reports.filters.filterBy')}
                                        </Label>
                                        {entityFilters(true)}
                                    </div>

                                    {/* Reset */}
                                    {hasActiveFilters && (
                                        <Button
                                            variant="outline"
                                            onClick={() => {
                                                onReset()
                                                setSheetOpen(false)
                                            }}
                                            className="w-full"
                                        >
                                            <RotateCcw className="size-4 mr-2" />
                                            {t('reports.filters.resetFilters')}
                                        </Button>
                                    )}
                                </div>
                            </SheetContent>
                        </Sheet>
                    </div>
                </CardContent>
            </Card>
        )
    }

    // Desktop version
    return (
        <Card className="mb-6">
            <CardContent className="py-4">
                <div className="flex flex-wrap items-center gap-3">
                    {/* Period Type & Selection */}
                    <div className="flex items-center gap-2">
                        <Calendar className="size-4 text-muted-foreground" />
                        <PeriodTypeBadges />
                        <PeriodSelector />
                    </div>

                    <div className="h-6 w-px bg-border" />

                    {/* Comparison Period */}
                    <ComparisonSelector />

                    <div className="h-6 w-px bg-border" />

                    {/* Filter Dropdowns */}
                    <div className="flex items-center gap-2">
                        <Filter className="size-4 text-muted-foreground" />
                        {entityFilters()}
                    </div>

                    {/* Reset Button */}
                    {hasActiveFilters && (
                        <>
                            <div className="h-6 w-px bg-border" />
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={onReset}
                                className="h-8 text-muted-foreground"
                            >
                                <RotateCcw className="size-3 mr-1" />
                                {t('reports.filters.reset')}
                            </Button>
                        </>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}