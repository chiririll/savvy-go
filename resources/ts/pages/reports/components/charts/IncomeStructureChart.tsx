import { TransactionStructureChart } from './TransactionStructureChart'
import type { ReportFilters } from '../../types'

export function IncomeStructureChart({ filters }: { filters: ReportFilters }) {
    return <TransactionStructureChart type="income" filters={filters} />
}
