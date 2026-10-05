import { describe, expect, it } from 'vitest'

import { toQueryString } from './query-string'

describe('toQueryString', () => {
    it('returns an empty string when there is nothing to send', () => {
        expect(toQueryString()).toBe('')
        expect(toQueryString({})).toBe('')
        expect(toQueryString({ a: undefined, b: null, c: false, d: '' })).toBe('')
    })

    it('prefixes with ? and joins params', () => {
        expect(toQueryString({ page: 2, q: 'abc' })).toBe('?page=2&q=abc')
    })

    it('keeps zero, which is a real value', () => {
        expect(toQueryString({ offset: 0 })).toBe('?offset=0')
    })

    it('sends true as "true" and drops false', () => {
        expect(toQueryString({ pending: true, archived: false })).toBe('?pending=true')
    })

    it('repeats array params with a [] suffix', () => {
        expect(toQueryString({ tags: [1, 2], ids: ['a'] })).toBe(`?${encodeURIComponent('tags[]')}=1&${encodeURIComponent('tags[]')}=2&${encodeURIComponent('ids[]')}=a`)
    })

    it('sends no param for an empty array', () => {
        expect(toQueryString({ tags: [] })).toBe('')
    })

    it('escapes special characters', () => {
        expect(toQueryString({ q: 'a&b=c d' })).toBe('?q=a%26b%3Dc+d')
    })
})
