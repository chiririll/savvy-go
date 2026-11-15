import { TransactionTypeSummary } from '../components/summaries/TransactionTypeSummary'
import { ExpensesStructureChart } from '../components/charts/ExpensesStructureChart'
import { ExpensesDynamicsChart } from '../components/charts/ExpensesDynamicsChart'
import { TopExpenses } from '../components/summaries/TopExpenses'
import type { ReportFilters } from '../types'

export function ExpensesTab({ filters }: { filters: ReportFilters }) {
    return (
        <div className="space-y-6">
            <TransactionTypeSummary filters={filters} type="expense" />
            <ExpensesStructureChart filters={filters} />
            <ExpensesDynamicsChart filters={filters} />
            <TopExpenses filters={filters} />
        </div>
    )
}
