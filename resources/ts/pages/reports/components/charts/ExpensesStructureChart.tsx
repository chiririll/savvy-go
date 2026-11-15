import { TransactionStructureChart } from './TransactionStructureChart'
import type { ReportFilters } from '../../types'

export function ExpensesStructureChart({ filters }: { filters: ReportFilters }) {
    return <TransactionStructureChart type="expense" filters={filters} />
}
