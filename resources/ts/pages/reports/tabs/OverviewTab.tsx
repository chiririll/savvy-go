import { useTranslation } from 'react-i18next'
import { OverviewMetricCards } from '../components/OverviewMetricCards'
import { SankeyDiagram } from '../components/SankeyDiagram'
import { ExpensePaceChart } from '../components/ExpensePaceChart'
import { ExpensesByCategory } from '../components/ExpensesByCategory'
import { ActivityHeatmap } from '../components/ActivityHeatmap'
import { useOverviewMetrics } from '@/hooks'
import type { ReportFilters } from '../types'

interface OverviewTabProps {
    filters: ReportFilters
}

export function OverviewTab({ filters }: OverviewTabProps) {
    const { t } = useTranslation('pages')
    const { data, isLoading, error } = useOverviewMetrics(filters)

    if (error) {
        return (
            <div className="text-center py-8 text-destructive">
                {t('reports.errors.overview')}
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* Key Metrics - 4 cards in a row */}
            <OverviewMetricCards
                className="sm:grid-cols-2 lg:grid-cols-4"
                metrics={[
                    { key: 'income', title: t('reports.metrics.income'), type: 'income' },
                    { key: 'expenses', title: t('reports.metrics.expenses'), type: 'expense' },
                    { key: 'netCashFlow', title: t('reports.metrics.netCashFlow'), type: 'net' },
                    { key: 'savingsRate', title: t('reports.metrics.savingsRate'), type: 'percent' },
                ]}
                data={data}
                isLoading={isLoading}
                compareWith={filters.compareWith}
            />

            {/* Sankey Diagram */}
            <SankeyDiagram filters={filters} />

            {/* Expense Pace Chart and Activity Heatmap */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <ExpensePaceChart filters={filters} />
                <ActivityHeatmap filters={filters} />
            </div>

            {/* Expenses by Category */}
            <ExpensesByCategory filters={filters} />
        </div>
    )
}
