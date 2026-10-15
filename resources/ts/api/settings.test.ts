import { beforeEach, describe, expect, it, vi } from 'vitest'

const get = vi.fn()
const spaceSettings = vi.fn()
let spaceId: number | null

vi.mock('./client', () => ({ api: { get: (url: string) => get(url), patch: vi.fn() } }))
vi.mock('./spaces', () => ({ spacesApi: { settings: (id: number) => spaceSettings(id), updateSettings: vi.fn() } }))
vi.mock('@/stores/space', () => ({ currentSpaceId: () => spaceId }))

import { settingsApi } from './settings'

beforeEach(() => {
    get.mockReset().mockResolvedValue({ onboarding_server_done: false })
    spaceSettings.mockReset()
    spaceId = 9
})

describe('settings api', () => {
    it('reads instance settings without asking for a space that may be gone', async () => {
        await expect(settingsApi.getInstance()).resolves.toEqual({ onboarding_server_done: false })
        expect(spaceSettings).not.toHaveBeenCalled()
    })
})
