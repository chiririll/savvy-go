import { PiggyBank } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { FeedProgress, FeedRow, progressAmounts, RowActions, type FeedGroupKey } from '@/components/shared'
import { categoryIconStyle } from '@/lib/category-color'
import i18n from '@/lib/i18n'
import { localizeDefaultName } from '@/lib/localized-name'
import { Budget, BudgetPeriod } from '@/types'

const PERIOD_ORDER: BudgetPeriod[] = ['weekly', 'monthly', 'yearly', 'one_time']

/** Sections of the budget list: one per period, shortest first. */
export function budgetGroup(budget: Budget): FeedGroupKey {
    return {
        key: budget.period,
        title: i18n.t(`pages:budgets.periods.${budget.period}`, { defaultValue: budget.period }),
    }
}

/** Period order, then active before inactive. */
export function compareBudgets(a: Budget, b: Budget): number {
    return PERIOD_ORDER.indexOf(a.period) - PERIOD_ORDER.indexOf(b.period)
        || Number(b.isActive) - Number(a.isActive)
}

/** Up to two of the budget's category emojis, else a piggy bank. */
function BudgetIcon({ budget }: { budget: Budget }) {
    const icons = budget.categories.map((category) => category.icon).filter(Boolean).slice(0, 2)
    if (icons.length === 0) {
        return <PiggyBank className="size-4" />
    }
    if (icons.length === 1) {
        return <span aria-hidden>{icons[0]}</span>
    }
    return (
        <span aria-hidden className="relative block size-full text-sm">
            <span className="absolute left-1 top-1">{icons[0]}</span>
            <span className="absolute bottom-1 right-1">{icons[1]}</span>
        </span>
    )
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
            icon={<BudgetIcon budget={budget} />}
            iconClassName={color ? undefined : 'bg-muted text-muted-foreground'}
            iconStyle={categoryIconStyle(color)}
            title={budget.name}
            subtitle={scope || undefined}
            {...(progress && progressAmounts({
                remaining: budget.amount - progress.spent,
                total: budget.amount,
                currency: budget.currency,
            }))}
            muted={!budget.isActive}
            below={progress && (
                <FeedProgress value={progress.percent} exceeded={progress.isExceeded} muted={!budget.isActive} />
            )}
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
