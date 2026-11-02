import { useOutletContext } from 'react-router-dom'
import { OverviewTab, CashFlowTab, ExpensesTab, IncomeTab, NetWorthTab } from './tabs'
import type { ReportFilters } from './types'

/** The filters the reports page keeps above its outlet. */
export const useReportFilters = () => useOutletContext<ReportFilters>()

export const OverviewRoute = () => <OverviewTab filters={useReportFilters()} />
export const CashFlowRoute = () => <CashFlowTab filters={useReportFilters()} />
export const ExpensesRoute = () => <ExpensesTab filters={useReportFilters()} />
export const IncomeRoute = () => <IncomeTab filters={useReportFilters()} />
export const NetWorthRoute = () => <NetWorthTab filters={useReportFilters()} />
