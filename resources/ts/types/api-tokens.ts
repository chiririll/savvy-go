// API token (external app connection) types
export type ApiTokenScope = 'read' | 'read-write'

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
