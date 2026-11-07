import { createContext, lazy, Suspense, useCallback, useContext, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useCreateTransaction, useUpdateTransaction } from '@/hooks'
import i18n from '@/lib/i18n'
import { isSimpleTransaction } from '@/lib/transaction-form-values'
import type { Transaction, TransactionInput } from '@/types'
import { fromCreateDefaults, type CreateTransactionDefaults } from './create-defaults'
import type { TransactionFormDefaults } from './use-transaction-form'

// The form pulls in zod, react-hook-form and most shared inputs; keep it out of the entry chunk.
const TransactionFormDialog = lazy(() =>
    import('./TransactionFormDialog').then((m) => ({ default: m.TransactionFormDialog })),
)

export type { CreateTransactionDefaults }

interface TransactionFormContextValue {
    openCreate: (defaults?: CreateTransactionDefaults) => void
    openEdit: (transaction: Transaction) => void
}

const TransactionFormContext = createContext<TransactionFormContextValue | null>(null)

export function CreateTransactionProvider({ children }: { children: React.ReactNode }) {
    const location = useLocation()
    const navigate = useNavigate()
    const createTransaction = useCreateTransaction()
    const updateTransaction = useUpdateTransaction()
    const [open, setOpen] = useState(false)
    const [mounted, setMounted] = useState(false)
    const [defaults, setDefaults] = useState<TransactionFormDefaults>()
    const [transaction, setTransaction] = useState<Transaction | null>(null)

    const openCreate = useCallback((next?: CreateTransactionDefaults) => {
        setTransaction(null)
        setDefaults(fromCreateDefaults(next))
        setMounted(true)
        setOpen(true)
    }, [])

    const openEdit = useCallback((next: Transaction) => {
        if (!next.actions.edit) {
            return
        }
        // The form shows one payment in one line for now; several payments and lines, change and
        // transfers come with the next steps of the form. Better a message than a save that drops them.
        if (!isSimpleTransaction(next)) {
            toast.error(i18n.t('toasts.transaction.notEditableYet'))
            return
        }

        setDefaults(undefined)
        setTransaction(next)
        setMounted(true)
        setOpen(true)
    }, [])

    const handleSubmit = (data: TransactionInput) => {
        if (transaction) {
            updateTransaction.mutate({ id: transaction.id, data }, { onSuccess: () => setOpen(false) })
            return
        }

        createTransaction.mutate(data, {
            onSuccess: () => {
                setOpen(false)
                if (location.pathname !== '/transactions') {
                    navigate('/transactions')
                }
            },
        })
    }

    const value = useMemo(() => ({ openCreate, openEdit }), [openCreate, openEdit])

    return (
        <TransactionFormContext.Provider value={value}>
            {children}
            {mounted && (
                <Suspense fallback={null}>
                    <TransactionFormDialog
                        transaction={transaction}
                        open={open}
                        onOpenChange={setOpen}
                        onSubmit={handleSubmit}
                        isSubmitting={createTransaction.isPending || updateTransaction.isPending}
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
