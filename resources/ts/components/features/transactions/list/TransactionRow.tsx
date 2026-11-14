import { testId } from '@/lib/test-id'
import { ArrowDown, ArrowUp, Check, Copy, SkipForward } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { FeedRow, FeedStatusBadge, RowActions } from '@/components/shared'
import { useRelativeDay } from '@/hooks'
import { cn } from '@/lib/utils'
import {
    displayTransactionDescription,
    transactionAmountAppearance,
    transactionSubtitle,
} from '@/lib/transaction/transaction-description'
import { Transaction } from '@/types'
import { SkipTransactionAlert } from '../deferred/SkipTransactionAlert'
import { TransactionAmount } from './TransactionAmount'
import { transactionIconProps } from './transaction-icon'

interface TransactionRowProps {
    transaction: Transaction
    onDelete?: (id: number) => void
    onDuplicate?: (id: number) => void
    onConfirm?: (transaction: Transaction) => void
    onSkip?: (id: number) => void
    onEdit?: (transaction: Transaction) => void
    /** What a click on the row does; the form (`onEdit`) when left out. The menu's Edit is always `onEdit`. */
    onOpen?: (transaction: Transaction) => void
    isReadOnly?: boolean
    showActions?: boolean
    showDate?: boolean
    /** The amount is a guess (the pending occurrence of a rule with an estimate): shown as `≈`. */
    approximate?: boolean
    /** Move the row one place up or down within its day; each is set only when the row can go that way. */
    onMoveUp?: () => void
    onMoveDown?: () => void
}

interface RowHandlers {
    edit?: () => void
    confirm?: () => void
    skip?: () => void
    duplicate?: () => void
    moveUp?: () => void
    moveDown?: () => void
    remove?: () => void
}

/** Each action the row offers, or undefined when the transaction, the role or the caller rules it out. */
function rowHandlers(
    transaction: Transaction,
    { onEdit, onConfirm, onSkip, onDuplicate, onDelete, onMoveUp, onMoveDown, isReadOnly }: TransactionRowProps,
): RowHandlers {
    const { edit, duplicate, delete: canRemove, confirm, skip } = transaction.actions
    const writable = !isReadOnly
    return {
        edit: edit && onEdit ? () => onEdit(transaction) : undefined,
        confirm: writable && confirm && onConfirm ? () => onConfirm(transaction) : undefined,
        skip: writable && skip && onSkip ? () => onSkip(transaction.id) : undefined,
        duplicate: writable && duplicate && onDuplicate ? () => onDuplicate(transaction.id) : undefined,
        moveUp: writable ? onMoveUp : undefined,
        moveDown: writable ? onMoveDown : undefined,
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
            {handlers.moveUp && (
                <DropdownMenuItem onClick={handlers.moveUp} {...testId('move-up')}>
                    <ArrowUp className="mr-2 size-4" />
                    {t('actions.moveUp')}
                </DropdownMenuItem>
            )}
            {handlers.moveDown && (
                <DropdownMenuItem onClick={handlers.moveDown} {...testId('move-down')}>
                    <ArrowDown className="mr-2 size-4" />
                    {t('actions.moveDown')}
                </DropdownMenuItem>
            )}
        </RowActions>
    )
}

export function TransactionRow(props: TransactionRowProps) {
    const { transaction, showActions = true, showDate = false } = props
    const relativeDay = useRelativeDay()
    const { sign, className } = transactionAmountAppearance(transaction.type, transaction.status)
    const isTransfer = transaction.type === 'transfer'
    const handlers = rowHandlers(transaction, props)
    const dateLabel = showDate ? relativeDay(transaction.date, { weekday: true }) : null
    const categories = [...new Map(transaction.lines.flatMap((l) => (l.category ? [[l.category.id, l.category]] : []))).values()]

    return (
        <FeedRow
            {...transactionIconProps(transaction.type, categories)}
            title={displayTransactionDescription(transaction)}
            badge={<TransactionStatusBadge status={transaction.status} />}
            subtitle={
                <>
                    {dateLabel && <span className="mr-1.5 font-medium">{dateLabel} ·</span>}
                    {transactionSubtitle(transaction)}
                </>
            }
            amount={<TransactionAmount amount={transaction.rowAmount} sign={sign} approximate={props.approximate} />}
            amountClassName={cn(className, transaction.status === 'pending' && 'italic')}
            extraAmount={
                isTransfer && transaction.receivedAmount ? (
                    <TransactionAmount
                        amount={transaction.receivedAmount}
                        sign={transaction.status === 'skipped' ? '' : '+'}
                        approximate={props.approximate}
                    />
                ) : undefined
            }
            onOpen={props.onOpen ? () => props.onOpen?.(transaction) : handlers.edit}
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
        />
    )
}
