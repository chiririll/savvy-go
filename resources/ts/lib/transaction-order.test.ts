import { describe, expect, it } from 'vitest'

import { dayOrderForApi, placeInSlots } from './transaction-order'

const rows = (...ids: number[]) => ids.map((id) => ({ id }))

describe('placeInSlots', () => {
    it('refills the slots of the moved items and leaves the others', () => {
        expect(placeInSlots(rows(1, 2, 3, 4), [3, 1])).toEqual(rows(3, 2, 1, 4))
    })

    it('ignores ids that are not in the list', () => {
        expect(placeInSlots(rows(1, 2), [9, 2, 1])).toEqual(rows(2, 1))
    })
})

describe('dayOrderForApi', () => {
    it('sends the order oldest first', () => {
        expect(dayOrderForApi([3, 2, 1], 'desc')).toEqual([1, 2, 3])
        expect(dayOrderForApi([1, 2, 3], 'asc')).toEqual([1, 2, 3])
    })
})
