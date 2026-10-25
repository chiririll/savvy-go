// @vitest-environment jsdom
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { oauthClientsApi, oauthRegistrationApi, type OAuthClient } from '@/api/oauth'
import { useOAuthClients, useOAuthRegistration } from './use-oauth-registration'

vi.mock('@/api/oauth', () => ({
    oauthClientsApi: { list: vi.fn(), remove: vi.fn() },
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
        vi.mocked(oauthClientsApi.list).mockResolvedValueOnce([app('a')]).mockResolvedValue([app('a'), app('b')])
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
