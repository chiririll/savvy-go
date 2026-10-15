import type { TransactionFormValues } from '@/schemas/transactions'
import type { SpaceTransferInput } from '@/types/spaces'

/** The transfers API request for a transfer_out the form produced. */
export function toSpaceTransferInput(data: TransactionFormValues): SpaceTransferInput {
    return {
        to_space_id: Number(data.to_space_id),
        from_account_id: data.account_id,
        to_account_id: Number(data.to_account_id),
        from_amount: String(data.amount),
        to_amount: String(data.to_amount ?? data.amount),
        date: data.date ?? '',
        description: data.description?.trim() || null,
    }
}
