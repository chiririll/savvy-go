import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
    Wallet,
    ArrowDownLeft,
    ArrowUpRight,
    ArrowRight,
    Calendar,
    Clock,
    Plus,
    ArrowLeftRight,
    PiggyBank,
    HandCoins,
    Banknote,
    TrendingDown,
    TrendingUp,
} from 'lucide-react'
import { Progress } from '@/components/ui/progress'
import { useTotalBalance, useTransactions, usePendingSummary, useAccounts, useCategorySummary, useBudgets, useDebtsWithSummary, useBalanceComparison } from '@/hooks'
import { useOverviewMetrics } from '@/hooks/use-reports'
import { cn, formatCurrency, formatDateLocal, formatYearMonth, addDaysLocal } from '@/lib/utils'
import { BalanceDynamicsChart } from '@/components/features/accounts/BalanceDynamicsChart'
import { TransactionRow, useCreateTransactionDialog } from '@/components/features/transactions'
import { localizeDefaultName } from '@/lib/localized-name'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from 'echarts-for-react'
import { useTheme } from '@/hooks/use-theme'
import { Link } from 'react-router-dom'
import { AccountType } from '@/types'
import { ACCOUNT_TYPE_CONFIG, CATEGORY_COLORS } from '@/constants'
import { DEFAULT_FILTERS, type ReportFilters } from '@/pages/reports/types'

type PeriodPreset = 'last_30_days' | 'this_month' | 'last_month' | 'last_3_months' | 'last_6_months' | 'this_year' | 'custom'


function getPresetDates(preset: PeriodPreset): { start_date: string; end_date: string } {
    const now = new Date()
    const year = now.getFullYear()
    const month = now.getMonth()

    switch (preset) {
        case 'last_30_days': {
            return {
                start_date: addDaysLocal(now, -29),
                end_date: formatDateLocal(now),
            }
        }
        case 'this_month': {
            const firstDay = new Date(year, month, 1)
            const lastDay = new Date(year, month + 1, 0)
            return {
                start_date: formatDateLocal(firstDay),
                end_date: formatDateLocal(lastDay),
            }
        }
        case 'last_month': {
            const firstDay = new Date(year, month - 1, 1)
            const lastDay = new Date(year, month, 0)
            return {
                start_date: formatDateLocal(firstDay),
                end_date: formatDateLocal(lastDay),
            }
        }
        case 'last_3_months': {
            const firstDay = new Date(year, month - 2, 1)
            const lastDay = new Date(year, month + 1, 0)
            return {
                start_date: formatDateLocal(firstDay),
                end_date: formatDateLocal(lastDay),
            }
        }
        case 'last_6_months': {
            const firstDay = new Date(year, month - 5, 1)
            const lastDay = new Date(year, month + 1, 0)
            return {
                start_date: formatDateLocal(firstDay),
                end_date: formatDateLocal(lastDay),
            }
        }
        case 'this_year': {
            const firstDay = new Date(year, 0, 1)
            const lastDay = new Date(year, 11, 31)
            return {
                start_date: formatDateLocal(firstDay),
                end_date: formatDateLocal(lastDay),
            }
        }
        default:
            return getPresetDates('last_30_days')
    }
}

function toReportFilters(
    preset: PeriodPreset,
    customStartDate: string,
    customEndDate: string,
): ReportFilters {
    const now = new Date()
    const useCustomDates = preset === 'custom' && Boolean(customStartDate && customEndDate)
    const dates = useCustomDates
        ? { start_date: customStartDate, end_date: customEndDate }
        : getPresetDates(preset === 'custom' ? 'last_30_days' : preset)

    const filters: ReportFilters = {
        ...DEFAULT_FILTERS,
        customStartDate: dates.start_date,
        customEndDate: dates.end_date,
        compareWith: 'previous_period',
    }

    switch (preset) {
        case 'last_30_days':
            return { ...filters, periodType: 'last_30_days' }
        case 'this_month':
            return { ...filters, periodType: 'month', selectedMonth: formatYearMonth(now) }
        case 'last_month':
            return {
                ...filters,
                periodType: 'month',
                selectedMonth: formatYearMonth(new Date(now.getFullYear(), now.getMonth() - 1, 1)),
            }
        case 'this_year':
            return { ...filters, periodType: 'year', selectedYear: String(now.getFullYear()) }
        default:
            return { ...filters, periodType: 'custom' }
    }
}

