import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { useDeleteTransaction, useSkipTransaction } from '@/hooks'
import type { Transaction } from '@/types'
import { ApplyDeferredDateDialog } from '../deferred/ApplyDeferredDateDialog'
import { useApplyDeferred } from '../deferred/use-apply-deferred'
import { TransactionDetailDialog } from './TransactionDetailDialog'

interface TransactionDetailHostProps {
    transaction: Transaction | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onEdit: (transaction: Transaction) => void
}

/** The detail dialog wired to the mutations; every action closes it once it has gone through. */
export default function TransactionDetailHost({ transaction, open, onOpenChange, onEdit }: TransactionDetailHostProps) {
    const isReadOnly = useReadOnly()
    const deleteTransaction = useDeleteTransaction()
    const skipTransaction = useSkipTransaction()
    const { startApplying, dialogProps } = useApplyDeferred()
    const close = { onSuccess: () => onOpenChange(false) }

    return (
        <>
            <TransactionDetailDialog
                transaction={transaction}
                open={open}
                onOpenChange={onOpenChange}
                isReadOnly={isReadOnly}
                onEdit={(next) => {
                    onOpenChange(false)
                    onEdit(next)
                }}
                onConfirm={(next) => {
                    onOpenChange(false)
                    startApplying(next)
                }}
                onSkip={(next) => skipTransaction.mutate(next.id, close)}
                onDelete={(next) => deleteTransaction.mutate(next.id, close)}
            />
            <ApplyDeferredDateDialog {...dialogProps} />
        </>
    )
}
