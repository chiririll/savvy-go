import { permissionLevel, type ApiPermission, type ApiResource } from '@/types'
import type { OAuthRequest } from '@/api/oauth'

/** The authorization request in the URL, or null when a required part is missing. */
export function parseOAuthRequest(params: URLSearchParams): OAuthRequest | null {
    const get = (key: string) => params.get(key) ?? ''
    const request = {
        client_id: get('client_id'),
        redirect_uri: get('redirect_uri'),
        state: get('state'),
        code_challenge: get('code_challenge'),
        code_challenge_method: get('code_challenge_method'),
    }
    return request.client_id && request.redirect_uri && request.code_challenge ? request : null
}

/** Where the browser goes back to, shown so the user can check it; empty if the address is not a URL. */
export function redirectHost(redirectUri: string | undefined): string {
    try {
        return new URL(redirectUri ?? '').host
    } catch {
        return ''
    }
}

/**
 * Resources the app would be able to change, and those it could delete from.
 * Changing data is never the default, so these are what the user confirms once more.
 */
export function changedResources(resources: ApiResource[], permissions: ApiPermission[]) {
    return {
        changes: resources.filter((r) => {
            const level = permissionLevel(permissions, r)
            return level !== null && level !== 'read'
        }),
        deletes: resources.filter((r) => permissionLevel(permissions, r) === 'delete'),
    }
}
