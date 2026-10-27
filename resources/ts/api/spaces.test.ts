import type { InternalAxiosRequestConfig } from 'axios'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { useSpaceStore } from '@/stores/space'
import { apiClient } from './client'
import { spacesApi } from './spaces'

let requested: string[] = []
const adapter = apiClient.defaults.adapter

beforeEach(() => {
    requested = []
    useSpaceStore.setState({ currentId: 7 })
    apiClient.defaults.adapter = async (config: InternalAxiosRequestConfig) => {
        requested.push(config.url ?? '')
        return { data: { data: [] }, status: 200, statusText: 'OK', headers: {}, config }
    }
})

afterEach(() => {
    apiClient.defaults.adapter = adapter
    useSpaceStore.setState({ currentId: null })
})

describe('account-wide catalog requests', () => {
    it('asks for the server-wide currency list, not the catalog of the current space', async () => {
        await spacesApi.currencyCatalog()
        // The space's own catalog hides the currencies the space already has.
        expect(requested).toEqual(['/catalog/currencies'])
    })

    it('keeps the presets out of the current space', async () => {
        await spacesApi.presets()
        expect(requested).toEqual(['/catalog/presets'])
    })
})
