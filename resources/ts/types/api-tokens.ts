// API token (external app connection) types
export const API_TOKEN_SCOPES = ['read', 'read-write'] as const
export type ApiTokenScope = (typeof API_TOKEN_SCOPES)[number]

export interface ApiTokenSummary {
    id: number
    name: string
    prefix: string
    scope: ApiTokenScope
    expires_at: string | null
    last_used_at: string | null
    created_at: string | null
}

export interface CreatedApiToken extends ApiTokenSummary {
    token: string
}

export interface CreateApiTokenInput {
    name: string
    scope: ApiTokenScope
    expires_at: string | null
}
