import { describe, expect, it } from 'vitest'

import { previewSplit } from './palette-preview'

describe('previewSplit', () => {
    it.each([
        [false, 0, { shown: 0, moreThan: 0 }],
        [false, 8, { shown: 8, moreThan: 0 }],
        [false, 13, { shown: 13, moreThan: 0 }], // everything fits: listed, no row
        [true, 13, { shown: 8, moreThan: 13 }], // more than 13 match: 8 and the row
    ])('has more %s, fetched %i', (hasMore, fetched, expected) => {
        expect(previewSplit(hasMore, fetched)).toEqual(expected)
    })
})
