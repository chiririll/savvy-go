import { beforeEach, describe, expect, it, vi } from 'vitest'

// Persisted stores read localStorage when they are created, so it has to exist
// before they are imported. (Node's own experimental localStorage is unusable here.)
const storage = vi.hoisted(() => {
    const data = new Map<string, string>()
    const fake = {
        getItem: (key: string) => data.get(key) ?? null,
        setItem: (key: string, value: string) => void data.set(key, value),
        removeItem: (key: string) => void data.delete(key),
        clear: () => data.clear(),
    }
    vi.stubGlobal('localStorage', fake)
    return fake
})

import { currentSpaceId, useSpaceStore } from './space'
import { useUiStore } from './ui'

beforeEach(() => {
    storage.clear()
    useSpaceStore.setState(useSpaceStore.getInitialState(), true)
    useUiStore.setState(useUiStore.getInitialState(), true)
})

describe('space store', () => {
    it('starts with no space selected', () => {
        expect(currentSpaceId()).toBeNull()
    })

    it('exposes the selected space outside React', () => {
        useSpaceStore.getState().setCurrentId(7)
        expect(currentSpaceId()).toBe(7)
        useSpaceStore.getState().setCurrentId(null)
        expect(currentSpaceId()).toBeNull()
    })

    it('remembers the choice in the browser under a stable key', () => {
        useSpaceStore.getState().setCurrentId(7)
        const saved = JSON.parse(storage.getItem('savvy-space')!)
        expect(saved.state).toEqual({ currentId: 7 })
    })

    it('restores the choice on the next visit', async () => {
        storage.setItem('savvy-space', JSON.stringify({ state: { currentId: 9 }, version: 0 }))
        await useSpaceStore.persist.rehydrate()
        expect(currentSpaceId()).toBe(9)
    })
})

describe('ui store', () => {
    it('opens the sidebar and keeps the menu sections closed by default', () => {
        expect(useUiStore.getState()).toMatchObject({ sidebarOpen: true, settingsOpen: false, adminOpen: false })
    })

    it('changes each flag independently', () => {
        useUiStore.getState().setSettingsOpen(true)
        useUiStore.getState().setSidebarOpen(false)
        expect(useUiStore.getState()).toMatchObject({ sidebarOpen: false, settingsOpen: true, adminOpen: false })
    })

    it('remembers the layout under a stable key', () => {
        useUiStore.getState().setAdminOpen(true)
        const saved = JSON.parse(storage.getItem('savvy-ui')!)
        expect(saved.state).toMatchObject({ adminOpen: true, sidebarOpen: true })
    })
})
