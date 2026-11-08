import { useState } from 'react'
import { Check, MoreHorizontal, Pencil, SkipForward, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Confirm } from '@/components/shared/dialogs/Confirm'
import { ResponsiveDialog } from '@/components/shared/dialogs/ResponsiveDialog'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useIsMobile } from '@/hooks/use-mobile'
import { displayTransactionDescription } from '@/lib/transaction-description'
import { formModeOf } from '@/lib/transaction-details'
import type { Transaction } from '@/types'
import { TransactionDetailContent } from './TransactionDetailContent'

export interface TransactionDetailDialogProps {
    transaction: Transaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    /** Each action is offered only when given, the transaction allows it and the role is not read-only. */
    onEdit?: (transaction: Transaction) => void
    onConfirm?: (transaction: Transaction) => void
    onSkip?: (transaction: Transaction) => void
    onDelete?: (transaction: Transaction) => void
    isReadOnly?: boolean
}

type Asking = 'skip' | 'delete' | null

interface Handlers {
    edit?: () => void
    confirm?: () => void
    skip?: () => void
    remove?: () => void
}

/** Each action the footer offers, or undefined when the transaction, the role or the caller rules it out. */
function handlersOf(
    transaction: Transaction,
    { onEdit, onConfirm, onSkip, onDelete, isReadOnly }: TransactionDetailDialogProps,
    ask: (asking: Asking) => void,
): Handlers {
    const { actions } = transaction
    const writable = !isReadOnly
    return {
        edit: writable && onEdit && formModeOf(transaction) !== null ? () => onEdit(transaction) : undefined,
        confirm: writable && actions.confirm && onConfirm ? () => onConfirm(transaction) : undefined,
        skip: writable && actions.skip && onSkip ? () => ask('skip') : undefined,
        remove: writable && actions.delete && onDelete ? () => ask('delete') : undefined,
    }
}

function MobileMenu({ skip, remove }: Pick<Handlers, 'skip' | 'remove'>) {
    const { t } = useTranslation('common')
    if (!skip && !remove) {
        return null
    }

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon" aria-label={t('actions.more')}>
                    <MoreHorizontal className="size-4" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
                {skip && (
                    <DropdownMenuItem onClick={skip}>
                        <SkipForward className="mr-2 size-4" />
                        {t('actions.skip')}
                    </DropdownMenuItem>
                )}
                {remove && (
                    <DropdownMenuItem variant="destructive" onClick={remove}>
                        <Trash2 className="mr-2 size-4" />
                        {t('actions.delete')}
                    </DropdownMenuItem>
                )}
            </DropdownMenuContent>
        </DropdownMenu>
    )
}

/** Delete at the start; the rest at the end. On a phone only Edit and Confirm stay as buttons. */
function DetailFooter({ handlers, isMobile }: { handlers: Handlers; isMobile: boolean }) {
    const { t } = useTranslation('common')
    const { edit, confirm, skip, remove } = handlers

    return (
        <div className="flex w-full items-center gap-2">
            {!isMobile && remove && (
                <Button variant="ghost" className="text-destructive hover:text-destructive" onClick={remove}>
                    <Trash2 className="size-4" />
                    {t('actions.delete')}
                </Button>
            )}
            <div className="ml-auto flex items-center gap-2">
                {!isMobile && skip && (
                    <Button variant="outline" onClick={skip}>
                        <SkipForward className="size-4" />
                        {t('actions.skip')}
                    </Button>
                )}
                {edit && (
                    <Button variant="outline" onClick={edit}>
                        <Pencil className="size-4" />
                        {t('actions.edit')}
                    </Button>
                )}
                {confirm && (
                    <Button onClick={confirm}>
                        <Check className="size-4" />
                        {t('actions.confirm')}
                    </Button>
                )}
                {isMobile && <MobileMenu skip={skip} remove={remove} />}
            </div>
        </div>
    )
}

/** A view-only dialog of a transaction, with the actions of `transaction.actions` in its footer. */
export function TransactionDetailDialog(props: TransactionDetailDialogProps) {
    const { transaction, open, onOpenChange, onSkip, onDelete } = props
    const { t } = useTranslation(['common', 'pages'])
    const isMobile = useIsMobile()
    const [asking, setAsking] = useState<Asking>(null)

    if (!transaction) {
        return null
    }

    const handlers = handlersOf(transaction, props, setAsking)

    return (
        <>
            <ResponsiveDialog
                open={open}
                onOpenChange={onOpenChange}
                title={displayTransactionDescription(transaction)}
                footer={Object.values(handlers).some(Boolean) ? <DetailFooter handlers={handlers} isMobile={isMobile} /> : undefined}
                className="sm:max-w-lg"
            >
                <TransactionDetailContent transaction={transaction} onNavigate={() => onOpenChange(false)} />
            </ResponsiveDialog>
            <Confirm
                open={asking === 'skip'}
                onOpenChange={(next) => !next && setAsking(null)}
                title={t('pages:transactions.skipTitle')}
                description={t('pages:transactions.skipDescription')}
                action={t('common:actions.skip')}
                onConfirm={() => onSkip?.(transaction)}
            />
            <Confirm
                open={asking === 'delete'}
                onOpenChange={(next) => !next && setAsking(null)}
                title={t('pages:transactions.deleteTitle')}
                description={t('pages:transactions.deleteDescription')}
                action={t('common:actions.delete')}
                destructive
                onConfirm={() => onDelete?.(transaction)}
            />
        </>
    )
}
