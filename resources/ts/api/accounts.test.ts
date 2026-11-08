import type { InternalAxiosRequestConfig } from 'axios'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import type { AccountFormData } from '@/schemas'
import { accountsApi } from './accounts'
import { apiClient } from './client'

interface Call { method?: string; url?: string; body: unknown }

let calls: Call[] = []
const adapter = apiClient.defaults.adapter

beforeEach(() => {
    calls = []
    apiClient.defaults.adapter = async (config: InternalAxiosRequestConfig) => {
        calls.push({ method: config.method, url: config.url, body: config.data ? JSON.parse(config.data) : undefined })
        return { data: { data: {} }, status: 200, statusText: 'OK', headers: {}, config }
    }
})

afterEach(() => {
    apiClient.defaults.adapter = adapter
})

const terms = { limit: 2000, statement_day: 28, grace_days: 21 }
const base: AccountFormData = { name: 'Visa', type: 'credit', currency_id: 1, initial_balance: 0, is_active: true, ...terms }

describe('accountsApi card routing', () => {
    it('creates a card on /credit-cards', async () => {
        await accountsApi.create(base)
        expect(calls).toHaveLength(1)
        expect(calls[0]).toMatchObject({ method: 'post', url: '/credit-cards' })
        expect(calls[0].body).toMatchObject({ name: 'Visa', currency_id: 1, ...terms })
        expect(calls[0].body).not.toHaveProperty('type')
    })

    it('creates other accounts on /accounts', async () => {
        await accountsApi.create({ ...base, type: 'bank' })
        expect(calls[0]).toMatchObject({ method: 'post', url: '/accounts' })
    })

    it('updates a card on /credit-cards/{id}', async () => {
        await accountsApi.update(5, base)
        expect(calls[0]).toMatchObject({ method: 'patch', url: '/credit-cards/5' })
        expect(calls[0].body).not.toHaveProperty('convert')
    })

    it('converts an existing account by posting its id with the terms', async () => {
        await accountsApi.update(5, { ...base, convert: true })
        expect(calls[0]).toMatchObject({ method: 'post', url: '/credit-cards' })
        expect(calls[0].body).toEqual({ account_id: 5, ...terms })
    })

    it('updates a bank account on /accounts/{id}', async () => {
        await accountsApi.update(5, { name: 'Main', type: 'bank' })
        expect(calls[0]).toMatchObject({ method: 'patch', url: '/accounts/5' })
    })
})
