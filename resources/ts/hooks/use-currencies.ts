import { useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { currenciesApi } from '@/api'
import { useCurrenciesStore } from '@/stores/currencies'
import { Currency } from '@/types'
import { CurrencyFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n/i18n'

const QUERY_KEY = ['currencies']

export function useCurrencies() {
    const query = useQuery({
        queryKey: QUERY_KEY,
        queryFn: async () => {
            const data = await currenciesApi.getAll()
            useCurrenciesStore.getState().setAll(data)
            return data
        },
    })

    const synced = useRef<Currency[] | undefined>(undefined)

    if (query.data && query.data !== synced.current) {
        synced.current = query.data
        useCurrenciesStore.getState().setAll(query.data)
    }

    return query
}

export function useCurrencyCatalog(enabled = true) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'catalog'],
        queryFn: currenciesApi.getCatalog,
        enabled,
        staleTime: 60 * 60 * 1000,
    })
}

export function useCurrency(id: string | number) {
    return useResourceItem(QUERY_KEY, () => currenciesApi.getById(id), id)
}

export function useCreateCurrency(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: CurrencyFormData) => currenciesApi.create(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.currency.created'),
        redirectTo,
    })
}

export function useUpdateCurrency(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<CurrencyFormData> }) =>
            currenciesApi.update(id, data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.currency.updated'),
        redirectTo,
    })
}

export function useDeleteCurrency() {
    return useResourceMutation({
        mutationFn: (id: string | number) => currenciesApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.currency.deleted'),
    })
}

export function useSetBaseCurrency() {
    return useResourceMutation({
        mutationFn: (id: string | number) => currenciesApi.setBase(id),
        invalidateAll: true,
        successMessage: i18n.t('toasts.currency.baseUpdated'),
    })
}
