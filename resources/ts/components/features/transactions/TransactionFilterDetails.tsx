import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import type { Category, Tag } from '@/types'
import type { TransactionListFilters } from '@/hooks/use-transaction-list-filters'

function FilterGroup({ label, children }: { label: string; children: ReactNode }) {
    return (
        <div>
            <label className="text-sm font-medium mb-2 block">{label}</label>
            {children}
        </div>
    )
}

function ToggleBadge({ selected, onClick, children }: { selected: boolean; onClick: () => void; children: ReactNode }) {
    return (
        <Badge
            variant={selected ? 'default' : 'outline'}
            className={cn('cursor-pointer transition-colors', selected ? 'hover:bg-primary/80' : 'hover:bg-muted')}
            onClick={onClick}
        >
            {children}
        </Badge>
    )
}

interface TransactionFilterDetailsProps {
    list: TransactionListFilters
    categories: Category[]
    tags?: Tag[]
}

/** The date range, category and tag filters inside the collapsible panel. */
export function TransactionFilterDetails({ list, categories, tags }: TransactionFilterDetailsProps) {
    const { t } = useTranslation('pages')
    const { params, setDateRange, toggleCategory, toggleTag } = list

    return (
        <Card>
            <CardContent className="pt-4 space-y-4">
                <FilterGroup label={t('transactions.dateRange')}>
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
                </FilterGroup>

                {categories.length > 0 && (
                    <FilterGroup label={t('transactions.categories')}>
                        <div className="flex flex-wrap gap-2">
                            {categories.map((category) => (
                                <ToggleBadge
                                    key={category.id}
                                    selected={params.categoryIds.includes(category.id)}
                                    onClick={() => toggleCategory(category.id)}
                                >
                                    {category.icon} {localizeDefaultName(category.name)}
                                </ToggleBadge>
                            ))}
                        </div>
                    </FilterGroup>
                )}

                {tags && tags.length > 0 && (
                    <FilterGroup label={t('transactions.tags')}>
                        <div className="flex flex-wrap gap-2">
                            {tags.map((tag) => (
                                <ToggleBadge
                                    key={tag.id}
                                    selected={params.tagIds.includes(tag.id)}
                                    onClick={() => toggleTag(tag.id)}
                                >
                                    #{tag.name}
                                </ToggleBadge>
                            ))}
                        </div>
                    </FilterGroup>
                )}
            </CardContent>
        </Card>
    )
}
