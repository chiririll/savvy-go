import { permissionLevel, type ApiResource, type ApiTokenSummary } from '@/types'

export type Expiry = 'never' | '30' | '90' | '365'

export interface GrantedResources {
    read: ApiResource[]
    write: ApiResource[]
    remove: ApiResource[]
}

/** What a token holds, as lists of resources by their highest level. */
export function grantedResources(token: Pick<ApiTokenSummary, 'permissions'>): GrantedResources {
    const granted: GrantedResources = { read: [], write: [], remove: [] }
    const resources = new Set(token.permissions.map((p) => p.split(':')[0]))
    for (const resource of resources) {
        const level = permissionLevel(token.permissions, resource)
        if (level === 'delete') granted.remove.push(resource)
        else if (level === 'write') granted.write.push(resource)
        else if (level === 'read') granted.read.push(resource)
    }
    return granted
}

export function expiryToIso(expiry: Expiry, now = new Date()): string | null {
    if (expiry === 'never') return null
    const date = new Date(now)
    date.setDate(date.getDate() + Number(expiry))
    return date.toISOString()
}
