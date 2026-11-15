import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { FormControl } from '@/components/ui/form'
import { useCategories } from '@/hooks'
import { localizeDefaultName } from '@/lib/format/localized-name'
import { testId } from '@/lib/test-id'
import { categoryTree } from '@/lib/categories/tree'

const NONE = 'none'

interface CategorySelectProps {
    value?: number | null
    onChange: (value: number) => void
    type?: 'income' | 'expense'
    excludeIds?: number[]
    /** Offer only these categories (subcategories are not indented unless their parent is offered too). */
    onlyIds?: number[]
    /** Adds a leading option with this label that calls onClear. */
    noneLabel?: string
    onClear?: () => void
    placeholder?: string
    disabled?: boolean
    sortByPopularity?: boolean
    plain?: boolean
    /** Accessible name of the trigger; set it when no visible label points at the select. */
    ariaLabel?: string
    /** Marks the field for scripts/screenshots. */
    testName?: string
}

export function CategorySelect({
    value,
    onChange,
    type,
    excludeIds,
    onlyIds,
    noneLabel,
    onClear,
    placeholder,
    disabled,
    sortByPopularity = true,
    plain,
    ariaLabel,
    testName,
}: CategorySelectProps) {
    const { t } = useTranslation('forms')
    const { data: categories } = useCategories()

    const filteredCategories = useMemo(() => {
        let filtered = type
            ? (categories?.filter((category) => category.type === type) ?? [])
            : [...(categories ?? [])]
        if (excludeIds?.length) {
            filtered = filtered.filter((category) => !excludeIds.includes(category.id))
        }
        if (onlyIds) {
            filtered = filtered.filter((category) => onlyIds.includes(category.id))
        }
        if (sortByPopularity) {
            filtered.sort((a, b) => (b.transactionsCount ?? 0) - (a.transactionsCount ?? 0))
        }
        return categoryTree(filtered)
    }, [categories, type, excludeIds, onlyIds, sortByPopularity])

    const trigger = (
        <SelectTrigger className="w-full" aria-label={ariaLabel} {...(testName ? testId(testName) : {})}>
            <SelectValue placeholder={placeholder ?? t('selectCategory')} />
        </SelectTrigger>
    )

    return (
        <Select
            onValueChange={(val) => (val === NONE ? onClear?.() : onChange(Number(val)))}
            value={value ? value.toString() : noneLabel ? NONE : undefined}
            disabled={disabled}
        >
            {plain ? trigger : <FormControl>{trigger}</FormControl>}
            <SelectContent>
                {noneLabel && <SelectItem value={NONE}>{noneLabel}</SelectItem>}
                {filteredCategories.map(({ category, parent }) => (
                    <SelectItem
                        key={category.id}
                        value={category.id.toString()}
                        className={parent ? 'pl-8' : undefined}
                    >
                        <div className="flex items-center gap-2">
                            <span
                                className="w-5 h-5 rounded flex items-center justify-center text-xs text-white"
                                style={{ backgroundColor: category.color ?? undefined }}
                            >
                                {category.icon}
                            </span>
                            <span>{localizeDefaultName(category.name)}</span>
                        </div>
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    )
}
