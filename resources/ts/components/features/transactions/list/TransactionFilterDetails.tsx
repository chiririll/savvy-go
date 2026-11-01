import { useEffect, useState, type ReactNode } from 'react'
import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import type { Account, Category, Currency, Tag } from '@/types'
import type { TransactionListFilters } from '@/hooks/use-transaction-list-filters'

function FilterGroup({ label, children }: { label: string; children: ReactNode }) {
    return (
        <div>
            <p className="text-sm font-medium mb-2">{label}</p>
            {children}
        </div>
    )
}

function ToggleBadge({ selected, onClick, children }: { selected: boolean; onClick: () => void; children: ReactNode }) {
    return (
        <Badge
            asChild
            variant={selected ? 'default' : 'outline'}
            className={cn('cursor-pointer transition-colors', selected ? 'hover:bg-primary/80' : 'hover:bg-muted')}
        >
            <button type="button" aria-pressed={selected} onClick={onClick}>
                {children}
            </button>
        </Badge>
    )
}

/** Free-text field; it sends the query once typing pauses. */
function SearchField({ value, onCommit }: { value: string; onCommit: (value: string) => void }) {
    const { t } = useTranslation('pages')
    const [text, setText] = useState(value)

    // The palette or a link can change the query from outside.
    useEffect(() => setText(value), [value])

    useEffect(() => {
        if (text.trim() === value.trim()) return
        const timer = setTimeout(() => onCommit(text), 300)
        return () => clearTimeout(timer)
    }, [text, value, onCommit])

    return (
        <div className="relative">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
                type="search"
                value={text}
                onChange={(event) => setText(event.target.value)}
                placeholder={t('transactions.searchPlaceholder')}
                aria-label={t('transactions.search')}
                className="pl-9"
            />
        </div>
    )
}

interface TransactionFilterDetailsProps {
    list: TransactionListFilters
    categories: Category[]
    tags?: Tag[]
    accounts?: Account[]
    currencies?: Currency[]
}

/** The search, date range, account, currency, category and tag filters inside the collapsible panel. */
export function TransactionFilterDetails({ list, categories, tags, accounts, currencies }: TransactionFilterDetailsProps) {
    const { t } = useTranslation('pages')
    const { params, setDateRange, toggleCategory, toggleTag, toggleAccount, toggleCurrency, setQuery } = list

    return (
        <Card>
            <CardContent className="pt-4 space-y-4">
                <SearchField value={params.q ?? ''} onCommit={setQuery} />

                <FilterGroup label={t('transactions.dateRange')}>
                    <div className="flex items-center gap-2">
                        <Input
                            type="date"
                            aria-label={t('transactions.dateFrom')}
                            value={params.startDate ?? ''}
                            onChange={(event) => setDateRange('startDate', event.target.value)}
                            className="w-auto"
                        />
                        <span className="text-muted-foreground">{t('transactions.to')}</span>
                        <Input
                            type="date"
                            aria-label={t('transactions.dateTo')}
                            value={params.endDate ?? ''}
                            onChange={(event) => setDateRange('endDate', event.target.value)}
                            className="w-auto"
                        />
                    </div>
                </FilterGroup>

                {accounts && accounts.length > 0 && (
                    <FilterGroup label={t('transactions.accounts')}>
                        <div className="flex flex-wrap gap-2">
                            {accounts.map((account) => (
                                <ToggleBadge
                                    key={account.id}
                                    selected={params.accountIds.includes(account.id)}
                                    onClick={() => toggleAccount(account.id)}
                                >
                                    {account.name}
                                </ToggleBadge>
                            ))}
                        </div>
                    </FilterGroup>
                )}

                {currencies && currencies.length > 1 && (
                    <FilterGroup label={t('transactions.currencies')}>
                        <div className="flex flex-wrap gap-2">
                            {currencies.map((currency) => (
                                <ToggleBadge
                                    key={currency.code}
                                    selected={params.currency.includes(currency.code)}
                                    onClick={() => toggleCurrency(currency.code)}
                                >
                                    {currency.code}
                                </ToggleBadge>
                            ))}
                        </div>
                    </FilterGroup>
                )}

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
                                    #{localizeDefaultName(tag.name, 'tags')}
                                </ToggleBadge>
                            ))}
                        </div>
                    </FilterGroup>
                )}
            </CardContent>
        </Card>
    )
}
