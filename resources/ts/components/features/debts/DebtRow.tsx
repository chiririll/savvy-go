import { Banknote, HandCoins, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { FeedProgress, FeedRow, progressAmounts, RowActions } from '@/components/shared'
import { useRelativeDay } from '@/hooks'
import { isDateOverdue } from '@/lib/dates'
import { cn } from '@/lib/utils'
import { Debt } from '@/types'

const DEBT_TYPE_CONFIG = {
    i_owe: { icon: Banknote, tone: 'bg-red-100 text-red-600 dark:bg-red-950/40' },
    owed_to_me: { icon: HandCoins, tone: 'bg-green-100 text-green-600 dark:bg-green-950/40' },
} as const

interface DebtRowProps {
    debt: Debt
    onEdit?: (debt: Debt) => void
    onDelete?: (id: number) => void
    onPayment?: (debt: Debt) => void
    onCollect?: (debt: Debt) => void
    onReopen?: (id: number) => void
    isReadOnly?: boolean
}

export function DebtRow({ debt, onEdit, onDelete, onPayment, onCollect, onReopen, isReadOnly }: DebtRowProps) {
    const { t } = useTranslation('pages')
    const relativeDay = useRelativeDay()
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const { icon: Icon, tone } = DEBT_TYPE_CONFIG[debt.debtType]
    const settle = debt.debtType === 'i_owe' ? onPayment : onCollect
    const canSettle = !isReadOnly && !debt.isPaidOff && !!settle
    const canReopen = !isReadOnly && debt.isPaidOff && !!onReopen
    const overdue = !debt.isPaidOff && isDateOverdue(debt.dueDate?.slice(0, 10))

    return (
        <FeedRow
            icon={<Icon className="size-4" />}
            iconClassName={tone}
            title={debt.name}
            subtitle={(debt.dueDate || debt.counterparty) && (
                <>
                    {debt.dueDate && (
                        <span className={cn(overdue && 'font-medium text-red-600')}>
                            {t('debts.due', { date: relativeDay(debt.dueDate) })}
                        </span>
                    )}
                    {debt.dueDate && debt.counterparty && ' · '}
                    {debt.counterparty}
                </>
            )}
            {...progressAmounts({
                remaining: debt.currentBalance,
                total: debt.targetAmount,
                currency: debt.currency,
                done: debt.isPaidOff ? t('debts.paidOff') : undefined,
            })}
            below={!debt.isPaidOff && debt.targetAmount > 0 && <FeedProgress value={debt.paymentProgress} />}
            onOpen={canEdit ? () => onEdit(debt) : undefined}
            hasActions={canEdit || canDelete || canSettle || canReopen}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(debt) : undefined}
                    onDelete={canDelete ? () => onDelete(debt.id) : undefined}
                    deleteTitle={t('debts.deleteTitle')}
                    deleteDescription={t('debts.deleteDescription')}
                    leading={(canSettle || canReopen) && (
                        <>
                            {canSettle && (
                                <DropdownMenuItem onClick={() => settle(debt)}>
                                    <Icon className="mr-2 size-4" />
                                    {t(debt.debtType === 'i_owe' ? 'debts.makePayment' : 'debts.collectPayment')}
                                </DropdownMenuItem>
                            )}
                            {canReopen && (
                                <DropdownMenuItem onClick={() => onReopen(debt.id)}>
                                    <RotateCcw className="mr-2 size-4" />
                                    {t('debts.reopen')}
                                </DropdownMenuItem>
                            )}
                            <DropdownMenuSeparator />
                        </>
                    )}
                />
            )}
        />
    )
}
