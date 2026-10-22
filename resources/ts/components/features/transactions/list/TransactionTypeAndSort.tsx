import { useTranslation } from 'react-i18next'
import type { LucideIcon } from 'lucide-react'
import { ArrowUpDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { TRANSACTION_TYPE_OPTIONS } from '@/constants/transactions'
import { sortOptionLabel, transactionTypeLabel } from '@/lib/labels'

type TypeFilter = 'income' | 'expense' | 'transfer' | null

const TYPE_FILTERS: {
    value: TypeFilter
    labelKey: 'all' | 'income' | 'expense' | 'transfer'
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

export function TransactionTypeFilter({
    value,
    onChange,
}: {
    value: TypeFilter
    onChange: (value: TypeFilter) => void
}) {
    const { t } = useTranslation('pages')

    return (
        <div className="flex min-w-0 gap-1.5 sm:gap-2">
            {TYPE_FILTERS.map(({ value: type, labelKey, icon: Icon }) => {
                const label = labelKey === 'all' ? t('common:actions.all') : transactionTypeLabel(t, labelKey)

                return (
                    <Button
                        key={labelKey}
                        variant={value === type ? 'default' : 'outline'}
                        size="sm"
                        className="min-w-0 px-2.5 sm:shrink-0"
                        onClick={() => onChange(type)}
                        aria-pressed={value === type}
                        aria-label={label}
                        title={label}
                    >
                        {Icon && <Icon className="size-4" />}
                        <span className={cn('truncate', Icon && 'hidden sm:inline')}>{label}</span>
                    </Button>
                )
            })}
        </div>
    )
}

interface TransactionSortSelectProps {
    sortBy: 'date' | 'amount'
    sortDir: 'asc' | 'desc'
    onChange: (sortBy: 'date' | 'amount', sortDir: 'asc' | 'desc') => void
}

export function TransactionSortSelect({ sortBy, sortDir, onChange }: TransactionSortSelectProps) {
    const { t } = useTranslation('pages')
    const current = `${sortBy}:${sortDir}`

    return (
        <Select
            value={current}
            onValueChange={(val) => {
                const [by, dir] = val.split(':') as ['date' | 'amount', 'asc' | 'desc']
                onChange(by, dir)
            }}
        >
            <SelectTrigger
                className={cn(
                    'h-8 w-auto max-sm:px-2.5 max-sm:[&>svg:last-child]:hidden',
                    'max-sm:[&>[data-slot=select-value]]:hidden sm:h-9 sm:w-[180px]',
                )}
                aria-label={sortOptionLabel(t, SORT_OPTIONS.find((o) => o.value === current)?.labelKey ?? 'dateNewest')}
            >
                <ArrowUpDown className="size-4" />
                <SelectValue />
            </SelectTrigger>
            <SelectContent align="end">
                {SORT_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                        {sortOptionLabel(t, opt.labelKey)}
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    )
}
