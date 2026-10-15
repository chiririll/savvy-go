import { describe, expect, it, vi } from 'vitest'
import { emptyDraft } from '@/components/features/spaces/create/draft'
import { sameShape } from '@/lib/draft-store'

vi.mock('@/stores/auth', () => ({ useAuthStore: { getState: () => ({ user: null }) } }))

import { spaceDraft } from './space-draft'

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
