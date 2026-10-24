import { ArrowDownLeft, ArrowLeftRight, ArrowUpRight, Boxes } from 'lucide-react'

export const TRANSACTION_TYPE_OPTIONS = [
    { value: 'income', icon: ArrowDownLeft, color: 'text-success' },
    { value: 'expense', icon: ArrowUpRight, color: 'text-destructive' },
    { value: 'transfer', icon: ArrowLeftRight, color: 'text-info' },
] as const

/** A transfer to a linked space: created like a transaction, stored by the transfers API. */
export const SPACE_TRANSFER_OPTION = { value: 'transfer_out', icon: Boxes, color: 'text-info' } as const

export const TRANSACTION_TYPES = TRANSACTION_TYPE_OPTIONS.map((option) => option.value)
