import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
    useAccounts,
    useBalanceComparison,
    useBudgets,
    useCategorySummary,
    useDebtsWithSummary,
    usePendingSummary,
    useTotalBalance,
    useTransactions,
} from '@/hooks'
import { useOverviewMetrics } from '@/hooks/use-reports'
import { BalanceDynamicsChart } from '@/components/features/accounts/BalanceDynamicsChart'
import { useCreateTransactionDialog } from '@/components/features/transactions'
import { AccountBalancesCard } from '@/components/features/dashboard/AccountBalancesCard'
import { BudgetsCard } from '@/components/features/dashboard/BudgetsCard'
import { DebtsCard } from '@/components/features/dashboard/DebtsCard'
import { ExpensesByCategoryCard } from '@/components/features/dashboard/ExpensesByCategoryCard'
import { PeriodPicker } from '@/components/features/dashboard/PeriodPicker'
import { LargestTransactionsCard } from '@/components/features/dashboard/LargestTransactionsCard'
import { DashboardStats } from '@/components/features/dashboard/StatCards'
import {
    getPresetDates,
    percentChange,
    resolvePeriodDates,
    toReportFilters,
    type PeriodPreset,
} from '@/components/features/dashboard/period'

export default function DashboardPage() {
    const { t } = useTranslation('pages')
    const { openCreate, openEdit } = useCreateTransactionDialog()
    const { data: balance } = useTotalBalance()
    const { data: accounts } = useAccounts({ active: true, exclude_debts: true })

    const [period, setPeriod] = useState<PeriodPreset>('last_30_days')
    const [customStartDate, setCustomStartDate] = useState('')
    const [customEndDate, setCustomEndDate] = useState('')

    const periodDates = useMemo(
        () => resolvePeriodDates(period, customStartDate, customEndDate),
        [period, customStartDate, customEndDate],
    )
    const reportFilters = useMemo(
        () => toReportFilters(period, customStartDate, customEndDate),
        [period, customStartDate, customEndDate],
    )

    // Always the last 30 days, whatever the period picker says. The API sorts by amount in the base currency.
    const largestRange = useMemo(() => getPresetDates('last_30_days'), [])
    const { data: largestTransactions } = useTransactions({
        ...largestRange,
        per_page: 5,
        status: 'confirmed',
        sort_by: 'amount',
        sort_direction: 'desc',
    })
    const { data: pendingSummary } = usePendingSummary()
    const { data: expensesByCategory } = useCategorySummary({ type: 'expense', ...periodDates })
    const { data: budgets } = useBudgets()
    const { data: debtsData } = useDebtsWithSummary()
    const { data: overviewData } = useOverviewMetrics(reportFilters)
    const { data: balanceComparison } = useBalanceComparison(periodDates)

    const activeBudgets = useMemo(() => budgets?.filter((b) => b.isActive).slice(0, 4) ?? [], [budgets])
    const activeDebts = debtsData?.data?.filter((d) => !d.isPaidOff).slice(0, 4) ?? []

    const currency = balance?.currency
    const periodIncome = overviewData?.income.value ?? 0
    const periodExpense = overviewData?.expenses.value ?? 0

    const handlePeriodChange = (value: PeriodPreset) => {
        setPeriod(value)
        if (value !== 'custom') {
            setCustomStartDate('')
            setCustomEndDate('')
        }
    }

    return (
        <div className="min-w-0 space-y-6">
            <div className="flex min-w-0 flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">{t('dashboard.title')}</h1>
                    <p className="text-muted-foreground">{t('dashboard.welcome')}</p>
                </div>
                <PeriodPicker
                    period={period}
                    onPeriodChange={handlePeriodChange}
                    customStartDate={customStartDate}
                    customEndDate={customEndDate}
                    onCustomStartChange={setCustomStartDate}
                    onCustomEndChange={setCustomEndDate}
                />
            </div>

            <DashboardStats
                currency={currency}
                totalBalance={balance?.totalBalance ?? 0}
                periodIncome={periodIncome}
                periodExpense={periodExpense}
                balanceChange={
                    balanceComparison ? percentChange(balanceComparison.current, balanceComparison.previous) : null
                }
                incomeChange={percentChange(periodIncome, overviewData?.income?.previous)}
                expenseChange={percentChange(periodExpense, overviewData?.expenses?.previous)}
                pendingSummary={pendingSummary}
            />

            <div className="grid min-w-0 gap-4 lg:grid-cols-3">
                <BalanceDynamicsChart
                    className="min-w-0 overflow-x-auto lg:col-span-2"
                    startDate={periodDates.start_date}
                    endDate={periodDates.end_date}
                />

                <AccountBalancesCard accounts={accounts} openCreate={openCreate} />
            </div>

            <div className="grid min-w-0 gap-4 lg:grid-cols-2">
                <ExpensesByCategoryCard summary={expensesByCategory} currency={currency} />
                <LargestTransactionsCard transactions={largestTransactions?.data} onEdit={openEdit} />
            </div>

            <BudgetsCard budgets={activeBudgets} />

            <DebtsCard summary={debtsData?.summary} debts={activeDebts} />
        </div>
    )
}
