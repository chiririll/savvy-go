import { apiClient } from './client'
import type { ApiTokenSummary, CreatedApiToken, CreateApiTokenInput } from '@/types'

export const apiTokensApi = {
    list: async (): Promise<ApiTokenSummary[]> => {
        const { data } = await apiClient.get('/auth/api-tokens')
        return data.data
    },

    create: async (input: CreateApiTokenInput): Promise<CreatedApiToken> => {
        const { data } = await apiClient.post('/auth/api-tokens', input)
        return data.data
    },

    revoke: async (id: number): Promise<void> => {
        await apiClient.delete(`/auth/api-tokens/${id}`)
    },
}
