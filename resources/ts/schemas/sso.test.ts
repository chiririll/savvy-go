import { describe, expect, it } from 'vitest'

import { identityProviderSchema } from './sso'

const base = { name: 'GitHub', slug: 'github', preset: 'github' }

describe('identityProviderSchema', () => {
    // A preset field the admin never touched is still undefined in the form.
    it('accepts a field left untouched, as empty', () => {
        const parsed = identityProviderSchema.safeParse({ ...base, fields: { client_id: 'abc', scopes: undefined } })
        expect(parsed.success).toBe(true)
        expect(parsed.data?.fields).toEqual({ client_id: 'abc', scopes: '' })
    })

    it('keeps typed fields as they are', () => {
        const parsed = identityProviderSchema.safeParse({ ...base, fields: { client_id: 'abc' } })
        expect(parsed.data?.fields).toEqual({ client_id: 'abc' })
    })
})
