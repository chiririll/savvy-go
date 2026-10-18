import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Calendar, Filter, RotateCcw, SlidersHorizontal } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { useIsMobile } from '@/hooks/use-mobile'
import { testId } from '@/lib/test-id'
import type { ReportFilters } from '../types'
import { ComparisonSelector } from './filters/ComparisonSelector'
import { EntityFilters } from './filters/EntityFilters'
import { PeriodSelector } from './filters/PeriodSelector'
import { PeriodTypeBadges } from './filters/PeriodTypeBadges'
import { cn } from '@/lib/utils'

interface FiltersBarProps {
    filters: ReportFilters
    onFilterChange: <K extends keyof ReportFilters>(key: K, value: ReportFilters[K]) => void
    onToggleArrayFilter: (key: 'accountIds' | 'categoryIds' | 'tagIds', id: number) => void
    onReset: () => void
}

export function FiltersBar({ filters, onFilterChange, onToggleArrayFilter, onReset }: FiltersBarProps) {
    const { t } = useTranslation('pages')
    const isMobile = useIsMobile()
    const [sheetOpen, setSheetOpen] = useState(false)

    const activeFiltersCount = filters.accountIds.length + filters.categoryIds.length + filters.tagIds.length
    const hasActiveFilters = activeFiltersCount > 0

    const periodBadges = (scroll = false) => (
        <PeriodTypeBadges
            value={filters.periodType}
            onChange={(type) => onFilterChange('periodType', type)}
            scroll={scroll}
        />
    )
    const periodSelector = <PeriodSelector filters={filters} onFilterChange={onFilterChange} />
    const comparison = (
        <ComparisonSelector value={filters.compareWith} onChange={(val) => onFilterChange('compareWith', val)} />
    )

    if (isMobile) {
        return (
            <div className="mb-6">
                <div className="flex items-center justify-between gap-2">
                    <div
                        className={cn(
                            'min-w-0 flex-1 overflow-x-auto [scrollbar-width:none]',
                            '[&::-webkit-scrollbar]:hidden',
                        )}
                    >
                        {periodBadges(true)}
                    </div>

                    <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
                        <SheetTrigger asChild>
                            <Button variant="outline" size="sm" className="shrink-0" {...testId('filters')}>
                                <SlidersHorizontal className="size-4" />
                                {hasActiveFilters && (
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
                                <div className="space-y-3">
                                    <Label className="text-sm font-medium flex items-center gap-2">
                                        <Calendar className="size-4" />
                                        {t('reports.filters.period')}
                                    </Label>
                                    {periodBadges()}
                                    {periodSelector}
                                </div>

                                <div className="space-y-2">
                                    <Label className="text-sm font-medium">{t('reports.filters.compareWith')}</Label>
                                    {comparison}
                                </div>

                                <div className="space-y-4">
                                    <Label className="text-sm font-medium flex items-center gap-2">
                                        <Filter className="size-4" />
                                        {t('reports.filters.filterBy')}
                                    </Label>
                                    <EntityFilters filters={filters} onToggle={onToggleArrayFilter} inSheet />
                                </div>

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
            </div>
        )
    }

    // Desktop: period on the left, everything else on the right.
    return (
        <div className="mb-6 flex flex-wrap items-center justify-between gap-x-6 gap-y-3">
            <div className="flex flex-wrap items-center gap-2">
                <Calendar className="size-4 text-muted-foreground" />
                {periodBadges()}
                {periodSelector}
            </div>

            <div className="ml-auto flex flex-wrap items-center justify-end gap-3">
                {comparison}

                <div className="h-6 w-px bg-border" />

                <div className="flex items-center gap-2">
                    <Filter className="size-4 text-muted-foreground" />
                    <EntityFilters filters={filters} onToggle={onToggleArrayFilter} />
                </div>

                {hasActiveFilters && (
                    <>
                        <div className="h-6 w-px bg-border" />
                        <Button variant="ghost" size="sm" onClick={onReset} className="h-8 text-muted-foreground">
                            <RotateCcw className="size-3 mr-1" />
                            {t('reports.filters.reset')}
                        </Button>
                    </>
                )}
            </div>
        </div>
    )
}
