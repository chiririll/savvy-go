import { useState } from 'react'
import { useConfirmTransaction } from '@/hooks'
import type { Transaction } from '@/types'
import type { ApplyOptions } from './apply-deferred'

/** Which pending transaction is being applied, with the props that wire it to ApplyDeferredDateDialog. */
export function useApplyDeferred() {
    const confirmTransaction = useConfirmTransaction()
    const [applying, setApplying] = useState<Transaction | null>(null)

    const dialogProps = {
        transaction: applying,
        open: applying !== null,
        onOpenChange: (open: boolean) => {
            if (!open) setApplying(null)
        },
        isSubmitting: confirmTransaction.isPending,
        onConfirm: (options: ApplyOptions) => {
            if (!applying) return
            confirmTransaction.mutate({ id: applying.id, ...options }, { onSuccess: () => setApplying(null) })
        },
    }

    return { startApplying: setApplying, dialogProps }
}
