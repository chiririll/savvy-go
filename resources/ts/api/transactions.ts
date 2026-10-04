import { api, apiClient } from './client'
import { createCrudApi } from './crud'
import { toQueryString } from '@/lib/query-string'
import { Transaction, TransactionFilters, TransactionSummary } from '@/types'
import { TransactionFormData } from '@/schemas'

const ENDPOINT = '/transactions'

export interface ConfirmTransactionOptions {
    date?: string | null
    amount?: number | null
    to_amount?: number | null
}

type ConfirmTransactionBody = Omit<ConfirmTransactionOptions, 'date'> & { date?: string }

export interface TransactionsResponse {
    data: Transaction[]
    summary?: TransactionSummary
    meta?: {
        current_page: number
        last_page: number
        per_page: number
        total: number
        from: number
        to: number
    }
}

const crud = createCrudApi<Transaction, TransactionFormData>(ENDPOINT)

export const transactionsApi = {
    ...crud,

    getAll: async (filters?: TransactionFilters & { with_summary?: boolean }): Promise<TransactionsResponse> => {
        const query = toQueryString({
            type: filters?.type,
            account_id: filters?.account_id,
            category_id: filters?.category_id,
            category_ids: filters?.category_ids,
            tag_ids: filters?.tag_ids,
            start_date: filters?.start_date,
            end_date: filters?.end_date,
            sort_by: filters?.sort_by,
            sort_direction: filters?.sort_direction,
            per_page: filters?.per_page,
            page: filters?.page,
            with_summary: filters?.with_summary,
            status: filters?.status,
        })
        const response = await apiClient.get(`${ENDPOINT}${query}`)
        return response.data
    },

    duplicate: (id: number | string) =>
        api.post<Transaction, void>(`${ENDPOINT}/${id}/duplicate`, undefined),

    confirm: (id: number | string, options: ConfirmTransactionOptions = {}) =>
        api.post<Transaction, ConfirmTransactionBody>(`${ENDPOINT}/${id}/confirm`, {
            ...(options.date ? { date: options.date } : {}),
            ...(options.amount != null ? { amount: options.amount } : {}),
            ...(options.to_amount != null ? { to_amount: options.to_amount } : {}),
        }),

    getPendingSummary: () =>
        api.get<TransactionSummary>('/transactions-pending-summary'),

    skip: (id: number | string) =>
        api.post<Transaction>(`${ENDPOINT}/${id}/skip`),

    getSummary: (filters?: TransactionFilters) =>
        api.get<TransactionSummary>(`/transactions-summary${toQueryString({
            type: filters?.type,
            account_id: filters?.account_id,
            category_id: filters?.category_id,
            start_date: filters?.start_date,
            end_date: filters?.end_date,
        })}`),
}
