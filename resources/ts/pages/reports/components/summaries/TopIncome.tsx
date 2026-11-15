import { TopTransactions } from './TopTransactions'
import type { ReportFilters } from '../../types'

export function TopIncome({ filters, limit }: { filters: ReportFilters; limit?: number }) {
    return <TopTransactions type="income" filters={filters} limit={limit} />
}
