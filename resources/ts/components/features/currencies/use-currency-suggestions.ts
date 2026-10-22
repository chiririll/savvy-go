import { useMemo, useState } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useCurrencyCatalog } from '@/hooks'
import type { CurrencyFormData } from '@/schemas'
import type { CurrencyCatalogItem } from '@/types'

function filterCatalog(catalog: CurrencyCatalogItem[], query: string, field: 'code' | 'name') {
    const q = query.trim().toLowerCase()
    if (!q) {
        return []
    }

    return catalog
        .filter((item) =>
            field === 'code'
                ? item.code.toLowerCase().startsWith(q) || item.name.toLowerCase().includes(q)
                : item.name.toLowerCase().includes(q) || item.code.toLowerCase().startsWith(q),
        )
        .slice(0, 8)
}

// Catalog suggestions under the code and name inputs; picking one fills the whole currency.
export function useCurrencySuggestions(form: UseFormReturn<CurrencyFormData>, isEditing: boolean) {
    const [suggestField, setSuggestField] = useState<'code' | 'name' | null>(null)
    const { data: catalog = [] } = useCurrencyCatalog(!isEditing)

    const code = form.watch('code')
    const name = form.watch('name')

    const suggestions = useMemo(() => {
        if (isEditing || !suggestField) {
            return []
        }

        return filterCatalog(catalog, suggestField === 'code' ? code : name, suggestField)
    }, [catalog, code, isEditing, name, suggestField])

    const applyCatalogItem = (item: CurrencyCatalogItem) => {
        const options = { shouldValidate: true, shouldDirty: true }
        form.setValue('code', item.code, options)
        form.setValue('name', item.name, options)
        form.setValue('symbol', item.symbol, options)
        form.setValue('decimals', item.decimals, options)
        if (item.rate) {
            form.setValue('rate', item.rate, options)
        }
        setSuggestField(null)
    }

    return { suggestField, setSuggestField, suggestions, applyCatalogItem }
}
