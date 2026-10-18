import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { TransactionRow, useCreateTransactionDialog } from '@/components/features/transactions'
import type { Transaction } from '@/types'

interface RecentTransactionsCardProps {
    transactions?: Transaction[]
    onEdit: ReturnType<typeof useCreateTransactionDialog>['openEdit']
}

export function RecentTransactionsCard({ transactions, onEdit }: RecentTransactionsCardProps) {
    const { t } = useTranslation('pages')
    const { t: tCommon } = useTranslation('common')

    return (
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
                    {transactions && transactions.length > 0 ? (
                        <div className="divide-y divide-border/60">
                            {transactions.map((transaction) => (
                                <TransactionRow
                                    key={transaction.id}
                                    transaction={transaction}
                                    onEdit={onEdit}
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
    )
}
