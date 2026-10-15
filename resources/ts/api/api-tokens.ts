import { apiClient } from './client'
import type { ApiTokenSummary, CreatedApiToken, CreateApiTokenInput, TokenCapabilities, UpdateApiTokenInput } from '@/types'

export const apiTokensApi = {
    /** What each kind of token can be given. */
    capabilities: async (): Promise<TokenCapabilities> => {
        const { data } = await apiClient.get('/auth/api-token-capabilities')
        return data.data
    },

    list: async (): Promise<ApiTokenSummary[]> => {
        const { data } = await apiClient.get('/auth/api-tokens')
        return data.data
    },

    create: async (input: CreateApiTokenInput): Promise<CreatedApiToken> => {
        const { data } = await apiClient.post('/auth/api-tokens', input)
        return data.data
    },

    update: async (id: number, input: UpdateApiTokenInput): Promise<ApiTokenSummary> => {
        const { data } = await apiClient.patch(`/auth/api-tokens/${id}`, input)
        return data.data
    },

    revoke: async (id: number): Promise<void> => {
        await apiClient.delete(`/auth/api-tokens/${id}`)
    },
}
