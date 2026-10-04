import { api, apiClient } from './client'
import { createCrudApi } from './crud'
import { toQueryString } from '@/lib/query-string'
import { Account } from '@/types'
import { AccountFormData } from '@/schemas'

export interface AccountsSummary {
    totalBalance: number
    currency: string | null
    decimals: number
    accountsCount: number
}

export interface AccountsResponse {
    data: Account[]
    summary?: AccountsSummary
}

export interface BalanceHistorySeries {
    id: number | null
    name: string
    type: string
    data: number[]
    native_data?: number[]
    currency: string | null
}

export interface BalanceHistoryResponse {
    dates: string[]
    series: BalanceHistorySeries[]
    currency: string | null
    decimals: number
}

export interface BalanceComparisonResponse {
    current: number
    previous: number | null
    currency: string | null
    decimals: number
}

const crud = createCrudApi<Account, AccountFormData>('/accounts')

export const accountsApi = {
    ...crud,

    getAll: (params?: { active?: boolean; exclude_debts?: boolean }) =>
        crud.getAll(params),

    getAllWithSummary: async (params?: { active?: boolean; exclude_debts?: boolean }): Promise<AccountsResponse> => {
        const response = await apiClient.get(`/accounts${toQueryString({
            with_summary: true,
            active: params?.active,
            exclude_debts: params?.exclude_debts,
        })}`)
        return response.data
    },

    reorder: (ids: number[]) =>
        api.post<{ success: boolean }, { ids: number[] }>('/accounts/reorder', { ids }),

    getBalanceHistory: async (params?: { start_date?: string; end_date?: string }): Promise<BalanceHistoryResponse> => {
        const response = await apiClient.get(`/accounts-balance-history${toQueryString(params)}`)
        return response.data
    },

    getBalanceComparison: async (params?: { start_date?: string; end_date?: string }): Promise<BalanceComparisonResponse> => {
        const response = await apiClient.get(`/accounts-balance-comparison${toQueryString(params)}`)
        return response.data
    },
}
