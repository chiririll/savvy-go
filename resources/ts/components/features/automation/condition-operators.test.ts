import { describe, expect, it } from 'vitest'

import { operatorsForField } from './condition-operators'

describe('operatorsForField', () => {
    it('is empty for an unknown field', () => {
        expect(operatorsForField('nope')).toEqual([])
    })

    it('expands the operator groups of a field into operators', () => {
        const amount = operatorsForField('amount')
        expect(amount).toContain('between')
        expect(amount).not.toContain('contains')
    })

    it('offers only real operators', () => {
        expect(operatorsForField('type').length).toBeGreaterThan(0)
        expect(operatorsForField('type').every((op) => typeof op === 'string')).toBe(true)
    })
})
