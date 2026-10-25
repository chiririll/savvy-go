import { useTranslation } from 'react-i18next'
import { OverviewMetricCards } from '../components/OverviewMetricCards'
import { CashFlowChart } from '../components/CashFlowChart'
import { useOverviewMetrics } from '@/hooks'
import type { ReportFilters } from '../types'

interface CashFlowTabProps {
    filters: ReportFilters
}

export function CashFlowTab({ filters }: CashFlowTabProps) {
    const { t } = useTranslation('pages')
    const { data, isLoading, error } = useOverviewMetrics(filters)

    if (error) {
        return (
            <div className="text-center py-8 text-destructive">
                {t('reports.errors.cashflow')}
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* Cash Flow Metrics - 3 cards in a row */}
            <OverviewMetricCards
                className="sm:grid-cols-3"
                metrics={[
                    { key: 'income', title: t('reports.metrics.inflow'), type: 'income' },
                    { key: 'expenses', title: t('reports.metrics.outflow'), type: 'expense' },
                    { key: 'netCashFlow', title: t('reports.metrics.netCashFlow'), type: 'net' },
                ]}
                data={data}
                isLoading={isLoading}
                compareWith={filters.compareWith}
            />

            {/* Cash Flow Chart */}
            <CashFlowChart filters={filters} />
        </div>
    )
}
