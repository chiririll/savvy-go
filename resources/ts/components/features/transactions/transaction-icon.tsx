import {
    ArrowDownLeft,
    ArrowLeftRight,
    ArrowUpRight,
    Banknote,
    HandCoins,
} from 'lucide-react'
import type { Category, TransactionType } from '@/types'

const TYPE_ICONS = {
    income: ArrowDownLeft,
    expense: ArrowUpRight,
    transfer: ArrowLeftRight,
    debt_payment: Banknote,
    debt_collection: HandCoins,
    debt_lend: HandCoins,
    debt_borrow: Banknote,
    transfer_out: ArrowUpRight,
    transfer_in: ArrowDownLeft,
} as const

const TYPE_ICON_TONES = {
    income: 'bg-green-100 text-green-600 dark:bg-green-950/40',
    expense: 'bg-red-100 text-red-600 dark:bg-red-950/40',
    transfer: 'bg-blue-100 text-blue-600 dark:bg-blue-950/40',
    debt_payment: 'bg-orange-100 text-orange-600 dark:bg-orange-950/40',
    debt_collection: 'bg-purple-100 text-purple-600 dark:bg-purple-950/40',
    debt_lend: 'bg-red-100 text-red-600 dark:bg-red-950/40',
    debt_borrow: 'bg-green-100 text-green-600 dark:bg-green-950/40',
    transfer_out: 'bg-blue-100 text-blue-600 dark:bg-blue-950/40',
    transfer_in: 'bg-blue-100 text-blue-600 dark:bg-blue-950/40',
} as const

/** FeedRow icon props of a transaction: its category's emoji, else its type. */
export function transactionIconProps(type: TransactionType, category?: Pick<Category, 'icon' | 'color'> | null) {
    const TypeIcon = TYPE_ICONS[type]
    return {
        icon: category?.icon ? <span aria-hidden>{category.icon}</span> : <TypeIcon className="size-4" />,
        iconClassName: category?.color ? undefined : TYPE_ICON_TONES[type],
        iconStyle: category?.color ? { backgroundColor: `${category.color}20` } : undefined,
    }
}
