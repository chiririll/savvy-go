import { testId } from '@/lib/test-id'
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

interface DebtHandlers {
    edit?: () => void
    remove?: () => void
    settle?: () => void
    reopen?: () => void
}

/** Each action the row offers, or undefined when the debt, the role or the caller rules it out. */
function debtHandlers(debt: Debt, { onEdit, onDelete, onPayment, onCollect, onReopen, isReadOnly }: DebtRowProps) {
    const writable = !isReadOnly
    const settle = debt.debtType === 'i_owe' ? onPayment : onCollect
    return {
        edit: writable && onEdit ? () => onEdit(debt) : undefined,
        remove: writable && onDelete ? () => onDelete(debt.id) : undefined,
        settle: writable && !debt.isPaidOff && settle ? () => settle(debt) : undefined,
        reopen: writable && debt.isPaidOff && onReopen ? () => onReopen(debt.id) : undefined,
    } satisfies DebtHandlers
}

function DebtSubtitle({ debt }: { debt: Debt }) {
    const { t } = useTranslation('pages')
    const relativeDay = useRelativeDay()
    const overdue = !debt.isPaidOff && isDateOverdue(debt.dueDate?.slice(0, 10))
    if (!debt.dueDate && !debt.counterparty) return null
    return (
        <>
            {debt.dueDate && (
                <span className={cn(overdue && 'font-medium text-red-600')}>
                    {t('debts.due', { date: relativeDay(debt.dueDate) })}
                </span>
            )}
            {debt.dueDate && debt.counterparty && ' · '}
            {debt.counterparty}
        </>
    )
}

interface DebtMenuProps {
    debt: Debt
    handlers: DebtHandlers
    open: boolean
    onOpenChange: (open: boolean) => void
    showTrigger: boolean
}

function DebtMenu({ debt, handlers, open, onOpenChange, showTrigger }: DebtMenuProps) {
    const { t } = useTranslation('pages')
    const { icon: Icon } = DEBT_TYPE_CONFIG[debt.debtType]
    return (
        <RowActions
            open={open}
            onOpenChange={onOpenChange}
            showTrigger={showTrigger}
            onEdit={handlers.edit}
            onDelete={handlers.remove}
            deleteTitle={t('debts.deleteTitle')}
            deleteDescription={t('debts.deleteDescription')}
            leading={
                (handlers.settle || handlers.reopen) && (
                    <>
                        {handlers.settle && (
                            <DropdownMenuItem onClick={handlers.settle} {...testId('settle')}>
                                <Icon className="mr-2 size-4" />
                                {debt.debtType === 'i_owe' ? t('debts.makePayment') : t('debts.collectPayment')}
                            </DropdownMenuItem>
                        )}
                        {handlers.reopen && (
                            <DropdownMenuItem onClick={handlers.reopen}>
                                <RotateCcw className="mr-2 size-4" />
                                {t('debts.reopen')}
                            </DropdownMenuItem>
                        )}
                        <DropdownMenuSeparator />
                    </>
                )
            }
        />
    )
}

export function DebtRow(props: DebtRowProps) {
    const { debt } = props
    const { t } = useTranslation('pages')
    const { icon: Icon, tone } = DEBT_TYPE_CONFIG[debt.debtType]
    const handlers = debtHandlers(debt, props)

    return (
        <FeedRow
            icon={<Icon className="size-4" />}
            iconClassName={tone}
            title={debt.name}
            subtitle={<DebtSubtitle debt={debt} />}
            {...progressAmounts({
                remaining: debt.currentBalance,
                total: debt.targetAmount,
                currency: debt.currency,
                done: debt.isPaidOff ? t('debts.paidOff') : undefined,
            })}
            below={!debt.isPaidOff && debt.targetAmount > 0 && <FeedProgress value={debt.paymentProgress} />}
            onOpen={handlers.edit}
            hasActions={Object.values(handlers).some(Boolean)}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <DebtMenu
                    debt={debt}
                    handlers={handlers}
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                />
            )}
        />
    )
}
