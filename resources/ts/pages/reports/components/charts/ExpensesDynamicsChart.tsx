import { TransactionDynamicsChart } from './TransactionDynamicsChart'
import type { ReportFilters } from '../../types'

export function ExpensesDynamicsChart({ filters }: { filters: ReportFilters }) {
    return <TransactionDynamicsChart type="expense" filters={filters} />
}
