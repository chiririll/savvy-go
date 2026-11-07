import { api, apiClient } from './client'
import { createCrudApi } from './crud'
import { toQueryString } from '@/lib/query-string'
import type {
    LineInput,
    PaymentInput,
    Transaction,
    TransactionFilters,
    TransactionInput,
    TransactionSummary,
} from '@/types'

const ENDPOINT = '/transactions'

/** Confirming the open occurrence of a recurring rule may bring the real payments and lines. */
export interface ConfirmTransactionOptions {
    date?: string | null
    payments?: PaymentInput[]
    lines?: LineInput[]
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
        /** Only when asked with `count: false`: `total` is then just the page. */
        has_more?: boolean
        from: number
        to: number
    }
}

const crud = createCrudApi<Transaction, TransactionInput>(ENDPOINT)

export const transactionsApi = {
    ...crud,

    getAll: async (
        filters?: TransactionFilters & { with_summary?: boolean; count?: boolean },
        signal?: AbortSignal,
    ): Promise<TransactionsResponse> => {
        const query = toQueryString({
            type: filters?.type,
            account_id: filters?.account_id,
            category_id: filters?.category_id,
            account_ids: filters?.account_ids?.join(','),
            currency: filters?.currency?.join(','),
            q: filters?.q,
            category_ids: filters?.category_ids,
            tag_ids: filters?.tag_ids,
            start_date: filters?.start_date,
            end_date: filters?.end_date,
            sort_by: filters?.sort_by,
            sort_direction: filters?.sort_direction,
            per_page: filters?.per_page,
            page: filters?.page,
            with_summary: filters?.with_summary,
            count: filters?.count === false ? '0' : undefined,
            status: filters?.status,
        })
        const response = await apiClient.get(`${ENDPOINT}${query}`, { signal })
        return response.data
    },

    duplicate: (id: number | string) =>
        api.post<Transaction, void>(`${ENDPOINT}/${id}/duplicate`, undefined),

    confirm: (id: number | string, options: ConfirmTransactionOptions = {}) =>
        api.post<Transaction, ConfirmTransactionBody>(`${ENDPOINT}/${id}/confirm`, {
            ...(options.date ? { date: options.date } : {}),
            ...(options.payments ? { payments: options.payments } : {}),
            ...(options.lines ? { lines: options.lines } : {}),
        }),

    /** Puts the transactions in that order within one day; ids go oldest first. */
    reorder: (date: string, ids: number[]) =>
        api.post<{ success: boolean }, { date: string; ids: number[] }>(`${ENDPOINT}/reorder`, { date, ids }),

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
