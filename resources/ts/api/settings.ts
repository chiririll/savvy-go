import { api } from './client'
import { spacesApi } from './spaces'
import { Settings } from '@/types'
import { currentSpaceId } from '@/stores/space'

const ENDPOINT = '/settings'

/** Settings kept per space; the rest are instance settings. */
const SPACE_KEYS: string[] = ['auto_update_currencies']

export const settingsApi = {
    /** Instance settings merged with those of the current space. */
    get: async (): Promise<Settings> => {
        const id = currentSpaceId()
        const [instance, space] = await Promise.all([
            api.get<Settings>(ENDPOINT),
            id === null ? Promise.resolve({}) : spacesApi.settings(id),
        ])
        return { ...instance, ...space } as Settings
    },

    update: async (data: Partial<Settings>): Promise<Settings> => {
        const id = currentSpaceId()
        const own: Record<string, unknown> = {}
        const instance: Record<string, unknown> = {}
        for (const [key, value] of Object.entries(data)) {
            if (SPACE_KEYS.includes(key)) own[key] = value
            else instance[key] = value
        }
        if (Object.keys(own).length && id !== null) await spacesApi.updateSettings(id, own)
        if (Object.keys(instance).length) await api.patch<Settings, Record<string, unknown>>(ENDPOINT, instance)
        return settingsApi.get()
    },
}
