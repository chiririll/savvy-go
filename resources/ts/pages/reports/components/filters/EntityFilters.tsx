import { useTranslation } from 'react-i18next'
import { useAccounts, useCategories, useTags } from '@/hooks'
import { localizeDefaultName } from '@/lib/localized-name'
import type { ReportFilters } from '../../types'
import { EntityFilter } from './EntityFilter'

interface EntityFiltersProps {
    filters: ReportFilters
    onToggle: (key: 'accountIds' | 'categoryIds' | 'tagIds', id: number) => void
    inSheet?: boolean
}

export function EntityFilters({ filters, onToggle, inSheet = false }: EntityFiltersProps) {
    const { t } = useTranslation('pages')
    const { data: categories } = useCategories()
    const { data: accounts } = useAccounts()
    const { data: tags } = useTags()

    return (
        <>
            <EntityFilter
                label={t('reports.filters.accounts')}
                items={accounts}
                selectedIds={filters.accountIds}
                onToggle={(id) => onToggle('accountIds', id)}
                inSheet={inSheet}
            />
            <EntityFilter
                label={t('reports.filters.categories')}
                items={categories}
                selectedIds={filters.categoryIds}
                onToggle={(id) => onToggle('categoryIds', id)}
                inSheet={inSheet}
                formatName={(item) => localizeDefaultName(item.name)}
                popoverClassName="w-64 p-2"
            />
            <EntityFilter
                label={t('reports.filters.tags')}
                items={tags}
                selectedIds={filters.tagIds}
                onToggle={(id) => onToggle('tagIds', id)}
                inSheet={inSheet}
                formatName={(item) => `#${item.name}`}
                popoverClassName="w-48 p-2"
                hideWhenEmpty
            />
        </>
    )
}
