// @vitest-environment jsdom
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { oauthClientsApi, oauthHostsApi, oauthRegistrationApi, type OAuthClient } from '@/api/oauth'
import {
    useAddOAuthHost,
    useCreateOAuthClient,
    useOAuthClients,
    useOAuthHosts,
    useOAuthRegistration,
    useRemoveOAuthHost,
} from './use-oauth-registration'

vi.mock('@/api/oauth', () => ({
    oauthClientsApi: { list: vi.fn(), remove: vi.fn(), create: vi.fn() },
    oauthHostsApi: { list: vi.fn(), add: vi.fn(), remove: vi.fn() },
    oauthRegistrationApi: { get: vi.fn(), open: vi.fn(), close: vi.fn() },
}))

const app = (id: string): OAuthClient => ({
    client_id: id,
    name: id,
    redirect_uris: [],
    ip: '',
    user_agent: '',
    origin: '',
    registered_at: '2026-01-01T00:00:00Z',
    kind: 'dcr',
    confidential: false,
})

/** Same defaults as the app: data stays fresh for five minutes. */
function makeClient() {
    return new QueryClient({
        defaultOptions: { queries: { staleTime: 5 * 60 * 1000, refetchOnWindowFocus: false, retry: false } },
    })
}

function wrapperFor(client: QueryClient) {
    return ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
    )
}

beforeEach(() => vi.clearAllMocks())

describe('useOAuthClients', () => {
    it('refetches on remount even when cached data is still fresh', async () => {
        vi.mocked(oauthClientsApi.list)
            .mockResolvedValueOnce([app('a')])
            .mockResolvedValue([app('a'), app('b')])
        const client = makeClient()

        const first = renderHook(() => useOAuthClients(), { wrapper: wrapperFor(client) })
        await waitFor(() => expect(first.result.current.data).toHaveLength(1))
        first.unmount()

        const second = renderHook(() => useOAuthClients(), { wrapper: wrapperFor(client) })
        await waitFor(() => expect(second.result.current.data).toHaveLength(2))
        expect(oauthClientsApi.list).toHaveBeenCalledTimes(2)
    })
})

describe('useOAuthRegistration', () => {
    it('refetches on remount even when cached data is still fresh', async () => {
        vi.mocked(oauthRegistrationApi.get)
            .mockResolvedValueOnce({ open: true, expires_at: null })
            .mockResolvedValue({ open: false, expires_at: null })
        const client = makeClient()

        const first = renderHook(() => useOAuthRegistration(), { wrapper: wrapperFor(client) })
        await waitFor(() => expect(first.result.current.data?.open).toBe(true))
        first.unmount()

        const second = renderHook(() => useOAuthRegistration(), { wrapper: wrapperFor(client) })
        await waitFor(() => expect(second.result.current.data?.open).toBe(false))
        expect(oauthRegistrationApi.get).toHaveBeenCalledTimes(2)
    })
})

describe('OAuth metadata document hosts', () => {
    it('reads the list again after a host is added and removed', async () => {
        vi.mocked(oauthHostsApi.list)
            .mockResolvedValueOnce([])
            .mockResolvedValueOnce(['claude.ai'])
            .mockResolvedValue([])
        vi.mocked(oauthHostsApi.add).mockResolvedValue()
        vi.mocked(oauthHostsApi.remove).mockResolvedValue()
        const wrapper = wrapperFor(makeClient())

        const hosts = renderHook(() => useOAuthHosts(), { wrapper })
        const add = renderHook(() => useAddOAuthHost(), { wrapper })
        const remove = renderHook(() => useRemoveOAuthHost(), { wrapper })
        await waitFor(() => expect(hosts.result.current.data).toEqual([]))

        await act(() => add.result.current.mutateAsync('claude.ai'))
        await waitFor(() => expect(hosts.result.current.data).toEqual(['claude.ai']))
        expect(oauthHostsApi.add).toHaveBeenCalledWith('claude.ai', expect.anything())

        await act(() => remove.result.current.mutateAsync('claude.ai'))
        await waitFor(() => expect(hosts.result.current.data).toEqual([]))
        expect(oauthHostsApi.remove).toHaveBeenCalledWith('claude.ai', expect.anything())
    })
})

describe('useCreateOAuthClient', () => {
    it('returns the secret once and refreshes the app list', async () => {
        vi.mocked(oauthClientsApi.list)
            .mockResolvedValueOnce([])
            .mockResolvedValue([{ ...app('new'), kind: 'manual', confidential: true }])
        vi.mocked(oauthClientsApi.create).mockResolvedValue({
            ...app('new'),
            kind: 'manual',
            confidential: true,
            client_secret: 's3cret',
        })
        const wrapper = wrapperFor(makeClient())

        const list = renderHook(() => useOAuthClients(), { wrapper })
        const create = renderHook(() => useCreateOAuthClient(), { wrapper })
        await waitFor(() => expect(list.result.current.data).toEqual([]))

        const body = { name: 'Mine', redirect_uris: ['https://x.example/cb'], confidential: true }
        const created = await act(() => create.result.current.mutateAsync(body))

        expect(oauthClientsApi.create).toHaveBeenCalledWith(body, expect.anything())
        expect(created.client_secret).toBe('s3cret')
        await waitFor(() => expect(list.result.current.data).toHaveLength(1))
    })
})
