import { useState } from 'react'
import { Check, Copy, SkipForward } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { FeedRow, FeedStatusBadge, RowActions } from '@/components/shared'
import { useRelativeDay } from '@/hooks'
import { cn, formatCurrency } from '@/lib/utils'
import { displayTransactionDescription, transactionAmountAppearance, transactionSubtitle } from '@/lib/transaction-description'
import { Transaction } from '@/types'
import { SkipTransactionAlert } from './SkipTransactionAlert'
import { TransactionItemsRow } from './TransactionItemsRow'
import { transactionIconProps } from './transaction-icon'

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
    const { t } = useTranslation(['common', 'pages'])
    const relativeDay = useRelativeDay()
    const dateLabel = showDate ? relativeDay(transaction.date) : null
    const [expanded, setExpanded] = useState(false)
    const itemsCount = transaction.items.length
    const canExpand = itemsCount > 1
    const { sign, className } = transactionAmountAppearance(transaction.type, transaction.status)
    const isTransfer = transaction.type === 'transfer'
    const { edit, duplicate, delete: canRemove, confirm, skip } = transaction.actions
    const canEdit = edit && !!onEdit
    const canConfirm = !isReadOnly && confirm && !!onConfirm
    const canSkip = !isReadOnly && skip && !!onSkip
    const canDuplicate = !isReadOnly && duplicate && !!onDuplicate
    const canDelete = !isReadOnly && canRemove && !!onDelete
    const hasActions = showActions && (canEdit || canConfirm || canSkip || canDuplicate || canDelete)

    return (
        <FeedRow
            {...transactionIconProps(transaction.type, transaction.category)}
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
            amount={`${transaction.isEstimated ? '≈ ' : ''}${sign}${formatCurrency(transaction.amount, transaction.account.currency)}`}
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
