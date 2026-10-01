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
import { localizeDefaultName } from '@/lib/localized-name'

interface CategorySelectProps {
    value?: number | null
    onChange: (value: number) => void
    type?: 'income' | 'expense'
    excludeIds?: number[]
    placeholder?: string
    disabled?: boolean
    sortByPopularity?: boolean
    plain?: boolean
}

export function CategorySelect({
    value,
    onChange,
    type,
    excludeIds,
    placeholder,
    disabled,
    sortByPopularity = true,
    plain,
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
        if (sortByPopularity) {
            return filtered.sort((a, b) => (b.transactionsCount ?? 0) - (a.transactionsCount ?? 0))
        }
        return filtered
    }, [categories, type, excludeIds, sortByPopularity])

    const trigger = (
        <SelectTrigger className="w-full">
            <SelectValue placeholder={placeholder ?? t('selectCategory')} />
        </SelectTrigger>
    )

    return (
        <Select
            onValueChange={(val) => onChange(Number(val))}
            value={value ? value.toString() : undefined}
            disabled={disabled}
        >
            {plain ? trigger : <FormControl>{trigger}</FormControl>}
            <SelectContent>
                {filteredCategories.map((category) => (
                    <SelectItem
                        key={category.id}
                        value={category.id.toString()}
                    >
                        <div className="flex items-center gap-2">
                            <span
                                className="w-5 h-5 rounded flex items-center justify-center text-xs text-white"
                                style={{ backgroundColor: category.color }}
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
