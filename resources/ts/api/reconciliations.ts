import { api } from './client'
import { toQueryString } from '@/lib/query-string'
import type {
    ReconciliationDay,
    ReconciliationFilters,
    ReconciliationInput,
    ReconciliationPoint,
} from '@/types'

interface Period {
    start_date?: string
    end_date?: string
}

export const reconciliationsApi = {
    getAll: ({ account_ids, ...period }: ReconciliationFilters = {}) =>
        api.get<ReconciliationPoint[]>(
            `/reconciliations${toQueryString({ ...period, account_ids: account_ids?.join(',') })}`,
        ),

    getForAccount: (accountId: number | string, period?: Period) =>
        api.get<ReconciliationPoint[]>(`/accounts/${accountId}/reconciliations${toQueryString({ ...period })}`),

    /** Pins every item's account on the date; all or none is stored. Answers the points in the order of the items. */
    create: (input: ReconciliationInput) => api.post<ReconciliationPoint[], ReconciliationInput>('/reconciliations', input),

    delete: (id: number | string) => api.delete<void>(`/reconciliations/${id}`),

    getDays: (period: { start_date: string; end_date: string }) =>
        api.get<ReconciliationDay[]>(`/reconciliations-days${toQueryString(period)}`),
}
