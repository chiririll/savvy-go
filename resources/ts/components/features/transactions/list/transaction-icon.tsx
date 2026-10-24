import {
    ArrowDownLeft,
    ArrowLeftRight,
    ArrowUpRight,
    Banknote,
    HandCoins,
} from 'lucide-react'
import { categoryIconStyle } from '@/lib/category-color'
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
    income: 'bg-success/15 text-success',
    expense: 'bg-destructive/15 text-destructive',
    transfer: 'bg-info/15 text-info',
    debt_payment: 'bg-warning/15 text-warning',
    debt_collection: 'bg-purple-100 text-purple-600 dark:bg-purple-950/40',
    debt_lend: 'bg-destructive/15 text-destructive',
    debt_borrow: 'bg-success/15 text-success',
    transfer_out: 'bg-info/15 text-info',
    transfer_in: 'bg-info/15 text-info',
} as const

/** FeedRow icon props of a transaction: its category's emoji, else its type. */
export function transactionIconProps(type: TransactionType, category?: Pick<Category, 'icon' | 'color'> | null) {
    const TypeIcon = TYPE_ICONS[type]
    return {
        icon: category?.icon ? <span aria-hidden>{category.icon}</span> : <TypeIcon className="size-4" />,
        iconClassName: category?.color ? undefined : TYPE_ICON_TONES[type],
        iconStyle: categoryIconStyle(category?.color),
    }
}
