import { TopTransactions } from './TopTransactions'
import type { ReportFilters } from '../../types'

export function TopExpenses({ filters, limit }: { filters: ReportFilters; limit?: number }) {
    return <TopTransactions type="expense" filters={filters} limit={limit} />
}
