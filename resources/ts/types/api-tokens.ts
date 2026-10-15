// API token (external app connection) types

/** Parts of a space a token can be granted. Mirrors auth.APIResources in Go. */
export const API_RESOURCES = [
    'accounts',
    'automation',
    'budgets',
    'categories',
    'currencies',
    'debts',
    'recurring',
    'reports',
    'space',
    'tags',
    'transactions',
    'transfers',
] as const
export type ApiResource = (typeof API_RESOURCES)[number]

/** Each level includes the ones before it: write includes read, delete includes write. */
export const API_PERMISSION_LEVELS = ['read', 'write', 'delete'] as const
export type ApiPermissionLevel = (typeof API_PERMISSION_LEVELS)[number]

/** A grant as the backend sends it: `transactions:read`. */
export type ApiPermission = `${ApiResource}:${ApiPermissionLevel}`

export interface ApiTokenSummary {
    id: number
    name: string
    prefix: string
    permissions: ApiPermission[]
    /** Reaches every space of the owner, including future ones; `space_ids` is then empty. */
    all_spaces: boolean
    space_ids: number[]
    expires_at: string | null
    last_used_at: string | null
    created_at: string | null
}

export interface CreatedApiToken extends ApiTokenSummary {
    token: string
}

/** Ready-made sets of permissions, named after space roles. Mirrors auth.Preset* in Go. */
export const API_TOKEN_PRESETS = ['viewer', 'editor', 'admin'] as const
export type ApiTokenPreset = (typeof API_TOKEN_PRESETS)[number]

/** A preset, or a list of permissions; never both. */
export interface CreateApiTokenInput {
    name: string
    preset?: ApiTokenPreset
    /** Lets an editor delete transactions as well. */
    allow_delete?: boolean
    permissions?: ApiPermission[]
    all_spaces: boolean
    space_ids: number[]
    expires_at: string | null
}

/**
 * What a preset grants over some resources (all of them by default). Mirrors
 * auth.PresetPermissions in Go; the editor's extra write is on transactions.
 * Returns sorted grants, so two sets can be compared with `samePermissions`.
 */
export function presetPermissions(preset: ApiTokenPreset, resources: readonly ApiResource[] = API_RESOURCES): ApiPermission[] {
    const level: ApiPermissionLevel = preset === 'admin' ? 'delete' : 'read'
    return resources
        .map((r) => `${r}:${preset === 'editor' && r === 'transactions' ? 'write' : level}` as ApiPermission)
        .sort()
}

/** Whether two lists hold the same grant on every resource. */
export function samePermissions(a: readonly string[], b: readonly string[]): boolean {
    return API_RESOURCES.every((r) => permissionLevel(a, r) === permissionLevel(b, r))
}

/** Highest level a token holds on a resource, or null. */
export function permissionLevel(permissions: readonly string[], resource: ApiResource): ApiPermissionLevel | null {
    if (permissions.includes(`${resource}:delete`)) return 'delete'
    if (permissions.includes(`${resource}:write`)) return 'write'
    if (permissions.includes(`${resource}:read`)) return 'read'
    return null
}
