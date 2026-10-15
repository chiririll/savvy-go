// API token (external app connection) types

/**
 * A part of a space a token can be granted (`transactions`, `budgets`, ...). The
 * server decides which parts each kind of token can hold: see `TokenCapabilities`.
 */
export type ApiResource = string

/** Each level includes the ones before it: write includes read, delete includes write. */
export const API_PERMISSION_LEVELS = ['read', 'write', 'delete'] as const
export type ApiPermissionLevel = (typeof API_PERMISSION_LEVELS)[number]

/** A grant as the backend sends it: `transactions:read`. */
export type ApiPermission = `${ApiResource}:${ApiPermissionLevel}`

/** `personal` is made in Settings; `mcp` is the token of a connected AI app. Fixed when the token is issued. */
export type ApiTokenKind = 'personal' | 'mcp'

export interface ApiTokenSummary {
    id: number
    name: string
    prefix: string
    kind: ApiTokenKind
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

/** A ready-made set of permissions, named after a space role (`viewer`, `editor`, `admin`). */
export type ApiTokenPreset = string

/** What one kind of token can be given, from `GET /auth/api-token-capabilities`. */
export interface KindCapabilities {
    resources: ApiResource[]
    /** Grants per preset, over `resources`. */
    presets: Record<ApiTokenPreset, ApiPermission[]>
}

export interface TokenCapabilities {
    levels: ApiPermissionLevel[]
    /** Preset names, the order a screen offers them. */
    presets: ApiTokenPreset[]
    kinds: Record<ApiTokenKind, KindCapabilities>
}

/** One kind of token as a screen offers it: its parts, and its presets in order. */
export interface KindOptions {
    resources: ApiResource[]
    presets: { name: ApiTokenPreset; permissions: ApiPermission[] }[]
}

export function kindOptions(caps: TokenCapabilities, kind: ApiTokenKind): KindOptions {
    const k = caps.kinds[kind]
    return { resources: k.resources, presets: caps.presets.map((name) => ({ name, permissions: [...(k.presets[name] ?? [])].sort() })) }
}

/** New rights for an existing token. The secret stays the same. */
export interface UpdateApiTokenInput {
    permissions: ApiPermission[]
    all_spaces: boolean
    space_ids: number[]
}

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

/** Whether two lists hold the same grant on every one of these resources. */
export function samePermissions(a: readonly string[], b: readonly string[], resources: readonly ApiResource[]): boolean {
    return resources.every((r) => permissionLevel(a, r) === permissionLevel(b, r))
}

/** Highest level a token holds on a resource, or null. */
export function permissionLevel(permissions: readonly string[], resource: ApiResource): ApiPermissionLevel | null {
    if (permissions.includes(`${resource}:delete`)) return 'delete'
    if (permissions.includes(`${resource}:write`)) return 'write'
    if (permissions.includes(`${resource}:read`)) return 'read'
    return null
}
