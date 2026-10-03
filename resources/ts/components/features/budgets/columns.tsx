import { ColumnDef } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { RowActions } from '@/components/shared'
import { Budget } from '@/types'
import { formatCurrency } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import i18n from '@/lib/i18n'

interface ColumnOptions {
    onDelete: (id: number) => void
    onEdit: (budget: Budget) => void
    isReadOnly?: boolean
}

export const createBudgetColumns = ({
    onDelete,
    onEdit,
    isReadOnly,
}: ColumnOptions): ColumnDef<Budget>[] => [
    {
        accessorKey: 'name',
        header: () => i18n.t('pages:budgets.columns.budget'),
        cell: ({ row }) => (
            <div>
                <p className="font-medium">{row.original.name}</p>
                <p className="text-xs text-muted-foreground">
                    {row.original.isGlobal
                        ? i18n.t('pages:budgets.allExpenses')
                        : row.original.categories.map(c => localizeDefaultName(c.name)).join(', ') || i18n.t('pages:budgets.noCategories')}
                </p>
            </div>
        ),
    },
    {
        accessorKey: 'amount',
        header: () => i18n.t('pages:budgets.columns.limit'),
        cell: ({ row }) => (
            <span className="font-mono font-medium">
                {formatCurrency(row.original.amount, row.original.currency)}
            </span>
        ),
    },
    {
        accessorKey: 'period',
        header: () => i18n.t('pages:budgets.columns.period'),
        cell: ({ row }) => (
            <Badge variant="outline">
                {i18n.t(`pages:budgets.periods.${row.original.period}`, { defaultValue: row.original.period })}
            </Badge>
        ),
    },
    {
        accessorKey: 'progress',
        header: () => i18n.t('pages:budgets.columns.progress'),
        cell: ({ row }) => {
            const progress = row.original.progress
            if (!progress) return null

            const isExceeded = progress.isExceeded
            const percent = Math.min(progress.percent, 100)
            return (
                <div className="w-44 space-y-1">
                    <div className="flex justify-between text-xs">
                        <span>{i18n.t('pages:budgets.spent', { amount: formatCurrency(progress.spent, row.original.currency) })}</span>
                        <span className={isExceeded ? 'text-red-600 font-medium' : ''}>
                            {progress.percent.toFixed(0)}%
                        </span>
                    </div>
                    <Progress
                        value={percent}
                        className={`h-2 ${isExceeded ? '[&>div]:bg-red-500' : ''}`}
                    />
                    <p className="text-xs text-muted-foreground">
                        {isExceeded
                            ? i18n.t('pages:budgets.exceededBy', { amount: formatCurrency(progress.spent - row.original.amount, row.original.currency) })
                            : i18n.t('pages:budgets.remaining', { amount: formatCurrency(progress.remaining, row.original.currency) })}
                    </p>
                </div>
            )
        },
    },
    {
        accessorKey: 'isActive',
        header: () => i18n.t('pages:budgets.columns.status'),
        cell: ({ row }) => (
            <Badge variant={row.original.isActive ? 'default' : 'secondary'}>
                {row.original.isActive
                    ? i18n.t('pages:budgets.status.active')
                    : i18n.t('pages:budgets.status.inactive')}
            </Badge>
        ),
    },
    {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
            <RowActions
                onEdit={() => onEdit(row.original)}
                onDelete={() => onDelete(row.original.id)}
                deleteTitle={i18n.t('pages:budgets.deleteTitle')}
                deleteDescription={i18n.t('pages:budgets.deleteDescription')}
                isReadOnly={isReadOnly}
            />
        ),
    },
]
