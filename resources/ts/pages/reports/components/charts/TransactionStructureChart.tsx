import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useTransactionReportByCategory } from '@/hooks'
import { localizeDefaultName } from '@/lib/format/localized-name'
import type { ReportFilters } from '../../types'
import type { ReportTransactionType } from '@/api/reports'
import { StructureChart } from './StructureChart'

interface TransactionStructureChartProps {
    filters: ReportFilters
    type: ReportTransactionType
}

export function TransactionStructureChart({ filters, type }: TransactionStructureChartProps) {
    const { t, i18n } = useTranslation('pages')
    const copy = type === 'income'
        ? {
            title: t('reports.incomeStructure.title'),
            subtitle: t('reports.incomeStructure.subtitle'),
            noData: t('reports.incomeStructure.noData'),
        }
        : {
            title: t('reports.expensesStructure.title'),
            subtitle: t('reports.expensesStructure.subtitle'),
            noData: t('reports.expensesStructure.noData'),
        }
    const { data, isLoading } = useTransactionReportByCategory(filters, type)

    const items = useMemo(() => {
        const parents = new Map((data?.parents ?? []).map((p) => [p.id, p]))
        const hasChildren = new Set((data?.items ?? []).map((item) => item.parentId).filter(Boolean))
        return (data?.items ?? []).map((item) => {
            // A parent's own lines sit beside its children's, named apart from the group.
            const own = hasChildren.has(item.id)
            const parent = parents.get(item.parentId ?? 0) ?? (own ? item : undefined)
            const name = localizeDefaultName(item.name)
            return {
                name: own ? t('reports.structure.ownLines', { name }) : name,
                value: item.value,
                color: item.color,
                parent: parent && { name: localizeDefaultName(parent.name), color: parent.color },
            }
        })
    }, [data, i18n.language])

    return (
        <StructureChart
            {...copy}
            items={items}
            total={data?.total || 0}
            currency={data?.currency}
            isLoading={isLoading}
        />
    )
}
