import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useTransactionReportByCategory } from '@/hooks'
import type { ReportFilters } from '../types'
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

    const items = useMemo(() => (data?.items ?? []).map((item) => ({
        name: item.name,
        value: item.value,
        color: item.color,
    })), [data, i18n.language])

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
