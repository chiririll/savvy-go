import { apiClient } from './client'
import type { ApiTokenSummary, ConfirmFields, CreatedApiToken, CreateApiTokenInput, TokenCapabilities, UpdateApiTokenInput } from '@/types'

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

    create: async (input: CreateApiTokenInput, confirm: ConfirmFields = {}): Promise<CreatedApiToken> => {
        const { data } = await apiClient.post('/auth/api-tokens', { ...input, ...confirm })
        return data.data
    },

    update: async (id: number, input: UpdateApiTokenInput, confirm: ConfirmFields = {}): Promise<ApiTokenSummary> => {
        const { data } = await apiClient.patch(`/auth/api-tokens/${id}`, { ...input, ...confirm })
        return data.data
    },

    revoke: async (id: number): Promise<void> => {
        await apiClient.delete(`/auth/api-tokens/${id}`)
    },
}
