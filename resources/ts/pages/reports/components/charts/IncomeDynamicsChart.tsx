import { TransactionDynamicsChart } from './TransactionDynamicsChart'
import type { ReportFilters } from '../../types'

export function IncomeDynamicsChart({ filters }: { filters: ReportFilters }) {
    return <TransactionDynamicsChart type="income" filters={filters} />
}
