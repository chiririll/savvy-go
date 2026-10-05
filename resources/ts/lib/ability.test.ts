import { describe, expect, it } from 'vitest'

import { canWriteSpace, defineAbilityFor } from './ability'

describe('defineAbilityFor', () => {
    it('lets a server admin do everything, including managing users', () => {
        const ability = defineAbilityFor('admin')
        for (const action of ['create', 'read', 'update', 'delete', 'manage'] as const) {
            expect(ability.can(action, 'User')).toBe(true)
            expect(ability.can(action, 'all')).toBe(true)
        }
    })

    it.each(['user', 'guest'] as const)('lets a %s only read users', (role) => {
        const ability = defineAbilityFor(role)
        expect(ability.can('read', 'User')).toBe(true)
        for (const action of ['create', 'update', 'delete', 'manage'] as const) {
            expect(ability.can(action, 'User')).toBe(false)
        }
    })

    it.each(['user', 'guest'] as const)('leaves everything else to a %s', (role) => {
        const ability = defineAbilityFor(role)
        expect(ability.can('manage', 'all')).toBe(true)
    })

    it('gives nobody who is signed out any rights', () => {
        const ability = defineAbilityFor(null)
        expect(ability.can('read', 'User')).toBe(false)
        expect(ability.can('read', 'all')).toBe(false)
        expect(ability.can('manage', 'all')).toBe(false)
    })
})

describe('canWriteSpace', () => {
    it('allows admins and editors, not viewers', () => {
        expect(canWriteSpace('admin')).toBe(true)
        expect(canWriteSpace('editor')).toBe(true)
        expect(canWriteSpace('viewer')).toBe(false)
    })
})
