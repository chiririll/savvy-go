import { describe, expect, it } from 'vitest'

import { spacePath } from './space-path'

describe('spacePath', () => {
    it('prefixes space data routes with the space', () => {
        expect(spacePath('/accounts', 3)).toBe('/spaces/3/accounts')
        expect(spacePath('/transactions/12', 3)).toBe('/spaces/3/transactions/12')
        expect(spacePath('/transactions?page=2', 3)).toBe('/spaces/3/transactions?page=2')
        expect(spacePath('/transactions-summary', 3)).toBe('/spaces/3/transactions-summary')
    })

    it('leaves other routes alone', () => {
        expect(spacePath('/auth/login', 3)).toBe('/auth/login')
        expect(spacePath('/spaces', 3)).toBe('/spaces')
        expect(spacePath('/', 3)).toBe('/')
    })

    it('does not prefix twice', () => {
        expect(spacePath('/spaces/3/accounts', 3)).toBe('/spaces/3/accounts')
        expect(spacePath('/spaces/9/accounts', 3)).toBe('/spaces/9/accounts')
    })

    it('leaves a path unchanged when no space is selected', () => {
        expect(spacePath('/accounts', null)).toBe('/accounts')
    })

    it('matches whole path segments only', () => {
        expect(spacePath('/accountsfoo', 3)).toBe('/accountsfoo')
        expect(spacePath('/tags-other', 3)).toBe('/tags-other')
    })

    it('leaves relative or absolute URLs alone', () => {
        expect(spacePath('accounts', 3)).toBe('accounts')
        expect(spacePath('https://example.com/accounts', 3)).toBe('https://example.com/accounts')
    })
})
