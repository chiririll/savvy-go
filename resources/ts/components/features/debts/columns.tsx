import { ColumnDef } from '@tanstack/react-table'
import { HandCoins, Banknote, RotateCcw } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { RowActions } from '@/components/shared'
import { Debt } from '@/types'
import { formatCurrency } from '@/lib/utils'
import i18n, { intlLocale } from '@/lib/i18n'

const DEBT_TYPE_CONFIG = {
    i_owe: {
        icon: Banknote,
        color: 'bg-red-100',
        textColor: 'text-red-600',
        label: 'I Owe'
    },
    owed_to_me: {
        icon: HandCoins,
        color: 'bg-green-100',
        textColor: 'text-green-600',
        label: 'Owed to Me'
    },
}

interface ColumnActions {
    onDelete: (id: number) => void
    onPayment: (debt: Debt) => void
    onCollect: (debt: Debt) => void
    onReopen: (id: number) => void
    onEdit: (debt: Debt) => void
    isReadOnly?: boolean
}

export const createDebtColumns = (
    actions: ColumnActions
): ColumnDef<Debt>[] => [
    {
        accessorKey: 'name',
        header: () => i18n.t('pages:debts.columns.debt'),
        cell: ({ row }) => {
            const config = DEBT_TYPE_CONFIG[row.original.debtType]
            const Icon = config.icon

            return (
                <div className="flex items-center gap-3">
                    <div className={`p-2 rounded-lg ${config.color}`}>
                        <Icon className={`size-5 ${config.textColor}`} />
                    </div>
                    <div>
                        <p className="font-medium">{row.original.name}</p>
                        {row.original.counterparty && (
                            <p className="text-xs text-muted-foreground">
                                {row.original.counterparty}
                            </p>
                        )}
                    </div>
                </div>
            )
        },
    },
    {
        accessorKey: 'debtType',
        header: () => i18n.t('pages:debts.columns.type'),
        cell: ({ row }) => {
            const config = DEBT_TYPE_CONFIG[row.original.debtType]
            return (
                <Badge variant="secondary" className={`${config.color} ${config.textColor}`}>
                    {i18n.t(`pages:debts.types.${row.original.debtType}`)}
                </Badge>
            )
        },
    },
    {
        accessorKey: 'progress',
        header: () => i18n.t('pages:debts.columns.progress'),
        cell: ({ row }) => {
            const debt = row.original
            const progress = debt.paymentProgress

            return (
                <div className="w-32">
                    <Progress value={progress} className="h-2" />
                    <div className="flex justify-between text-xs text-muted-foreground mt-1">
                        <span>{formatCurrency(debt.currentBalance, debt.currency)}</span>
                        <span>{progress.toFixed(0)}%</span>
                    </div>
                </div>
            )
        },
    },
    {
        accessorKey: 'targetAmount',
        header: () => i18n.t('pages:debts.columns.total'),
        cell: ({ row }) => (
            <div className="font-mono text-right">
                {formatCurrency(row.original.targetAmount, row.original.currency)}
            </div>
        ),
    },
    {
        accessorKey: 'currentBalance',
        header: () => i18n.t('pages:debts.columns.remaining'),
        cell: ({ row }) => (
            <div className={`font-mono text-right ${row.original.isPaidOff ? 'text-green-600' : 'text-orange-600'}`}>
                {row.original.isPaidOff
                    ? i18n.t('pages:debts.paidOff')
                    : formatCurrency(row.original.currentBalance, row.original.currency)
                }
            </div>
        ),
    },
    {
        accessorKey: 'dueDate',
        header: () => i18n.t('pages:debts.columns.dueDate'),
        cell: ({ row }) => {
            if (!row.original.dueDate) return <span className="text-muted-foreground">-</span>

            const dueDate = new Date(row.original.dueDate)
            const isOverdue = !row.original.isPaidOff && dueDate < new Date()

            return (
                <span className={isOverdue ? 'text-red-600 font-medium' : ''}>
                    {dueDate.toLocaleDateString(intlLocale())}
                </span>
            )
        },
    },
    {
        accessorKey: 'isActive',
        header: () => i18n.t('pages:debts.columns.status'),
        cell: ({ row }) => (
            <Badge variant={row.original.isPaidOff ? 'default' : row.original.isActive ? 'secondary' : 'outline'}>
                {row.original.isPaidOff
                    ? i18n.t('pages:debts.status.completed')
                    : row.original.isActive
                        ? i18n.t('pages:debts.status.active')
                        : i18n.t('pages:debts.status.inactive')}
            </Badge>
        ),
    },
    {
        id: 'actions',
        header: '',
        cell: ({ row }) => {
            const debt = row.original

            return (
                <RowActions
                    onEdit={() => actions.onEdit(debt)}
                    onDelete={() => actions.onDelete(debt.id)}
                    deleteTitle={i18n.t('pages:debts.deleteTitle')}
                    deleteDescription={i18n.t('pages:debts.deleteDescription')}
                    isReadOnly={actions.isReadOnly}
                    leading={(
                        <>
                            {!actions.isReadOnly && !debt.isPaidOff && (
                                <>
                                    {debt.debtType === 'i_owe' ? (
                                        <DropdownMenuItem onClick={() => actions.onPayment(debt)}>
                                            <Banknote className="mr-2 size-4" />
                                            {i18n.t('pages:debts.makePayment')}
                                        </DropdownMenuItem>
                                    ) : (
                                        <DropdownMenuItem onClick={() => actions.onCollect(debt)}>
                                            <HandCoins className="mr-2 size-4" />
                                            {i18n.t('pages:debts.collectPayment')}
                                        </DropdownMenuItem>
                                    )}
                                    <DropdownMenuSeparator />
                                </>
                            )}
                            {!actions.isReadOnly && debt.isPaidOff && (
                                <>
                                    <DropdownMenuItem onClick={() => actions.onReopen(debt.id)}>
                                        <RotateCcw className="mr-2 size-4" />
                                        {i18n.t('pages:debts.reopen')}
                                    </DropdownMenuItem>
                                    <DropdownMenuSeparator />
                                </>
                            )}
                        </>
                    )}
                />
            )
        },
    },
]
