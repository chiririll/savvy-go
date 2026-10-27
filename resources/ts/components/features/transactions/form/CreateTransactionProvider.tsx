import { createContext, lazy, Suspense, useCallback, useContext, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { transactionGroupsApi } from '@/api'
import {
    useCreateTransaction,
    useCreateTransactionGroup,
    useSendTransfer,
    useUpdateTransaction,
    useUpdateTransactionGroup,
} from '@/hooks'
import i18n from '@/lib/i18n'
import { getApiErrorMessage } from '@/lib/api-error'
import { paymentFormValues } from '@/lib/payment-input'
import type { PaymentLeg } from '@/lib/split-payment'
import type { TransactionFormValues } from '@/schemas'
import type { Transaction, TransactionGroup } from '@/types'
import { toSpaceTransferInput } from './space-transfer-input'

// The form pulls in zod, react-hook-form and most shared inputs; keep it out of the entry chunk.
const TransactionFormDialog = lazy(() =>
    import('./TransactionFormDialog').then((m) => ({ default: m.TransactionFormDialog })),
)

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
    const createGroup = useCreateTransactionGroup()
    const updateGroup = useUpdateTransactionGroup()
    const [open, setOpen] = useState(false)
    const [mounted, setMounted] = useState(false)
    const [defaults, setDefaults] = useState<CreateTransactionDefaults>()
    const [transaction, setTransaction] = useState<Transaction | null>(null)
    const [group, setGroup] = useState<TransactionGroup | null>(null)

    const openCreate = useCallback((next?: CreateTransactionDefaults) => {
        setTransaction(null)
        setGroup(null)
        setDefaults(next)
        setMounted(true)
        setOpen(true)
    }, [])

    const openEdit = useCallback((next: Transaction) => {
        if (!next.actions.edit) {
            return
        }

        setDefaults(undefined)

        // A leg of a split payment is edited with the others.
        if (next.groupUuid) {
            transactionGroupsApi
                .getByUuid(next.groupUuid)
                .then((loaded) => {
                    if (!paymentFormValues(loaded)) {
                        toast.error(i18n.t('toasts.transaction.paymentNotEditable'))
                        return
                    }
                    setTransaction(null)
                    setGroup(loaded)
                    setMounted(true)
                    setOpen(true)
                })
                .catch((error) => toast.error(getApiErrorMessage(error, i18n.t('toasts.transaction.updateFailed'))))
            return
        }

        setGroup(null)
        setTransaction(next)
        setMounted(true)
        setOpen(true)
    }, [])

    const handleSubmitPayment = (legs: PaymentLeg[]) => {
        if (group) {
            updateGroup.mutate({ uuid: group.uuid, legs }, { onSuccess: () => setOpen(false) })
            return
        }
        createGroup.mutate(legs, {
            onSuccess: () => {
                setOpen(false)
                if (location.pathname !== '/transactions') {
                    navigate('/transactions')
                }
            },
        })
    }

    const handleSubmit = (data: TransactionFormValues) => {
        if (transaction) {
            updateTransaction.mutate({ id: transaction.id, data }, { onSuccess: () => setOpen(false) })
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
            {mounted && (
                <Suspense fallback={null}>
                    <TransactionFormDialog
                        transaction={transaction}
                        group={group}
                        open={open}
                        onOpenChange={setOpen}
                        onSubmit={handleSubmit}
                        onSubmitPayment={handleSubmitPayment}
                        isSubmitting={
                            createTransaction.isPending ||
                            updateTransaction.isPending ||
                            sendTransfer.isPending ||
                            createGroup.isPending ||
                            updateGroup.isPending
                        }
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
