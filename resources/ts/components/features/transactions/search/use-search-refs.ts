import { useMemo } from 'react'
import { useAccounts, useCategories, useCurrencies, useTags } from '@/hooks'
import { localizeDefaultName } from '@/lib/localized-name'
import type { SearchRef, SearchRefs } from '@/lib/transaction-search'

function names(shown: string, stored: string): string[] {
    return shown === stored ? [shown] : [shown, stored]
}

/** The names the search keywords can resolve, from the lists the app already holds. */
export function useSearchRefs(): SearchRefs {
    const { data: accounts } = useAccounts({ exclude_debts: true })
    const { data: categories } = useCategories()
    const { data: tags } = useTags()
    const { data: currencies } = useCurrencies()

    return useMemo(() => {
        const refs = (list: Array<{ id: number; name: string }> | undefined, group?: 'categories' | 'tags'): SearchRef[] =>
            (list ?? []).map((item) => ({
                id: item.id,
                names: group ? names(localizeDefaultName(item.name, group), item.name) : [item.name],
            }))
        return {
            accounts: refs(accounts),
            categories: refs(categories, 'categories'),
            tags: refs(tags, 'tags'),
            currencies: (currencies ?? []).map((currency) => currency.code),
        }
    }, [accounts, categories, tags, currencies])
}
