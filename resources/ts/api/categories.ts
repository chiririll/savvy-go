import { api, apiClient } from './client'
import { createCrudApi } from './crud'
import { toQueryString } from '@/lib/query-string'
import { Category, CategorySummaryResponse } from '@/types'
import { CategoryFormData } from '@/schemas'

const crud = createCrudApi<Category, CategoryFormData>('/categories')

export const categoriesApi = {
    ...crud,

    delete: (id: number | string, successorId?: number | string) =>
        api.delete<void>(`/categories/${id}${toQueryString({ successor_id: successorId })}`),

    setDefault: (id: number | string) =>
        api.post<Category>(`/categories/${id}/set-default`),

    getSummary: async (params: {
        type: 'income' | 'expense'
        start_date?: string
        end_date?: string
    }): Promise<CategorySummaryResponse> => {
        const response = await apiClient.get(`/categories-summary${toQueryString(params)}`)
        return response.data
    },
}
