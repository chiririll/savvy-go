import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight, PiggyBank, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { formatCurrency } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import type { Budget } from '@/types'

function BudgetTile({ budget }: { budget: Budget }) {
    const { t } = useTranslation('pages')
    const progress = budget.progress
    const percent = progress ? Math.min(progress.percent, 100) : 0
    const isExceeded = progress?.isExceeded ?? false

    return (
        <Link
            to={`/budgets?edit=${budget.id}`}
            className="block p-4 rounded-lg border hover:bg-muted/50 transition-colors"
        >
            <div className="flex items-center justify-between mb-2">
                <p className="font-medium text-sm truncate">{budget.name}</p>
                <span className={`text-xs font-medium ${isExceeded ? 'text-destructive' : 'text-muted-foreground'}`}>
                    {progress?.percent.toFixed(0)}%
                </span>
            </div>
            <Progress
                aria-label={budget.name}
                value={percent}
                className={`h-2 mb-2 ${isExceeded ? '[&>div]:bg-destructive' : ''}`}
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
}

export function BudgetsCard({ budgets }: { budgets: Budget[] }) {
    const { t } = useTranslation('pages')
    const { t: tCommon } = useTranslation('common')

    return (
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
                {budgets.length > 0 ? (
                    <div className="grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                        {budgets.map((budget) => (
                            <BudgetTile key={budget.id} budget={budget} />
                        ))}
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
    )
}
