import type { InternalAxiosRequestConfig } from 'axios'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { apiClient } from './client'
import { reconciliationsApi } from './reconciliations'

interface Call { method?: string; url?: string; body: unknown }

let calls: Call[] = []
const adapter = apiClient.defaults.adapter

beforeEach(() => {
    calls = []
    apiClient.defaults.adapter = async (config: InternalAxiosRequestConfig) => {
        calls.push({ method: config.method, url: config.url, body: config.data ? JSON.parse(config.data) : undefined })
        return { data: { data: [] }, status: 200, statusText: 'OK', headers: {}, config }
    }
})

afterEach(() => {
    apiClient.defaults.adapter = adapter
})

describe('reconciliationsApi', () => {
    it('posts one batch', async () => {
        const input = { date: '2026-11-08', items: [{ accountId: 1, actualBalance: 10.5 }] }
        await reconciliationsApi.create(input)
        expect(calls).toEqual([{ method: 'post', url: '/reconciliations', body: input }])
    })

    it('lists with comma-separated accounts and a period', async () => {
        await reconciliationsApi.getAll({ account_ids: [1, 2], start_date: '2026-11-01', end_date: '2026-11-08' })
        expect(calls[0].method).toBe('get')
        expect(calls[0].url).toBe('/reconciliations?start_date=2026-11-01&end_date=2026-11-08&account_ids=1%2C2')
    })

    it('lists the points of an account, deletes one and reads the days', async () => {
        await reconciliationsApi.getForAccount(3)
        await reconciliationsApi.delete(7)
        await reconciliationsApi.getDays({ start_date: '2026-11-01', end_date: '2026-11-30' })
        expect(calls.map((c) => [c.method, c.url])).toEqual([
            ['get', '/accounts/3/reconciliations'],
            ['delete', '/reconciliations/7'],
            ['get', '/reconciliations-days?start_date=2026-11-01&end_date=2026-11-30'],
        ])
    })
})
