import { api } from './client'
import { toQueryString, type QueryParamValue } from '@/lib/api/query-string'

export function createCrudApi<T, TForm>(endpoint: string) {
    return {
        getAll: (params?: Record<string, QueryParamValue>) =>
            api.get<T[]>(`${endpoint}${toQueryString(params)}`),

        getById: (id: number | string) =>
            api.get<T>(`${endpoint}/${id}`),

        create: (data: TForm) =>
            api.post<T, TForm>(endpoint, data),

        update: (id: number | string, data: Partial<TForm>) =>
            api.patch<T, Partial<TForm>>(`${endpoint}/${id}`, data),

        delete: (id: number | string) =>
            api.delete<void>(`${endpoint}/${id}`),
    }
}