export default function DashboardPage() {
    const { t, i18n } = useTranslation('pages')
    const { t: tCommon } = useTranslation('common')
    const { t: tNav } = useTranslation('nav')
    const { theme } = useTheme()
    const { openCreate, openEdit } = useCreateTransactionDialog()
    const { data: balance } = useTotalBalance()
    const { data: accounts } = useAccounts({ active: true, exclude_debts: true })

    const [period, setPeriod] = useState<PeriodPreset>('last_30_days')
    const [customStartDate, setCustomStartDate] = useState('')
    const [customEndDate, setCustomEndDate] = useState('')

    const periodDates = useMemo(() => {
        if (period === 'custom' && customStartDate && customEndDate) {
            return { start_date: customStartDate, end_date: customEndDate }
        }
        return getPresetDates(period)
    }, [period, customStartDate, customEndDate])

    const reportFilters = useMemo(
        () => toReportFilters(period, customStartDate, customEndDate),
        [period, customStartDate, customEndDate],
    )

    const { data: recentTransactions } = useTransactions({ per_page: 5, status: 'confirmed' })
    const { data: pendingSummary } = usePendingSummary()
    const { data: expensesByCategory } = useCategorySummary({
        type: 'expense',
        ...periodDates,
    })
    const { data: budgets } = useBudgets()
    const { data: debtsData } = useDebtsWithSummary()
    const { data: overviewData } = useOverviewMetrics(reportFilters)
    const { data: balanceComparison } = useBalanceComparison()
    const activeBudgets = useMemo(() => {
        return budgets?.filter(b => b.isActive).slice(0, 4) ?? []
    }, [budgets])

    const activeDebts = debtsData?.data?.filter(d => !d.isPaidOff).slice(0, 4) ?? []
    const debtSummary = debtsData?.summary

    const totalBalance = balance?.total_balance ?? 0
    const currency = balance?.currency
    const periodIncome = overviewData?.income.value ?? 0
    const periodExpense = overviewData?.expenses.value ?? 0

    // Calculate percentage changes
    const balanceChange = useMemo(() => {
        if (balanceComparison?.previous == null) return null
        const current = balanceComparison.current
        const previous = balanceComparison.previous
        if (previous === 0 && current === 0) return 0
        if (previous === 0) return 100
        return ((current - previous) / Math.abs(previous)) * 100
    }, [balanceComparison])

    const incomeChange = useMemo(() => {
        if (overviewData?.income?.previous == null) return null
        const current = overviewData.income.value
        const previous = overviewData.income.previous
        if (previous === 0 && current === 0) return 0
        if (previous === 0) return 100
        return ((current - previous) / previous) * 100
    }, [overviewData])

    const expenseChange = useMemo(() => {
        if (overviewData?.expenses?.previous == null) return null
        const current = overviewData.expenses.value
        const previous = overviewData.expenses.previous
        if (previous === 0 && current === 0) return 0
        if (previous === 0) return 100
        return ((current - previous) / previous) * 100
    }, [overviewData])

    const pieChartOption = useMemo(() => {
        if (!expensesByCategory?.data.length) return {}

        const isDark = theme === 'dark'
        const categoryCurrency = expensesByCategory.currency || currency
        const data = expensesByCategory.data
            .filter((c) => (c.totalAmount ?? 0) > 0)
            .map((c, i) => ({
                name: localizeDefaultName(c.name),
                value: c.totalAmount ?? 0,
                itemStyle: { color: c.color || CATEGORY_COLORS[i % CATEGORY_COLORS.length] },
            }))

        return {
            tooltip: {
                trigger: 'item',
                confine: true,
                backgroundColor: isDark ? '#1f2937' : '#ffffff',
                borderColor: isDark ? '#374151' : '#e5e7eb',
                textStyle: { color: isDark ? '#f3f4f6' : '#1f2937' },
                formatter: (params: { name: string; value: number; percent: number }) =>
                    `${params.name}<br/>${formatCurrency(params.value, categoryCurrency)} (${params.percent.toFixed(1)}%)`,
            },
            legend: {
                orient: 'horizontal',
                bottom: 0,
                left: 'center',
                textStyle: {
                    color: isDark ? '#9ca3af' : '#6b7280',
                    fontSize: 11,
                },
                icon: 'circle',
                itemWidth: 8,
                itemHeight: 8,
                itemGap: 8,
            },
            grid: {
                containLabel: true,
            },
            series: [
                {
                    type: 'pie',
                    radius: ['35%', '60%'],
                    center: ['50%', '40%'],
                    avoidLabelOverlap: false,
                    itemStyle: {
                        borderRadius: 4,
                        borderColor: isDark ? '#1f2937' : '#ffffff',
                        borderWidth: 2,
                    },
                    label: { show: false },
                    emphasis: {
                        label: { show: false },
                    },
                    data,
                },
            ],
        }
    }, [expensesByCategory, theme, currency, i18n.language])

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
                <div className="flex flex-col gap-2 sm:items-end">
                    <div className="flex items-center gap-2">
                        <Calendar className="size-4 text-muted-foreground hidden sm:block" />
                        <Select value={period} onValueChange={handlePeriodChange}>
                            <SelectTrigger className="w-full sm:w-[200px] h-8">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="last_30_days">{t('dashboard.periods.last_30_days')}</SelectItem>
                                <SelectItem value="this_month">{t('dashboard.periods.this_month')}</SelectItem>
                                <SelectItem value="last_month">{t('dashboard.periods.last_month')}</SelectItem>
                                <SelectItem value="last_3_months">{t('dashboard.periods.last_3_months')}</SelectItem>
                                <SelectItem value="last_6_months">{t('dashboard.periods.last_6_months')}</SelectItem>
                                <SelectItem value="this_year">{t('dashboard.periods.this_year')}</SelectItem>
                                <SelectItem value="custom">{t('dashboard.periods.custom')}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    {period === 'custom' && (
                        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
                            <Input
                                type="date"
                                value={customStartDate}
                                onChange={(e) => setCustomStartDate(e.target.value)}
                                className="h-8 w-full sm:w-[140px]"
                            />
                            <span className="text-muted-foreground text-center hidden sm:block">—</span>
                            <Input
                                type="date"
                                value={customEndDate}
                                onChange={(e) => setCustomEndDate(e.target.value)}
                                className="h-8 w-full sm:w-[140px]"
                            />
                        </div>
                    )}
                </div>
            </div>

            <div className="grid min-w-0 gap-4 md:grid-cols-2 xl:grid-cols-4">
                <Card className="min-w-0">
                    <CardHeader className="flex flex-row items-center justify-between pb-2">
                        <CardTitle className="text-sm font-medium text-muted-foreground">
                            {t('dashboard.totalBalance')}
                        </CardTitle>
                        <Wallet className="size-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="min-w-0">
                        <div className="text-2xl font-bold font-mono break-all">
                            {formatCurrency(totalBalance, currency)}
                        </div>
                        {balanceChange !== null && (
                            <div className={cn(
                                "flex items-center gap-1 text-xs mt-1",
                                balanceChange >= 0 ? "text-green-600" : "text-red-600"
                            )}>
                                {balanceChange >= 0 ? (
                                    <TrendingUp className="size-3" />
                                ) : (
                                    <TrendingDown className="size-3" />
                                )}
                                <span>{Math.abs(balanceChange).toFixed(1)}%</span>
                            </div>
                        )}
                    </CardContent>
                </Card>

                <Card className="min-w-0">
                    <CardHeader className="flex flex-row items-center justify-between pb-2">
                        <CardTitle className="text-sm font-medium text-muted-foreground">
                            {t('dashboard.income')}
                        </CardTitle>
                        <ArrowDownLeft className="size-4 text-green-600" />
                    </CardHeader>
                    <CardContent className="min-w-0">
                        <div className="text-2xl font-bold font-mono text-green-600 break-all">
                            +{formatCurrency(periodIncome, currency)}
                        </div>
                        {incomeChange !== null && (
                            <div className={cn(
                                "flex items-center gap-1 text-xs mt-1",
                                incomeChange >= 0 ? "text-green-600" : "text-red-600"
                            )}>
                                {incomeChange >= 0 ? (
                                    <TrendingUp className="size-3" />
                                ) : (
                                    <TrendingDown className="size-3" />
                                )}
                                <span>{Math.abs(incomeChange).toFixed(1)}%</span>
                            </div>
                        )}
                    </CardContent>
                </Card>

                <Card className="min-w-0">
                    <CardHeader className="flex flex-row items-center justify-between pb-2">
                        <CardTitle className="text-sm font-medium text-muted-foreground">
                            {t('dashboard.expenses')}
                        </CardTitle>
                        <ArrowUpRight className="size-4 text-red-600" />
                    </CardHeader>
                    <CardContent className="min-w-0">
                        <div className="text-2xl font-bold font-mono text-red-600 break-all">
                            -{formatCurrency(periodExpense, currency)}
                        </div>
                        {expenseChange !== null && (
                            <div className={cn(
                                "flex items-center gap-1 text-xs mt-1",
                                expenseChange <= 0 ? "text-green-600" : "text-red-600"
                            )}>
                                {expenseChange >= 0 ? (
                                    <TrendingUp className="size-3" />
                                ) : (
                                    <TrendingDown className="size-3" />
                                )}
                                <span>{Math.abs(expenseChange).toFixed(1)}%</span>
                            </div>
                        )}
                    </CardContent>
                </Card>

                <Card className="min-w-0">
                    <CardHeader className="flex flex-row items-center justify-between pb-2">
                        <CardTitle className="text-sm font-medium text-muted-foreground">
                            {t('dashboard.pending')}
                        </CardTitle>
                        <Clock className="size-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="min-w-0">
                        <Link to="/transactions?status=pending" className="block min-w-0">
                            <div className={cn(
                                'text-2xl font-bold font-mono break-all',
                                (pendingSummary?.balance ?? 0) > 0 && 'text-green-600',
                                (pendingSummary?.balance ?? 0) < 0 && 'text-red-600',
                            )}>
                                {formatCurrency(pendingSummary?.balance ?? 0, pendingSummary?.currency ?? currency)}
                            </div>
                            <p className="text-xs text-muted-foreground mt-1">
                                {(pendingSummary?.transactions_count ?? 0) > 0
                                    ? t('dashboard.pendingCount', { count: pendingSummary?.transactions_count ?? 0 })
                                    : t('dashboard.pendingEmpty')}
                            </p>
                        </Link>
                    </CardContent>
                </Card>
            </div>

            <div className="grid min-w-0 gap-4 lg:grid-cols-3">
                <BalanceDynamicsChart
                    className="min-w-0 overflow-x-auto lg:col-span-2"
                    startDate={periodDates.start_date}
                    endDate={periodDates.end_date}
                />

                <Card className="min-w-0">
                    <CardHeader>
                        <CardTitle>{t('dashboard.accountBalances')}</CardTitle>
                    </CardHeader>
                    <CardContent className="min-w-0">
                        <div className="min-w-0 space-y-3">
                            {accounts && accounts.length > 0 ? (
                                accounts.map((account) => (
                                    <div
                                        key={account.id}
                                        className="flex min-w-0 items-center justify-between gap-2"
                                    >
                                        <div className="flex items-center gap-3 min-w-0 flex-1">
                                            {(() => {
                                                const config = ACCOUNT_TYPE_CONFIG[account.type as AccountType]
                                                const Icon = config?.icon || Wallet
                                                return (
                                                    <div className={`flex size-9 items-center justify-center rounded-lg shrink-0 ${config?.color || 'bg-muted'}`}>
                                                        <Icon className="size-4" />
                                                    </div>
                                                )
                                            })()}
                                            <div className="min-w-0">
                                                <p className="text-sm font-medium truncate">{account.name}</p>
                                                <p className="text-xs text-muted-foreground capitalize">
                                                    {t(`accounts.types.${account.type}`, { defaultValue: account.type })}
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex min-w-0 items-center gap-2">
                                            <div className="min-w-0 text-right">
                                                <p className="text-sm font-mono font-medium break-all">
                                                    {formatCurrency(account.currentBalance ?? 0, account.currency, { showSymbol: false })}
                                                </p>
                                                <p className="text-xs text-muted-foreground">
                                                    {account.currency?.symbol ?? ''}
                                                </p>
                                            </div>
                                            <DropdownMenu>
                                                <DropdownMenuTrigger asChild>
                                                    <Button variant="ghost" size="icon" className="size-8 shrink-0">
                                                        <Plus className="size-4" />
                                                    </Button>
                                                </DropdownMenuTrigger>
                                                <DropdownMenuContent align="end">
                                                    <DropdownMenuItem
                                                        onClick={() => openCreate({ type: 'income', account_id: account.id })}
                                                    >
                                                        <ArrowDownLeft className="size-4 mr-2 text-green-600" />
                                                        {tNav('income')}
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem
                                                        onClick={() => openCreate({ type: 'expense', account_id: account.id })}
                                                    >
                                                        <ArrowUpRight className="size-4 mr-2 text-red-600" />
                                                        {tNav('expense')}
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem
                                                        onClick={() => openCreate({ type: 'transfer', account_id: account.id })}
                                                    >
                                                        <ArrowLeftRight className="size-4 mr-2 text-blue-600" />
                                                        {tNav('transfer')}
                                                    </DropdownMenuItem>
                                                </DropdownMenuContent>
                                            </DropdownMenu>
                                        </div>
                                    </div>
                                ))
                            ) : (
                                <div className="text-center text-muted-foreground py-4">
                                    {t('dashboard.noAccounts')}
                                </div>
                            )}
                        </div>
                    </CardContent>
                </Card>
            </div>

            <div className="grid min-w-0 gap-4 lg:grid-cols-2">
                <Card className="min-w-0">
                    <CardHeader>
                        <CardTitle>{t('dashboard.expensesByCategory')}</CardTitle>
                    </CardHeader>
                    <CardContent className="min-w-0">
                        {expensesByCategory && expensesByCategory.data.some((c) => (c.totalAmount ?? 0) > 0) ? (
                            <div className="min-w-0 overflow-x-auto overscroll-x-contain">
                                <ReactECharts
                                    option={pieChartOption}
                                    style={{ height: '280px', width: '100%' }}
                                    className="min-w-0 max-w-full"
                                    opts={{ renderer: 'svg' }}
                                />
                            </div>
                        ) : (
                            <div className="flex items-center justify-center h-[280px] text-muted-foreground">
                                {t('dashboard.noDataPeriod')}
                            </div>
                        )}
                    </CardContent>
                </Card>

                <Card className="min-w-0">
                    <CardHeader className="flex min-w-0 flex-row items-center justify-between gap-2">
                        <CardTitle className="min-w-0 truncate">{t('dashboard.recentTransactions')}</CardTitle>
                        <Button variant="ghost" size="sm" asChild className="shrink-0">
                            <Link to="/transactions">
                                <span className="hidden sm:inline">{tCommon('actions.viewAll')}</span>
                                <span className="sm:hidden">{tCommon('actions.all')}</span>
                                <ArrowRight className="ml-1 size-4" />
                            </Link>
                        </Button>
                    </CardHeader>
                    <CardContent className="min-w-0">
                        <div className="min-w-0">
                            {recentTransactions?.data && recentTransactions.data.length > 0 ? (
                                <div className="divide-y divide-border/60">
                                    {recentTransactions.data.map((transaction) => (
                                        <TransactionRow
                                            key={transaction.id}
                                            transaction={transaction}
                                            onEdit={openEdit}
                                            showActions={false}
                                        />
                                    ))}
                                </div>
                            ) : (
                                <div className="text-center text-muted-foreground py-4">
                                    {t('dashboard.noTransactions')}
                                </div>
                            )}
                        </div>
                    </CardContent>
                </Card>
            </div>

            <Card className="min-w-0">
                <CardHeader className="flex min-w-0 flex-row items-center justify-between gap-2">
                    <CardTitle className="flex min-w-0 items-center gap-2">
                        <PiggyBank className="size-5 shrink-0" />
                        {t('dashboard.budgets')}
                    </CardTitle>
                    <Button variant="ghost" size="sm" asChild>
                        <Link to="/budgets">
                            {tCommon('actions.viewAll')}
                            <ArrowRight className="ml-1 size-4" />
                        </Link>
                    </Button>
                </CardHeader>
                <CardContent className="min-w-0">
                    {activeBudgets.length > 0 ? (
                        <div className="grid min-w-0 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                            {activeBudgets.map((budget) => {
                                const progress = budget.progress
                                const percent = progress ? Math.min(progress.percent, 100) : 0
                                const isExceeded = progress?.is_exceeded ?? false

                                return (
                                    <Link
                                        key={budget.id}
                                        to={`/budgets?edit=${budget.id}`}
                                        className="block p-4 rounded-lg border hover:bg-muted/50 transition-colors"
                                    >
                                        <div className="flex items-center justify-between mb-2">
                                            <p className="font-medium text-sm truncate">{budget.name}</p>
                                            <span className={`text-xs font-medium ${isExceeded ? 'text-red-600' : 'text-muted-foreground'}`}>
                                                {progress?.percent.toFixed(0)}%
                                            </span>
                                        </div>
                                        <Progress
                                            value={percent}
                                            className={`h-2 mb-2 ${isExceeded ? '[&>div]:bg-red-500' : ''}`}
                                        />
                                        <div className="flex justify-between text-xs text-muted-foreground">
                                            <span>{formatCurrency(progress?.spent ?? 0, budget.currency)}</span>
                                            <span>{formatCurrency(budget.amount, budget.currency)}</span>
                                        </div>
                                        <p className="text-xs text-muted-foreground mt-1">
                                            {budget.isGlobal
                                                ? t('dashboard.allExpenses')
                                                : budget.categories.map(c => localizeDefaultName(c.name)).join(', ') || t('dashboard.noCategories')}
                                        </p>
                                    </Link>
                                )
                            })}
                        </div>
                    ) : (
                        <div className="text-center py-8">
                            <PiggyBank className="size-12 mx-auto text-muted-foreground/50 mb-3" />
                            <p className="text-muted-foreground mb-3">{t('dashboard.noBudgets')}</p>
                            <Button asChild size="sm">
                                <Link to="/budgets?create=1">
                                    <Plus className="size-4 mr-1" />
                                    {t('dashboard.createBudget')}
                                </Link>
                            </Button>
                        </div>
                    )}
                </CardContent>
            </Card>

            <Card className="min-w-0">
                <CardHeader className="flex min-w-0 flex-row items-center justify-between gap-2">
                    <CardTitle className="flex min-w-0 items-center gap-2">
                        <HandCoins className="size-5 shrink-0" />
                        {t('dashboard.debts')}
                    </CardTitle>
                    <Button variant="ghost" size="sm" asChild>
                        <Link to="/debts">
                            {tCommon('actions.viewAll')}
                            <ArrowRight className="ml-1 size-4" />
                        </Link>
                    </Button>
                </CardHeader>
                <CardContent className="min-w-0">
                    {debtSummary && (debtSummary.total_i_owe > 0 || debtSummary.total_owed_to_me > 0) ? (
                        <div className="min-w-0 space-y-4">
                            <div className="grid min-w-0 gap-4 sm:grid-cols-3">
                                <div className="flex items-center gap-3 p-3 rounded-lg bg-red-50 dark:bg-red-950/20">
                                    <div className="p-2 rounded-lg bg-red-100 dark:bg-red-900/30">
                                        <TrendingDown className="size-4 text-red-600" />
                                    </div>
                                    <div>
                                        <p className="text-xs text-muted-foreground">{t('debts.types.i_owe')}</p>
                                        <p className="font-mono font-semibold text-red-600 break-all">
                                            {formatCurrency(debtSummary.total_i_owe, debtSummary.currency)}
                                        </p>
                                    </div>
                                </div>
                                <div className="flex items-center gap-3 p-3 rounded-lg bg-green-50 dark:bg-green-950/20">
                                    <div className="p-2 rounded-lg bg-green-100 dark:bg-green-900/30">
                                        <TrendingUp className="size-4 text-green-600" />
                                    </div>
                                    <div>
                                        <p className="text-xs text-muted-foreground">{t('debts.types.owed_to_me')}</p>
                                        <p className="font-mono font-semibold text-green-600 break-all">
                                            {formatCurrency(debtSummary.total_owed_to_me, debtSummary.currency)}
                                        </p>
                                    </div>
                                </div>
                                <div className={`flex items-center gap-3 p-3 rounded-lg ${debtSummary.net_debt >= 0 ? 'bg-green-50 dark:bg-green-950/20' : 'bg-red-50 dark:bg-red-950/20'}`}>
                                    <div className={`p-2 rounded-lg ${debtSummary.net_debt >= 0 ? 'bg-green-100 dark:bg-green-900/30' : 'bg-red-100 dark:bg-red-900/30'}`}>
                                        {debtSummary.net_debt >= 0 ? (
                                            <HandCoins className="size-4 text-green-600" />
                                        ) : (
                                            <Banknote className="size-4 text-red-600" />
                                        )}
                                    </div>
                                    <div>
                                        <p className="text-xs text-muted-foreground">{t('debts.netPosition')}</p>
                                        <p className={`font-mono font-semibold break-all ${debtSummary.net_debt >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                                            {formatCurrency(Math.abs(debtSummary.net_debt), debtSummary.currency)}
                                        </p>
                                    </div>
                                </div>
                            </div>

                            {activeDebts.length > 0 && (
                                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                                    {activeDebts.map((debt) => (
                                        <Link
                                            key={debt.id}
                                            to={`/debts?edit=${debt.id}`}
                                            className="block p-3 rounded-lg border hover:bg-muted/50 transition-colors"
                                        >
                                            <div className="flex items-center gap-2 mb-2">
                                                {debt.debtType === 'i_owe' ? (
                                                    <Banknote className="size-4 text-red-600" />
                                                ) : (
                                                    <HandCoins className="size-4 text-green-600" />
                                                )}
                                                <p className="font-medium text-sm truncate">{debt.name}</p>
                                            </div>
                                            <Progress
                                                value={debt.paymentProgress}
                                                className="h-1.5 mb-2"
                                            />
                                            <div className="flex justify-between text-xs">
                                                <span className="text-muted-foreground">
                                                    {t('dashboard.percentPaid', { percent: debt.paymentProgress.toFixed(0) })}
                                                </span>
                                                <span className={debt.debtType === 'i_owe' ? 'text-red-600' : 'text-green-600'}>
                                                    {formatCurrency(debt.remainingDebt, debt.currency)}
                                                </span>
                                            </div>
                                            {debt.counterparty && (
                                                <p className="text-xs text-muted-foreground mt-1 truncate">
                                                    {debt.debtType === 'i_owe'
                                                        ? t('dashboard.toCounterparty', { name: debt.counterparty })
                                                        : t('dashboard.fromCounterparty', { name: debt.counterparty })}
                                                </p>
                                            )}
                                        </Link>
                                    ))}
                                </div>
                            )}
                        </div>
                    ) : (
                        <div className="text-center py-8">
                            <HandCoins className="size-12 mx-auto text-muted-foreground/50 mb-3" />
                            <p className="text-muted-foreground mb-3">{t('dashboard.noDebts')}</p>
                            <Button asChild size="sm">
                                <Link to="/debts?create=1">
                                    <Plus className="size-4 mr-1" />
                                    {t('dashboard.addDebt')}
                                </Link>
                            </Button>
                        </div>
                    )}
                </CardContent>
            </Card>
        </div>
    )
}
