import type { TFunction } from 'i18next'
import { localizeDefaultName } from '@/lib/localized-name'
import type { TopTransactionItem } from '@/api/reports'

type Item = Pick<TopTransactionItem, 'description' | 'category' | 'account' | 'category_count' | 'account_count'>

/** The description, or the category, or the given fallback for a transaction with neither. */
export function topTransactionTitle(item: Item, fallback: string): string {
    return item.description || localizeDefaultName(item.category?.name) || fallback
}

/** The category, "N categories" for several lines, nothing when uncategorised. */
export function topTransactionCategory(item: Item, t: TFunction<'pages'>): string {
    if (item.category_count > 1) {
        return t('transactions.categoriesCount', { count: item.category_count })
    }
    return localizeDefaultName(item.category?.name)
}

/** The account that moved the most, and "+N" for the other accounts. */
export function topTransactionAccount(item: Item): string {
    const others = item.account_count - 1
    return others > 0 ? `${item.account.name} +${others}` : item.account.name
}
