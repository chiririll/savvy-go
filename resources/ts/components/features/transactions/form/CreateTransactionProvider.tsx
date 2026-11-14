import { createContext, lazy, Suspense, useCallback, useContext, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useCreateTransaction, useSendTransfer, useUpdateTransaction } from '@/hooks'
import i18n from '@/lib/i18n/i18n'
import { formModeOf, refusalReason, type TransactionPatch } from '@/lib/transaction/transaction-details'
import type { SpaceTransferInput, Transaction, TransactionInput } from '@/types'
import { restrictionText } from './restriction-text'
import { fromCreateDefaults, type CreateTransactionDefaults } from './create-defaults'
import type { TransactionFormDefaults } from './use-transaction-form'

// The form pulls in zod, react-hook-form and most shared inputs; keep it out of the entry chunk.
const TransactionFormDialog = lazy(() =>
    import('./TransactionFormDialog').then((m) => ({ default: m.TransactionFormDialog })),
)

// Same for the detail dialog: it brings the mutations and the confirm dialogs along.
const TransactionDetailHost = lazy(() => import('../detail/TransactionDetailHost'))

export type { CreateTransactionDefaults }

interface TransactionFormContextValue {
    openCreate: (defaults?: CreateTransactionDefaults) => void
    openEdit: (transaction: Transaction) => void
    /** Opens the view-only dialog of a transaction; what a click on a row does. */
    openView: (transaction: Transaction) => void
}

const TransactionFormContext = createContext<TransactionFormContextValue | null>(null)

export function CreateTransactionProvider({ children }: { children: React.ReactNode }) {
    const location = useLocation()
    const navigate = useNavigate()
    const createTransaction = useCreateTransaction()
    const updateTransaction = useUpdateTransaction()
    const sendTransfer = useSendTransfer()
    const [open, setOpen] = useState(false)
    const [mounted, setMounted] = useState(false)
    const [defaults, setDefaults] = useState<TransactionFormDefaults>()
    const [transaction, setTransaction] = useState<Transaction | null>(null)
    const [viewed, setViewed] = useState<Transaction | null>(null)
    const [viewOpen, setViewOpen] = useState(false)

    const openCreate = useCallback((next?: CreateTransactionDefaults) => {
        setTransaction(null)
        setDefaults(fromCreateDefaults(next))
        setMounted(true)
        setOpen(true)
    }, [])

    const openEdit = useCallback((next: Transaction) => {
        // The whole form, or the one with only the details, when that is all the server allows or all the form can
        // show without dropping something. What nothing can change says why, when the server gave a reason.
        if (formModeOf(next) === null) {
            const reason = refusalReason(next)
            if (reason) {
                toast.error(restrictionText(i18n.t.bind(i18n) as (key: string) => string, reason))
            }
            return
        }

        setDefaults(undefined)
        setTransaction(next)
        setMounted(true)
        setOpen(true)
    }, [])

    const openView = useCallback((next: Transaction) => {
        setViewed(next)
        setViewOpen(true)
    }, [])

    const onCreated = () => {
        setOpen(false)
        if (location.pathname !== '/transactions') {
            navigate('/transactions')
        }
    }

    const handleSubmit = (data: TransactionInput) => {
        if (transaction) {
            updateTransaction.mutate({ id: transaction.id, data }, { onSuccess: () => setOpen(false) })
            return
        }

        createTransaction.mutate(data, { onSuccess: onCreated })
    }

    const handleSubmitDetails = (data: TransactionPatch) => {
        if (transaction) {
            updateTransaction.mutate({ id: transaction.id, data }, { onSuccess: () => setOpen(false) })
        }
    }

    const handleSubmitSpaceTransfer = (data: SpaceTransferInput) => sendTransfer.mutate(data, { onSuccess: onCreated })

    const value = useMemo(() => ({ openCreate, openEdit, openView }), [openCreate, openEdit, openView])

    return (
        <TransactionFormContext.Provider value={value}>
            {children}
            {viewed && (
                <Suspense fallback={null}>
                    <TransactionDetailHost
                        transaction={viewed}
                        open={viewOpen}
                        onOpenChange={setViewOpen}
                        onEdit={openEdit}
                    />
                </Suspense>
            )}
            {mounted && (
                <Suspense fallback={null}>
                    <TransactionFormDialog
                        transaction={transaction}
                        open={open}
                        onOpenChange={setOpen}
                        onSubmit={handleSubmit}
                        onSubmitDetails={handleSubmitDetails}
                        onSubmitSpaceTransfer={handleSubmitSpaceTransfer}
                        isSubmitting={createTransaction.isPending || updateTransaction.isPending || sendTransfer.isPending}
                        defaultValues={defaults}
                    />
                </Suspense>
            )}
        </TransactionFormContext.Provider>
    )
}

export function useTransactionFormDialog() {
    const context = useContext(TransactionFormContext)
    if (!context) {
        throw new Error('useTransactionFormDialog must be used within CreateTransactionProvider')
    }
    return context
}

export const useCreateTransactionDialog = useTransactionFormDialog
