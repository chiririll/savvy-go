import { TransactionTypeSummary } from '../components/summaries/TransactionTypeSummary'
import { IncomeStructureChart } from '../components/charts/IncomeStructureChart'
import { IncomeDynamicsChart } from '../components/charts/IncomeDynamicsChart'
import { TopIncome } from '../components/summaries/TopIncome'
import type { ReportFilters } from '../types'

export function IncomeTab({ filters }: { filters: ReportFilters }) {
    return (
        <div className="space-y-6">
            <TransactionTypeSummary filters={filters} type="income" />
            <IncomeStructureChart filters={filters} />
            <IncomeDynamicsChart filters={filters} />
            <TopIncome filters={filters} />
        </div>
    )
}
