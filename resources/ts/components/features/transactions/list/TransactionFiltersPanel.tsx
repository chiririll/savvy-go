import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Filter, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { Account, Category, Currency, Tag } from '@/types'
import type { TransactionListFilters } from '@/hooks/use-transaction-list-filters'
import { TransactionFilterDetails } from './TransactionFilterDetails'
import { TransactionSortSelect, TransactionTypeFilter } from './TransactionTypeAndSort'
import { cn } from '@/lib/utils'

interface TransactionFiltersPanelProps {
    list: TransactionListFilters
    categories?: Category[]
    tags?: Tag[]
    accounts?: Account[]
    currencies?: Currency[]
}

export function TransactionFiltersPanel({ list, categories, tags, accounts, currencies }: TransactionFiltersPanelProps) {
    const { t } = useTranslation('pages')
    const [filtersOpen, setFiltersOpen] = useState(false)
    const { params, activeFiltersCount, setType, setStatus, setSort, clearFilters } = list

    const filteredCategories =
        categories?.filter((category) => !params.type || params.type === 'transfer' || category.type === params.type) ??
        []

    return (
        <>
            <Tabs
                value={params.status ?? 'all'}
                onValueChange={(value) => setStatus(value === 'pending' ? 'pending' : null)}
                className="mb-4"
            >
                <TabsList>
                    {/* The tabs switch a filter; there are no panels for aria-controls to point at. */}
                    <TabsTrigger value="all" aria-controls={undefined}>{t('transactions.tabs.confirmed')}</TabsTrigger>
                    <TabsTrigger value="pending" aria-controls={undefined}>{t('transactions.tabs.pending')}</TabsTrigger>
                </TabsList>
            </Tabs>

            <Collapsible open={filtersOpen} onOpenChange={setFiltersOpen} className="mb-4">
                <div className="flex min-w-0 items-center gap-1.5 sm:gap-2">
                    <TransactionTypeFilter value={params.type} onChange={setType} />
                    <div className="ml-auto flex shrink-0 items-center gap-1.5 sm:gap-2">
                        <TransactionSortSelect sortBy={params.sortBy} sortDir={params.sortDir} onChange={setSort} />
                        <CollapsibleTrigger asChild>
                            <Button
                                variant="outline"
                                size="sm"
                                className="relative px-2.5"
                                title={t('transactions.filters')}
                            >
                                <Filter className="size-4" />
                                <span className="sr-only sm:not-sr-only">{t('transactions.filters')}</span>
                                {activeFiltersCount > 0 && (
                                    <Badge
                                        variant="secondary"
                                        className={cn(
                                            'px-1.5 py-0 text-xs max-sm:absolute max-sm:-top-1.5 max-sm:-right-1.5',
                                        )}
                                    >
                                        {activeFiltersCount}
                                    </Badge>
                                )}
                            </Button>
                        </CollapsibleTrigger>
                        {activeFiltersCount > 0 && (
                            <Button
                                variant="ghost"
                                size="sm"
                                className="px-2.5"
                                onClick={clearFilters}
                                aria-label={t('transactions.clear')}
                                title={t('transactions.clear')}
                            >
                                <X className="size-4" />
                                <span className="hidden sm:inline">{t('transactions.clear')}</span>
                            </Button>
                        )}
                    </div>
                </div>
                <CollapsibleContent className="mt-4 space-y-4">
                    <TransactionFilterDetails
                        list={list}
                        categories={filteredCategories}
                        tags={tags}
                        accounts={accounts}
                        currencies={currencies}
                    />
                </CollapsibleContent>
            </Collapsible>
        </>
    )
}
