import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { intlLocale } from '@/lib/i18n'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ChevronRight } from 'lucide-react'
import { useTransactionReportTop } from '@/hooks'
import { formatCurrency } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import type { ReportFilters } from '../types'
import type { ReportTransactionType } from '@/api/reports'

interface TopTransactionsProps {
    filters: ReportFilters
    type: ReportTransactionType
    limit?: number
}

function formatShortDate(dateStr: string) {
    if (!dateStr) return ''
    const parts = dateStr.split('-')
    if (parts.length !== 3) return dateStr
    const [year, month, day] = parts.map(Number)
    return new Date(year, month - 1, day).toLocaleDateString(intlLocale(), { month: 'short', day: 'numeric' })
}

export function TopTransactions({ filters, type, limit = 10 }: TopTransactionsProps) {
    const { t } = useTranslation('pages')
    const navigate = useNavigate()
    const { data, isLoading } = useTransactionReportTop(filters, type, limit)

    const transactions = data?.items || []
    const copy = type === 'income'
        ? {
            title: t('reports.topIncome.title'),
            subtitle: t('reports.topIncome.subtitle'),
            topTotal: t('reports.topIncome.topTotal', { count: transactions.length }),
            noData: t('reports.topIncome.noData'),
        }
        : {
            title: t('reports.topExpenses.title'),
            subtitle: t('reports.topExpenses.subtitle'),
            topTotal: t('reports.topExpenses.topTotal', { count: transactions.length }),
            noData: t('reports.topExpenses.noData'),
        }
    const currency = data?.currency
    const amountClass = type === 'income' ? 'text-green-600' : 'text-red-600'
    const amountPrefix = type === 'income' ? '+' : '-'

    const totalTop = useMemo(() => {
        return transactions.reduce((sum, item) => sum + item.amount, 0)
    }, [transactions])

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                    <div>
                        <CardTitle className="text-lg">{copy.title}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {copy.subtitle}
                        </p>
                    </div>
                    {!isLoading && transactions.length > 0 && (
                        <div className="text-right">
                            <p className="text-sm text-muted-foreground">{copy.topTotal}</p>
                            <p className={`text-lg font-semibold ${amountClass}`}>{formatCurrency(totalTop, currency)}</p>
                        </div>
                    )}
                </div>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <div className="space-y-2">
                        {Array.from({ length: 5 }).map((_, index) => (
                            <Skeleton key={index} className="h-14" />
                        ))}
                    </div>
                ) : transactions.length === 0 ? (
                    <div className="h-[200px] flex items-center justify-center text-muted-foreground">
                        {copy.noData}
                    </div>
                ) : (
                    <div className="space-y-1">
                        {transactions.map((transaction, index) => (
                            <div
                                key={transaction.id}
                                className="group flex items-center gap-3 p-2 rounded-lg hover:bg-muted/50 cursor-pointer transition-colors"
                                onClick={() => navigate(`/transactions?id=${transaction.id}`)}
                            >
                                <div className="flex items-center justify-center size-6 rounded-full bg-muted text-xs font-medium text-muted-foreground">
                                    {index + 1}
                                </div>
                                <div
                                    className="flex items-center justify-center size-9 rounded-lg flex-shrink-0"
                                    style={{ backgroundColor: transaction.category.color }}
                                >
                                    <span className="text-lg">{transaction.category.icon}</span>
                                </div>
                                <div className="flex-1 min-w-0">
                                    <p className="font-medium text-sm truncate">
                                        {transaction.description || localizeDefaultName(transaction.category.name)}
                                    </p>
                                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                        <span>{formatShortDate(transaction.date)}</span>
                                        <span>•</span>
                                        <span>{localizeDefaultName(transaction.category.name)}</span>
                                        <span>•</span>
                                        <span>{transaction.account.name}</span>
                                    </div>
                                </div>
                                <div className="text-right flex-shrink-0">
                                    <p className={`font-semibold ${amountClass}`}>
                                        {amountPrefix}{formatCurrency(transaction.amount, currency)}
                                    </p>
                                </div>
                                <ChevronRight className="size-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0" />
                            </div>
                        ))}
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
