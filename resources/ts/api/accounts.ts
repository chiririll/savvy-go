import { api, apiClient } from './client'
import { createCrudApi } from './crud'
import { toQueryString } from '@/lib/api/query-string'
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
const cards = createCrudApi<Account, Partial<AccountFormData>>('/credit-cards')

/** The body of a card request: the type and the client-only flag are not the API's. */
function cardBody({ type: _type, convert: _convert, ...body }: Partial<AccountFormData>) {
    return body
}

export const accountsApi = {
    ...crud,

    /** A card is created on its own endpoint. */
    create: (data: AccountFormData) => (data.type === 'credit' ? cards.create(cardBody(data)) : crud.create(data)),

    /**
     * A card is saved on `/credit-cards`. Turning an existing account into one (`convert`) posts its id with the
     * terms; the API keeps the account's name, currency and balance.
     */
    update: (id: number | string, data: Partial<AccountFormData>) => {
        if (data.type !== 'credit') {
            return crud.update(id, data)
        }
        if (data.convert) {
            const { limit, statement_day, grace_days } = data
            return cards.create({ account_id: Number(id), limit, statement_day, grace_days } as Partial<AccountFormData>)
        }
        return cards.update(id, cardBody(data))
    },

    getAll: (params?: { active?: boolean; exclude_debts?: boolean }) => crud.getAll(params),

    getAllWithSummary: async (params?: { active?: boolean; exclude_debts?: boolean }): Promise<AccountsResponse> => {
        const response = await apiClient.get(
            `/accounts${toQueryString({
                with_summary: true,
                active: params?.active,
                exclude_debts: params?.exclude_debts,
            })}`,
        )
        return response.data
    },

    reorder: (ids: number[]) => api.post<{ success: boolean }, { ids: number[] }>('/accounts/reorder', { ids }),

    getBalanceHistory: async (params?: { start_date?: string; end_date?: string; account_ids?: number[] }): Promise<BalanceHistoryResponse> => {
        const response = await apiClient.get(`/accounts-balance-history${toQueryString(params)}`)
        return response.data
    },

    getBalanceComparison: async (params?: {
        start_date?: string
        end_date?: string
    }): Promise<BalanceComparisonResponse> => {
        const response = await apiClient.get(`/accounts-balance-comparison${toQueryString(params)}`)
        return response.data
    },
}
