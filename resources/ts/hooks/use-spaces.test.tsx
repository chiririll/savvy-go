// @vitest-environment jsdom
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { spacesApi } from '@/api/spaces'
import { useServerCurrencyCatalog } from './use-spaces'

vi.mock('@/api/spaces', () => ({
    spacesApi: { currencyCatalog: vi.fn() },
    adminSpacesApi: {},
    invitationsApi: {},
    syncGroupsApi: {},
}))

const result = (source: string) => ({ items: [], source })

function wrapper() {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    return ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

beforeEach(() => vi.clearAllMocks())

describe('useServerCurrencyCatalog', () => {
    it('asks for a retry only on a manual refetch, not on the first load', async () => {
        vi.mocked(spacesApi.currencyCatalog)
            .mockResolvedValueOnce(result('fallback'))
            .mockResolvedValue(result('live'))

        const { result: hook } = renderHook(() => useServerCurrencyCatalog(), { wrapper: wrapper() })
        await waitFor(() => expect(hook.current.data?.source).toBe('fallback'))
        expect(vi.mocked(spacesApi.currencyCatalog).mock.calls[0]).toEqual([undefined, { refresh: false }])

        await act(async () => {
            await hook.current.refetch()
        })
        await waitFor(() => expect(hook.current.data?.source).toBe('live'))
        expect(vi.mocked(spacesApi.currencyCatalog).mock.calls[1]).toEqual([undefined, { refresh: true }])
    })

    it('sends the flag on every manual refetch', async () => {
        vi.mocked(spacesApi.currencyCatalog).mockResolvedValue(result('live'))
        const { result: hook } = renderHook(() => useServerCurrencyCatalog(), { wrapper: wrapper() })
        await waitFor(() => expect(hook.current.data).toBeDefined())

        await act(async () => {
            await hook.current.refetch()
        })
        await act(async () => {
            await hook.current.refetch()
        })
        const calls = vi.mocked(spacesApi.currencyCatalog).mock.calls
        expect(calls.map((c) => c[1]?.refresh)).toEqual([false, true, true])
    })
})
