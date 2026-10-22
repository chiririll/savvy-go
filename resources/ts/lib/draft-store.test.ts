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

import { createDraftStore, sameShape } from './draft-store'

interface Fake {
    name: string
    tags: string[]
}
const empty = (): Fake => ({ name: '', tags: [] })
const store = createDraftStore({ key: 'test-draft', empty })
const key = `${store.key}:7`
const draft = () => store.useStore.getState().draft
const loadDraft = store.load

beforeEach(() => {
    storage.clear()
    auth.user = { id: 7 }
    store.useStore.setState({ draft: empty() })
})

describe('draft store', () => {
    it('is saved on every change, per user', () => {
        store.useStore.getState().update({ name: 'Home', tags: ['a'] })
        const saved = JSON.parse(storage.getItem(key)!)
        expect(saved.state.draft).toMatchObject({ name: 'Home', tags: ['a'] })
    })

    it('is restored when the modal opens', async () => {
        storage.setItem(
            key,
            JSON.stringify({ state: { draft: { ...empty(), name: 'Home', tags: ['a'] } }, version: 0 }),
        )
        await loadDraft()
        expect(draft()).toMatchObject({ name: 'Home', tags: ['a'] })
    })

    it('does not show the draft of another user', async () => {
        store.useStore.getState().update({ name: 'Home' })
        auth.user = { id: 8 }
        await loadDraft()
        expect(draft().name).toBe('')
    })

    it('is cleared after a successful create', () => {
        store.useStore.getState().update({ name: 'Home' })
        store.useStore.getState().clear()
        expect(draft().name).toBe('')
        expect(storage.getItem(key)).toBeNull()
    })

    it('drops a draft with other fields', async () => {
        storage.setItem(key, JSON.stringify({ state: { draft: { name: 'Old', tags: [], removed: 1 } }, version: 0 }))
        await loadDraft()
        expect(draft()).toEqual(empty())
        expect(JSON.parse(storage.getItem(key)!).state.draft).toEqual(empty())
    })

    it('drops a draft whose field is of another kind', async () => {
        storage.setItem(key, JSON.stringify({ state: { draft: { name: 'Old', tags: 'a' } }, version: 0 }))
        await loadDraft()
        expect(draft()).toEqual(empty())
    })

    it('drops a draft that the caller does not accept', async () => {
        const picky = createDraftStore({ key: 'test-draft', empty, accept: (d) => d.name !== 'Old' })
        storage.setItem(key, JSON.stringify({ state: { draft: { name: 'Old', tags: [] } }, version: 0 }))
        await picky.load()
        expect(picky.useStore.getState().draft).toEqual(empty())
    })

    it('ignores broken storage', async () => {
        storage.setItem(key, '{not json')
        await loadDraft()
        expect(draft()).toEqual(empty())
    })

    it('works without storage', async () => {
        const broken = () => {
            throw new Error('denied')
        }
        vi.stubGlobal('localStorage', { getItem: broken, setItem: broken, removeItem: broken })
        await loadDraft()
        store.useStore.getState().update({ name: 'Home' })
        expect(draft().name).toBe('Home')
        store.useStore.getState().clear()
        expect(draft().name).toBe('')
        vi.stubGlobal('localStorage', storage)
    })
})

describe('sameShape', () => {
    const base = { text: '', n: 0, on: false, list: [], pick: null, nested: { a: '' } }

    it('accepts the same fields of the same kinds, whatever the values', () => {
        expect(sameShape(base, { text: 'x', n: 5, on: true, list: [1], pick: 'EUR', nested: { a: 'y' } })).toBe(true)
    })

    it('takes anything where the default is null', () => {
        expect(sameShape(base, { ...base, pick: 3 })).toBe(true)
        expect(sameShape(base, { ...base, pick: null })).toBe(true)
    })

    it('refuses a missing, extra or differently typed field', () => {
        expect(sameShape(base, { n: 0, on: false, list: [], pick: null, nested: { a: '' } })).toBe(false)
        expect(sameShape(base, { ...base, extra: 1 })).toBe(false)
        expect(sameShape(base, { ...base, n: '5' })).toBe(false)
        expect(sameShape(base, { ...base, text: null })).toBe(false)
        expect(sameShape(base, { ...base, list: {} })).toBe(false)
        expect(sameShape(base, { ...base, nested: { b: '' } })).toBe(false)
    })

    it('refuses what is not a record', () => {
        expect(sameShape(base, null)).toBe(false)
        expect(sameShape(base, [])).toBe(false)
    })
})
