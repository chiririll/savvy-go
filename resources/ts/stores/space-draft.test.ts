import { describe, expect, it, vi } from 'vitest'
import { emptyDraft } from '@/components/features/spaces/create/draft'
import { sameShape } from '@/lib/draft-store'

vi.mock('@/stores/auth', () => ({ useAuthStore: { getState: () => ({ user: null }) } }))

import { loadSpaceDraft, spaceDraft } from './space-draft'

const store = (initial: Record<string, string> = {}) => {
    const data = { ...initial }
    vi.stubGlobal('localStorage', {
        getItem: (k: string) => data[k] ?? null,
        setItem: (k: string, v: string) => void (data[k] = v),
        removeItem: (k: string) => void delete data[k],
    })
}
const saved = (draft: object) => ({ [`${spaceDraft.key}:anonymous`]: JSON.stringify({ state: { draft, instance: '' } }) })

describe('space draft saved with the old entities step', () => {
    it('resumes on the currencies step with its content kept', async () => {
        store(saved({ ...emptyDraft(), step: 'entities', name: 'Trip', baseCurrency: 'EUR', tags: ['a'] }))
        await loadSpaceDraft()
        const draft = spaceDraft.useStore.getState().draft
        expect(draft).toMatchObject({ step: 'currencies', name: 'Trip', baseCurrency: 'EUR', tags: ['a'] })
    })

    it('keeps a step that exists, and drops a draft with an unknown one', async () => {
        store(saved({ ...emptyDraft(), step: 'tags', name: 'Trip' }))
        await loadSpaceDraft()
        expect(spaceDraft.useStore.getState().draft.step).toBe('tags')
        store(saved({ ...emptyDraft(), step: 'bogus', name: 'Trip' }))
        await loadSpaceDraft()
        expect(spaceDraft.useStore.getState().draft).toEqual(emptyDraft())
    })
})

describe('space draft', () => {
    it('is stored under its own key', () => {
        expect(spaceDraft.key).toBe('savvy-space-draft')
    })

    it('fits the shape of its own empty draft, filled', () => {
        const filled = JSON.parse(JSON.stringify({ ...emptyDraft(), baseCurrency: 'EUR', sourceId: 3 }))
        expect(sameShape(emptyDraft(), filled)).toBe(true)
    })

    it('does not fit a draft that still has the old invitation field', () => {
        expect(sameShape(emptyDraft(), { ...emptyDraft(), invitations: [] })).toBe(false)
    })
})
