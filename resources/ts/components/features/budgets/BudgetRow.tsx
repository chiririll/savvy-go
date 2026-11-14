import { PiggyBank } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { categoryIcons, FeedRow, progressAmounts, RowActions, type FeedGroupKey } from '@/components/shared'
import i18n from '@/lib/i18n/i18n'
import { localizeDefaultName } from '@/lib/format/localized-name'
import { Budget, BudgetPeriod } from '@/types'
import { budgetPeriodLabelLoose } from '@/lib/i18n/labels'

const PERIOD_ORDER: BudgetPeriod[] = ['weekly', 'monthly', 'yearly', 'one_time']

/** Sections of the budget list: one per period, shortest first. */
export function budgetGroup(budget: Budget): FeedGroupKey {
    return {
        key: budget.period,
        title: budgetPeriodLabelLoose(i18n.t, budget.period),
    }
}

/** Period order, then active before inactive. */
export function compareBudgets(a: Budget, b: Budget): number {
    return PERIOD_ORDER.indexOf(a.period) - PERIOD_ORDER.indexOf(b.period)
        || Number(b.isActive) - Number(a.isActive)
}

interface BudgetRowProps {
    budget: Budget
    onEdit?: (budget: Budget) => void
    onDelete?: (id: number) => void
    isReadOnly?: boolean
}

export function BudgetRow({ budget, onEdit, onDelete, isReadOnly }: BudgetRowProps) {
    const { t } = useTranslation('pages')
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const progress = budget.progress
    const color = budget.categories.find((category) => category.color)?.color
    const scope = budget.isGlobal
        ? t('budgets.allExpenses')
        : budget.categories.map((category) => localizeDefaultName(category.name)).join(', ')

    return (
        <FeedRow
            icon={categoryIcons(budget.categories) ?? <PiggyBank className="size-4" />}
            iconClassName={color ? undefined : 'bg-muted text-muted-foreground'}
            iconColor={color}
            title={budget.name}
            subtitle={scope || undefined}
            {...(progress && progressAmounts({
                remaining: budget.amount - progress.spent,
                total: budget.amount,
                currency: budget.currency,
            }))}
            muted={!budget.isActive}
            progress={progress ? { value: progress.percent, exceeded: progress.isExceeded, muted: !budget.isActive } : undefined}
            onOpen={canEdit ? () => onEdit(budget) : undefined}
            hasActions={canEdit || canDelete}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(budget) : undefined}
                    onDelete={canDelete ? () => onDelete(budget.id) : undefined}
                    deleteTitle={t('budgets.deleteTitle')}
                    deleteDescription={t('budgets.deleteDescription')}
                />
            )}
        />
    )
}
