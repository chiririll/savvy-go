import { describe, expect, it } from 'vitest'

import { safeReturnPath } from './return-path'

describe('safeReturnPath', () => {
    it('keeps a local path given as a string', () => {
        expect(safeReturnPath('/transactions?status=pending')).toBe('/transactions?status=pending')
    })

    it('joins the path and search of a location object', () => {
        expect(safeReturnPath({ pathname: '/budgets', search: '?edit=3' })).toBe('/budgets?edit=3')
        expect(safeReturnPath({ pathname: '/budgets' })).toBe('/budgets')
    })

    it.each(['//evil.example', 'https://evil.example', 'budgets'])('refuses %s', (from) => {
        expect(safeReturnPath(from)).toBe('/')
        expect(safeReturnPath({ pathname: from })).toBe('/')
    })

    it('goes home without a place to return to', () => {
        expect(safeReturnPath(undefined)).toBe('/')
        expect(safeReturnPath({})).toBe('/')
    })
})
