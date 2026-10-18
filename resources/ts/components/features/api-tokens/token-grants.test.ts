import { describe, expect, it } from 'vitest'

import { expiryToIso, grantedResources } from './token-grants'

describe('expiryToIso', () => {
    it('never expires without a date', () => {
        expect(expiryToIso('never')).toBeNull()
    })

    it('counts days from the given moment', () => {
        const now = new Date('2026-01-01T10:00:00Z')
        expect(expiryToIso('30', now)).toBe('2026-01-31T10:00:00.000Z')
        expect(now.toISOString()).toBe('2026-01-01T10:00:00.000Z')
    })
})

describe('grantedResources', () => {
    it('is empty for a token with no permissions', () => {
        expect(grantedResources({ permissions: [] })).toEqual({ read: [], write: [], remove: [] })
    })

    it('files each resource under its highest level only', () => {
        const granted = grantedResources({
            permissions: ['accounts:read', 'accounts:write', 'tags:read'] as never,
        })
        expect(granted.write).toEqual(['accounts'])
        expect(granted.read).toEqual(['tags'])
        expect(granted.remove).toEqual([])
    })
})
