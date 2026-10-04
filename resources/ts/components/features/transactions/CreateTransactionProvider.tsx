import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useCreateTransaction, useSendTransfer, useUpdateTransaction } from '@/hooks'
import { TransactionFormValues } from '@/schemas'
import { Transaction } from '@/types'
import { TransactionFormDialog, toSpaceTransferInput } from './TransactionFormDialog'

export type CreateTransactionDefaults = Partial<TransactionFormValues>

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
    const sendTransfer = useSendTransfer()
    const [open, setOpen] = useState(false)
    const [defaults, setDefaults] = useState<CreateTransactionDefaults>()
    const [transaction, setTransaction] = useState<Transaction | null>(null)

    const openCreate = useCallback((next?: CreateTransactionDefaults) => {
        setTransaction(null)
        setDefaults(next)
        setOpen(true)
    }, [])

    const openEdit = useCallback((next: Transaction) => {
        if (!next.actions.edit) {
            return
        }

        setDefaults(undefined)
        setTransaction(next)
        setOpen(true)
    }, [])

    const handleSubmit = (data: TransactionFormValues) => {
        if (transaction) {
            updateTransaction.mutate(
                { id: transaction.id, data },
                { onSuccess: () => setOpen(false) }
            )
            return
        }

        const onSuccess = () => {
            setOpen(false)
            if (location.pathname !== '/transactions') {
                navigate('/transactions')
            }
        }

        if (data.type === 'transfer_out') {
            sendTransfer.mutate(toSpaceTransferInput(data), { onSuccess })
            return
        }

        createTransaction.mutate(data, { onSuccess })
    }

    const value = useMemo(() => ({ openCreate, openEdit }), [openCreate, openEdit])

    return (
        <TransactionFormContext.Provider value={value}>
            {children}
            <TransactionFormDialog
                transaction={transaction}
                open={open}
                onOpenChange={setOpen}
                onSubmit={handleSubmit}
                isSubmitting={createTransaction.isPending || updateTransaction.isPending || sendTransfer.isPending}
                defaultValues={defaults}
            />
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
