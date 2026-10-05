import { describe, expect, it } from 'vitest'

import { toggleIdInArray } from './ids'

describe('toggleIdInArray', () => {
    it('adds an id that is missing', () => {
        expect(toggleIdInArray([1, 2], 3)).toEqual([1, 2, 3])
        expect(toggleIdInArray([], 1)).toEqual([1])
    })

    it('removes an id that is present', () => {
        expect(toggleIdInArray([1, 2, 3], 2)).toEqual([1, 3])
    })

    it('does not mutate its input', () => {
        const ids = [1, 2]
        toggleIdInArray(ids, 3)
        toggleIdInArray(ids, 1)
        expect(ids).toEqual([1, 2])
    })
})
