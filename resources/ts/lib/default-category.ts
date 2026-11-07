import type { Category } from '@/types'

/** The category a new income or expense starts with: the default one, else the one used most. */
export function pickDefaultCategory(categories: Category[] | undefined, type: string): Category | undefined {
    const ofType = (categories ?? []).filter((category) => category.type === type)
    return ofType.find((category) => category.isDefault)
        ?? [...ofType].sort((a, b) => (b.transactionsCount ?? 0) - (a.transactionsCount ?? 0))[0]
}
