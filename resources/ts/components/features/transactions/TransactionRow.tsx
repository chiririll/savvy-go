import { useState } from 'react'
import {
    ArrowDownLeft,
    ArrowLeftRight,
    ArrowUpRight,
    Banknote,
    Check,
    Copy,
    HandCoins,
    SkipForward,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { FeedRow, FeedStatusBadge, RowActions } from '@/components/shared'
import { formatTransactionGroupHeading } from '@/lib/dates'
import { intlLocale } from '@/lib/i18n'
import { cn, formatCurrency } from '@/lib/utils'
import { displayTransactionDescription, transactionAmountAppearance, transactionSubtitle } from '@/lib/transaction-description'
import { Transaction } from '@/types'
import { SkipTransactionAlert } from './SkipTransactionAlert'
import { TransactionItemsRow } from './TransactionItemsRow'

const TYPE_ICONS = {
    income: ArrowDownLeft,
    expense: ArrowUpRight,
    transfer: ArrowLeftRight,
    debt_payment: Banknote,
    debt_collection: HandCoins,
    debt_lend: HandCoins,
    debt_borrow: Banknote,
} as const

const TYPE_ICON_TONES = {
    income: 'bg-green-100 text-green-600 dark:bg-green-950/40',
    expense: 'bg-red-100 text-red-600 dark:bg-red-950/40',
    transfer: 'bg-blue-100 text-blue-600 dark:bg-blue-950/40',
    debt_payment: 'bg-orange-100 text-orange-600 dark:bg-orange-950/40',
    debt_collection: 'bg-purple-100 text-purple-600 dark:bg-purple-950/40',
    debt_lend: 'bg-red-100 text-red-600 dark:bg-red-950/40',
    debt_borrow: 'bg-green-100 text-green-600 dark:bg-green-950/40',
} as const

interface TransactionRowProps {
    transaction: Transaction
    onDelete?: (id: number) => void
    onDuplicate?: (id: number) => void
    onConfirm?: (transaction: Transaction) => void
    onSkip?: (id: number) => void
    onEdit?: (transaction: Transaction) => void
    isReadOnly?: boolean
    showActions?: boolean
    showDate?: boolean
}

export function TransactionRow({
    transaction,
    onDelete,
    onDuplicate,
    onConfirm,
    onSkip,
    onEdit,
    isReadOnly,
    showActions = true,
    showDate = false,
}: TransactionRowProps) {
    const { t, i18n } = useTranslation(['common', 'pages'])
    const dateLabel = showDate
        ? formatTransactionGroupHeading(transaction.date, intlLocale(i18n.language), {
            today: t('pages:transactions.today'),
            yesterday: t('pages:transactions.yesterday'),
            noDate: t('pages:transactions.noDate'),
        })
        : null
    const [expanded, setExpanded] = useState(false)
    const itemsCount = transaction.items.length
    const canExpand = itemsCount > 1
    const { sign, className } = transactionAmountAppearance(transaction.type, transaction.status)
    const isTransfer = transaction.type === 'transfer'
    const TypeIcon = TYPE_ICONS[transaction.type]
    const { edit, duplicate, delete: canRemove, confirm, skip } = transaction.actions
    const canEdit = edit && !!onEdit
    const canConfirm = !isReadOnly && confirm && !!onConfirm
    const canSkip = !isReadOnly && skip && !!onSkip
    const canDuplicate = !isReadOnly && duplicate && !!onDuplicate
    const canDelete = !isReadOnly && canRemove && !!onDelete
    const hasActions = showActions && (canEdit || canConfirm || canSkip || canDuplicate || canDelete)

    return (
        <FeedRow
            icon={transaction.category?.icon ? (
                <span aria-hidden>{transaction.category.icon}</span>
            ) : (
                <TypeIcon className="size-4" />
            )}
            iconClassName={!transaction.category?.color ? TYPE_ICON_TONES[transaction.type] : undefined}
            iconStyle={transaction.category?.color
                ? { backgroundColor: `${transaction.category.color}20` }
                : undefined}
            title={displayTransactionDescription(transaction)}
            badge={(
                <>
                    {transaction.status === 'skipped' && (
                        <FeedStatusBadge variant="secondary">
                            {t('pages:transactions.status.skipped')}
                        </FeedStatusBadge>
                    )}
                    {transaction.status === 'pending' && (
                        <FeedStatusBadge variant="outline">
                            {t('pages:transactions.status.pending')}
                        </FeedStatusBadge>
                    )}
                </>
            )}
            subtitle={(
                <>
                    {dateLabel && <span className="mr-1.5 font-medium">{dateLabel} ·</span>}
                    {transactionSubtitle(transaction)}
                    {canExpand && (
                        <button
                            type="button"
                            className="ml-1.5 text-primary hover:underline"
                            aria-expanded={expanded}
                            onClick={(event) => {
                                event.stopPropagation()
                                setExpanded((value) => !value)
                            }}
                        >
                            ({t('pages:transactions.items.count', { count: itemsCount })})
                        </button>
                    )}
                </>
            )}
            amount={`${sign}${formatCurrency(transaction.amount, transaction.account.currency)}`}
            amountClassName={cn(className, transaction.status === 'pending' && 'opacity-60')}
            extraAmount={isTransfer && transaction.toAmount != null && transaction.toAccount
                ? `${transaction.status === 'skipped' ? '' : '+'}${formatCurrency(transaction.toAmount, transaction.toAccount.currency)}`
                : undefined}
            onOpen={canEdit ? () => onEdit(transaction) : undefined}
            hasActions={hasActions}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(transaction) : undefined}
                    onDelete={canDelete ? () => onDelete(transaction.id) : undefined}
                    deleteTitle={t('pages:transactions.deleteTitle')}
                    deleteDescription={t('pages:transactions.deleteDescription')}
                >
                    {canConfirm && (
                        <DropdownMenuItem onClick={() => onConfirm(transaction)}>
                            <Check className="mr-2 size-4" />
                            {t('actions.confirm')}
                        </DropdownMenuItem>
                    )}
                    {canSkip && (
                        <SkipTransactionAlert
                            onConfirm={() => onSkip(transaction.id)}
                            trigger={
                                <DropdownMenuItem onSelect={(event) => event.preventDefault()}>
                                    <SkipForward className="mr-2 size-4" />
                                    {t('actions.skip')}
                                </DropdownMenuItem>
                            }
                        />
                    )}
                    {canDuplicate && (
                        <DropdownMenuItem onClick={() => onDuplicate(transaction.id)}>
                            <Copy className="mr-2 size-4" />
                            {t('actions.duplicate')}
                        </DropdownMenuItem>
                    )}
                </RowActions>
            )}
            below={expanded && canExpand ? <TransactionItemsRow transaction={transaction} /> : undefined}
        />
    )
}
