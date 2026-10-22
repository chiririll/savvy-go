import { describe, expect, it } from 'vitest'

import { numberDraft, numberToDraft } from './setting-drafts'

describe('numberDraft', () => {
    it('is unchanged while the field shows the saved value', () => {
        expect(numberDraft('5', 5)).toEqual({ parsed: 5, valid: true, changed: false })
    })

    it('accepts an empty field only where null is allowed', () => {
        expect(numberDraft('  ', 5, true)).toEqual({ parsed: null, valid: true, changed: true })
        expect(numberDraft('', 5)).toMatchObject({ valid: false })
    })

    it('rejects negative and non-numeric input', () => {
        expect(numberDraft('-1', 0).valid).toBe(false)
        expect(numberDraft('abc', 0).valid).toBe(false)
    })

    it('accepts zero and decimals', () => {
        expect(numberDraft('0', null).valid).toBe(true)
        expect(numberDraft('2.5', null).valid).toBe(true)
    })

    it('round-trips null through the field', () => {
        expect(numberToDraft(null)).toBe('')
        expect(numberToDraft(7)).toBe('7')
    })
})
