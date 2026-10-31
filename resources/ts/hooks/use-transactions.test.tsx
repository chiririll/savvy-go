// @vitest-environment jsdom
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { transactionsApi } from '@/api'
import { useInfiniteTransactions, useUpdateTransaction } from './use-transactions'

vi.mock('@/api', () => ({
    transactionsApi: { getAll: vi.fn(), update: vi.fn() },
}))

const page = {
    data: [{ id: 1 }, { id: 2 }],
    meta: { current_page: 1, last_page: 1, total: 2 },
}

beforeEach(() => vi.clearAllMocks())

describe('useUpdateTransaction', () => {
    it('keeps the loaded list on screen while it refetches after an edit', async () => {
        vi.mocked(transactionsApi.getAll)
            .mockResolvedValueOnce(page as never)
            .mockImplementation(() => new Promise((resolve) => setTimeout(() => resolve(page as never), 50)))
        vi.mocked(transactionsApi.update).mockResolvedValue({ id: 1 } as never)
        const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
        const wrapper = ({ children }: { children: ReactNode }) => (
            <MemoryRouter>
                <QueryClientProvider client={client}>{children}</QueryClientProvider>
            </MemoryRouter>
        )

        const seen: number[] = []
        const { result } = renderHook(
            () => {
                const list = useInfiniteTransactions()
                seen.push(list.transactions.length)
                return { list, update: useUpdateTransaction() }
            },
            { wrapper },
        )
        await waitFor(() => expect(result.current.list.transactions).toHaveLength(2))
        seen.length = 0

        await act(async () => {
            await result.current.update.mutateAsync({ id: 1, data: {} })
        })
        await waitFor(() => expect(transactionsApi.getAll).toHaveBeenCalledTimes(2))
        await waitFor(() => expect(result.current.list.isFetching).toBe(false))

        expect(seen).not.toContain(0)
    })

    it('shows the edited transaction once the list has refetched', async () => {
        vi.mocked(transactionsApi.getAll)
            .mockResolvedValueOnce({ ...page, data: [{ id: 1, description: 'old' }] } as never)
            .mockResolvedValue({ ...page, data: [{ id: 1, description: 'new' }] } as never)
        vi.mocked(transactionsApi.update).mockResolvedValue({ id: 1 } as never)
        const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 5 * 60 * 1000 } } })
        const wrapper = ({ children }: { children: ReactNode }) => (
            <MemoryRouter>
                <QueryClientProvider client={client}>{children}</QueryClientProvider>
            </MemoryRouter>
        )

        const { result } = renderHook(
            () => ({ list: useInfiniteTransactions(), update: useUpdateTransaction() }),
            { wrapper },
        )
        await waitFor(() => expect(result.current.list.transactions).toHaveLength(1))

        await act(async () => {
            await result.current.update.mutateAsync({ id: 1, data: {} })
        })

        await waitFor(() =>
            expect((result.current.list.transactions[0] as { description?: string }).description).toBe('new'),
        )
    })

    it('puts the saved transaction into the list without waiting for the refetch', async () => {
        vi.mocked(transactionsApi.getAll)
            .mockResolvedValueOnce({ ...page, data: [{ id: 1, description: 'old' }] } as never)
            .mockImplementation(() => new Promise(() => {}))
        vi.mocked(transactionsApi.update).mockResolvedValue({ id: 1, description: 'new' } as never)
        const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 5 * 60 * 1000 } } })
        const wrapper = ({ children }: { children: ReactNode }) => (
            <MemoryRouter>
                <QueryClientProvider client={client}>{children}</QueryClientProvider>
            </MemoryRouter>
        )

        const { result } = renderHook(
            () => ({ list: useInfiniteTransactions(), update: useUpdateTransaction() }),
            { wrapper },
        )
        await waitFor(() => expect(result.current.list.transactions).toHaveLength(1))

        await act(async () => {
            await result.current.update.mutateAsync({ id: 1, data: {} })
        })

        await waitFor(() =>
            expect((result.current.list.transactions[0] as { description?: string }).description).toBe('new'),
        )
    })
})
