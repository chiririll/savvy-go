import { apiClient } from './client'
import type { ApiPermission, ConfirmFields } from '@/types'

export interface OAuthRequest {
    client_id: string
    redirect_uri: string
    state: string
    code_challenge: string
    code_challenge_method: string
}

export interface OAuthConsentInfo {
    client_name: string
    /** The parts of a space the app can be given (it only reads them), as in API token permissions. */
    resources: string[]
    spaces: Array<{ id: number; name: string; role: string }>
}

export const oauthApi = {
    /** The app that is asking, what it will read, and which of your spaces it can be given.
     * Errors when the client or redirect URI is not registered. */
    info: async (clientId: string, redirectUri: string): Promise<OAuthConsentInfo> => {
        const { data } = await apiClient.get('/oauth/authorize', {
            params: { client_id: clientId, redirect_uri: redirectUri },
        })
        return data.data
    },

    /** Records the decision; returns where to send the browser (the app's redirect URI).
     * An approval names the spaces the app may reach and what it may do there, and carries the
     * password or code that confirms it. */
    decide: async (
        request: OAuthRequest,
        approve: boolean,
        spaces: { all: boolean; ids: number[] },
        permissions: ApiPermission[],
        confirm: ConfirmFields = {},
    ): Promise<string> => {
        const { data } = await apiClient.post('/oauth/authorize', {
            ...request,
            approve,
            all_spaces: spaces.all,
            space_ids: spaces.ids,
            permissions,
            ...confirm,
        })
        return data.data.redirect_url
    },
}

export interface OAuthRegistration {
    open: boolean
    expires_at: string | null
}

/** Admins open a short window in which one new AI app may register itself. */
export const oauthRegistrationApi = {
    get: async (): Promise<OAuthRegistration> => (await apiClient.get('/admin/oauth/registration')).data.data,
    open: async (): Promise<OAuthRegistration> => (await apiClient.post('/admin/oauth/registration')).data.data,
    close: async (): Promise<OAuthRegistration> => (await apiClient.delete('/admin/oauth/registration')).data.data,
}

/** An app that registered itself, with what the server saw of its registration. */
export interface OAuthClient {
    client_id: string
    name: string
    redirect_uris: string[]
    /** What the server saw of the registration request. */
    ip: string
    user_agent: string
    origin: string
    registered_at: string
}

export const oauthClientsApi = {
    list: async (): Promise<OAuthClient[]> => (await apiClient.get('/admin/oauth/clients')).data.data,
    /** Disconnects an app: its tokens stop working at once. */
    remove: async (id: string): Promise<void> => {
        await apiClient.delete(`/admin/oauth/clients/${encodeURIComponent(id)}`)
    },
}
