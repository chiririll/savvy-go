import { describe, expect, it } from 'vitest'

import { changedResources, parseOAuthRequest, redirectHost } from './consent'

const query = (init: Record<string, string>) => new URLSearchParams(init)

describe('parseOAuthRequest', () => {
    const base = { client_id: 'c', redirect_uri: 'https://app.example/cb', code_challenge: 'x' }

    it('reads the request, with the optional parts empty', () => {
        expect(parseOAuthRequest(query(base))).toEqual({
            client_id: 'c',
            redirect_uri: 'https://app.example/cb',
            state: '',
            code_challenge: 'x',
            code_challenge_method: '',
        })
    })

    it.each(['client_id', 'redirect_uri', 'code_challenge'])('rejects a request without %s', (key) => {
        expect(parseOAuthRequest(query({ ...base, [key]: '' }))).toBeNull()
    })
})

describe('redirectHost', () => {
    it('shows the host and port', () => {
        expect(redirectHost('https://claude.ai/api/mcp/callback')).toBe('claude.ai')
        expect(redirectHost('http://localhost:8080/cb')).toBe('localhost:8080')
    })

    it('is empty for anything that is not a URL', () => {
        expect(redirectHost('not a url')).toBe('')
        expect(redirectHost(undefined)).toBe('')
    })
})

describe('changedResources', () => {
    it('has nothing to confirm for read access only', () => {
        const { changes, deletes } = changedResources(['accounts'] as never, ['accounts:read'] as never)
        expect(changes).toEqual([])
        expect(deletes).toEqual([])
    })

    it('lists writes as changes and deletes in both', () => {
        const { changes, deletes } = changedResources(
            ['accounts', 'tags', 'budgets'] as never,
            ['accounts:read', 'accounts:write', 'tags:read', 'tags:write', 'tags:delete'] as never,
        )
        expect(changes).toEqual(['accounts', 'tags'])
        expect(deletes).toEqual(['tags'])
    })
})
