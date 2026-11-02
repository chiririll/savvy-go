import { testId } from '@/lib/test-id'
import { useState } from 'react'
import { Check, Copy, SkipForward } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { FeedRow, FeedStatusBadge, RowActions } from '@/components/shared'
import { useRelativeDay } from '@/hooks'
import { cn, formatCurrency } from '@/lib/utils'
import {
    displayTransactionDescription,
    transactionAmountAppearance,
    transactionSubtitle,
} from '@/lib/transaction-description'
import { Transaction } from '@/types'
import { SkipTransactionAlert } from '../deferred/SkipTransactionAlert'
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

interface RowHandlers {
    edit?: () => void
    confirm?: () => void
    skip?: () => void
    duplicate?: () => void
    remove?: () => void
}

/** Each action the row offers, or undefined when the transaction, the role or the caller rules it out. */
function rowHandlers(
    transaction: Transaction,
    { onEdit, onConfirm, onSkip, onDuplicate, onDelete, isReadOnly }: TransactionRowProps,
): RowHandlers {
    const { edit, duplicate, delete: canRemove, confirm, skip } = transaction.actions
    const writable = !isReadOnly
    return {
        edit: edit && onEdit ? () => onEdit(transaction) : undefined,
        confirm: writable && confirm && onConfirm ? () => onConfirm(transaction) : undefined,
        skip: writable && skip && onSkip ? () => onSkip(transaction.id) : undefined,
        duplicate: writable && duplicate && onDuplicate ? () => onDuplicate(transaction.id) : undefined,
        remove: writable && canRemove && onDelete ? () => onDelete(transaction.id) : undefined,
    }
}

function TransactionStatusBadge({ status }: { status: Transaction['status'] }) {
    const { t } = useTranslation('pages')
    if (status === 'skipped') {
        return <FeedStatusBadge variant="secondary">{t('transactions.status.skipped')}</FeedStatusBadge>
    }
    if (status === 'pending') {
        return <FeedStatusBadge variant="outline">{t('transactions.status.pending')}</FeedStatusBadge>
    }
    return null
}

interface TransactionMenuProps {
    handlers: RowHandlers
    open: boolean
    onOpenChange: (open: boolean) => void
    showTrigger: boolean
}

function TransactionMenu({ handlers, open, onOpenChange, showTrigger }: TransactionMenuProps) {
    const { t } = useTranslation(['common', 'pages'])
    return (
        <RowActions
            open={open}
            onOpenChange={onOpenChange}
            showTrigger={showTrigger}
            onEdit={handlers.edit}
            onDelete={handlers.remove}
            deleteTitle={t('pages:transactions.deleteTitle')}
            deleteDescription={t('pages:transactions.deleteDescription')}
        >
            {handlers.confirm && (
                <DropdownMenuItem onClick={handlers.confirm}>
                    <Check className="mr-2 size-4" />
                    {t('actions.confirm')}
                </DropdownMenuItem>
            )}
            {handlers.skip && (
                <SkipTransactionAlert
                    onConfirm={handlers.skip}
                    trigger={
                        <DropdownMenuItem onSelect={(event) => event.preventDefault()} {...testId('skip')}>
                            <SkipForward className="mr-2 size-4" />
                            {t('actions.skip')}
                        </DropdownMenuItem>
                    }
                />
            )}
            {handlers.duplicate && (
                <DropdownMenuItem onClick={handlers.duplicate}>
                    <Copy className="mr-2 size-4" />
                    {t('actions.duplicate')}
                </DropdownMenuItem>
            )}
        </RowActions>
    )
}

export function TransactionRow(props: TransactionRowProps) {
    const { transaction, showActions = true, showDate = false } = props
    const { t } = useTranslation('pages')
    const relativeDay = useRelativeDay()
    const [expanded, setExpanded] = useState(false)
    const itemsCount = transaction.items.length
    const canExpand = itemsCount > 1
    const { sign, className } = transactionAmountAppearance(transaction.type, transaction.status)
    const amount = formatCurrency(transaction.amount, transaction.account.currency)
    const { toAmount, toAccount } = transaction
    const received =
        transaction.type === 'transfer' && toAmount != null && toAccount
            ? formatCurrency(toAmount, toAccount.currency)
            : null
    const handlers = rowHandlers(transaction, props)
    const dateLabel = showDate ? relativeDay(transaction.date, { weekday: true }) : null

    return (
        <FeedRow
            {...transactionIconProps(transaction.type, transaction.category)}
            title={displayTransactionDescription(transaction)}
            badge={<TransactionStatusBadge status={transaction.status} />}
            subtitle={
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
                            ({t('transactions.items.count', { count: itemsCount })})
                        </button>
                    )}
                </>
            }
            amount={`${transaction.isEstimated ? '≈ ' : ''}${sign}${amount}`}
            amountClassName={cn(className, transaction.status === 'pending' && 'italic')}
            extraAmount={received ? `${transaction.status === 'skipped' ? '' : '+'}${received}` : undefined}
            onOpen={handlers.edit}
            hasActions={showActions && Object.values(handlers).some(Boolean)}
            reserveActions={showActions && !props.isReadOnly}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <TransactionMenu
                    handlers={handlers}
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                />
            )}
            below={expanded && canExpand ? <TransactionItemsRow transaction={transaction} /> : undefined}
        />
    )
}
