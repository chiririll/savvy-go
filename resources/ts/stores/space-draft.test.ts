import { beforeEach, describe, expect, it, vi } from 'vitest'

const storage = vi.hoisted(() => {
    const data = new Map<string, string>()
    const fake = {
        data,
        getItem: (key: string) => data.get(key) ?? null,
        setItem: (key: string, value: string) => void data.set(key, value),
        removeItem: (key: string) => void data.delete(key),
        clear: () => data.clear(),
    }
    vi.stubGlobal('localStorage', fake)
    return fake
})

const auth = vi.hoisted(() => ({ user: { id: 7 } as { id: number } | null }))
vi.mock('@/stores/auth', () => ({ useAuthStore: { getState: () => auth } }))

import { DRAFT_KEY, DRAFT_VERSION, loadSpaceDraft, useSpaceDraftStore } from './space-draft'
import { emptyDraft } from '@/components/features/spaces/create/draft'

const key = `${DRAFT_KEY}:7`
const draft = () => useSpaceDraftStore.getState().draft

beforeEach(() => {
    storage.clear()
    auth.user = { id: 7 }
    useSpaceDraftStore.setState({ draft: emptyDraft() })
})

describe('space draft', () => {
    it('is saved on every change, per user', () => {
        useSpaceDraftStore.getState().update({ name: 'Home', step: 'entities' })
        const saved = JSON.parse(storage.getItem(key)!)
        expect(saved.version).toBe(DRAFT_VERSION)
        expect(saved.state.draft).toMatchObject({ name: 'Home', step: 'entities' })
    })

    it('is restored when the modal opens', async () => {
        storage.setItem(key, JSON.stringify({ state: { draft: { ...emptyDraft(), name: 'Home', tags: ['a'] } }, version: DRAFT_VERSION }))
        await loadSpaceDraft()
        expect(draft()).toMatchObject({ name: 'Home', tags: ['a'] })
    })

    it('does not show the draft of another user', async () => {
        useSpaceDraftStore.getState().update({ name: 'Home' })
        auth.user = { id: 8 }
        await loadSpaceDraft()
        expect(draft().name).toBe('')
    })

    it('is cleared after a successful create', () => {
        useSpaceDraftStore.getState().update({ name: 'Home' })
        useSpaceDraftStore.getState().clear()
        expect(draft().name).toBe('')
        expect(storage.getItem(key)).toBeNull()
    })

    it('ignores a draft of another version', async () => {
        storage.setItem(key, JSON.stringify({ state: { draft: { ...emptyDraft(), name: 'Old' } }, version: DRAFT_VERSION - 1 }))
        await loadSpaceDraft()
        expect(draft().name).toBe('')
    })

    it('ignores broken storage', async () => {
        storage.setItem(key, '{not json')
        await loadSpaceDraft()
        expect(draft()).toEqual(emptyDraft())
    })

    it('works without storage', async () => {
        const broken = () => {
            throw new Error('denied')
        }
        vi.stubGlobal('localStorage', { getItem: broken, setItem: broken, removeItem: broken })
        await loadSpaceDraft()
        useSpaceDraftStore.getState().update({ name: 'Home' })
        expect(draft().name).toBe('Home')
        useSpaceDraftStore.getState().clear()
        expect(draft().name).toBe('')
        vi.stubGlobal('localStorage', storage)
    })
})
