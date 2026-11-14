import axios from 'axios'
import { toast } from 'sonner'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { apiClient } from './client'

vi.mock('sonner', () => ({ toast: { error: vi.fn() } }))

const adapter = apiClient.defaults.adapter

afterEach(() => {
    apiClient.defaults.adapter = adapter
    vi.clearAllMocks()
})

describe('error interceptor', () => {
    it('does not toast a request canceled by the app (e.g. queries reset on space switch)', async () => {
        const controller = new AbortController()
        apiClient.defaults.adapter = async (config) => {
            throw new axios.CanceledError('canceled', config)
        }
        controller.abort()
        await expect(apiClient.get('/x', { signal: controller.signal })).rejects.toBeDefined()
        expect(toast.error).not.toHaveBeenCalled()
    })

    it('still toasts a real network failure', async () => {
        apiClient.defaults.adapter = async (config) => {
            throw new axios.AxiosError('Network Error', 'ERR_NETWORK', config)
        }
        await expect(apiClient.get('/x')).rejects.toBeDefined()
        expect(toast.error).toHaveBeenCalledTimes(1)
    })
})
